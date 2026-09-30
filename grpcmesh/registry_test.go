package grpcmesh_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/testproto/shop"
	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

// served is an RPCService built from raw endpoints and subscribers.
type served struct {
	endpoints   []mesh.Endpoint
	subscribers []mesh.Subscriber
}

func (s served) Endpoints() []mesh.Endpoint     { return s.endpoints }
func (s served) Subscribers() []mesh.Subscriber { return s.subscribers }

func route(transport string, segments ...string) mesh.Target {
	return mesh.Target{Segments: segments, Kind: mesh.KindRoute, Metadata: map[string]string{grpcmesh.TransportKey: transport}}
}

func topic(transport string, segments ...string) mesh.Target {
	return mesh.Target{Segments: segments, Kind: mesh.KindTopic, Metadata: map[string]string{grpcmesh.TransportKey: transport}}
}

func orderService() shop.OrderService {
	return shop.OrderService{
		Place:  func(context.Context, *shop.Order) (*shop.Order, error) { return nil, nil },
		Placed: func(context.Context, *shop.Order) error { return nil },
	}
}

func TestRegisterAcceptsAGeneratedService(t *testing.T) {
	registry := grpcmesh.NewRegistry()

	registry.Register(orderService())

	endpoints, subscribers := registry.Endpoints(), registry.Subscribers()
	if len(endpoints) != 1 || !endpoints[0].Target.Equal(shop.OrderTargets.Place) {
		t.Errorf("Endpoints = %v, want Place", endpoints)
	}
	if len(subscribers) != 1 || !subscribers[0].Target.Equal(shop.OrderTargets.Placed) {
		t.Errorf("Subscribers = %v, want Placed", subscribers)
	}
	if got := subscribers[0].Metadata[mesh.ConsumerGroupKey]; got != "audit" {
		t.Errorf("Placed consumer_group = %q, want the generated audit", got)
	}
}

func TestRegistryReturnsEveryEndpointAndSubscriberRegistered(t *testing.T) {
	registry := grpcmesh.NewRegistry()
	registry.Register(served{
		endpoints:   []mesh.Endpoint{{Target: route("mem", "shop", "A", "Get")}, {Target: route("nats", "billing", "B", "Get")}},
		subscribers: []mesh.Subscriber{{Target: topic("mem", "shop", "A", "Made")}, {Target: topic("nats", "billing", "B", "Made")}},
	})

	if got := len(registry.Endpoints()); got != 2 {
		t.Errorf("Endpoints = %d, want 2", got)
	}
	if got := len(registry.Subscribers()); got != 2 {
		t.Errorf("Subscribers = %d, want 2", got)
	}
}

func TestWithConsumerGroupReplacesTheGeneratedGroupOfASubscriber(t *testing.T) {
	registry := grpcmesh.NewRegistry()

	registry.Register(orderService(), grpcmesh.WithConsumerGroup(shop.OrderTargets.Placed, "billing-ledger"))

	if got := registry.Subscribers()[0].Metadata[mesh.ConsumerGroupKey]; got != "billing-ledger" {
		t.Errorf("Placed consumer_group = %q, want billing-ledger", got)
	}
}

func TestWithConsumerGroupSetsTheGroupOfAnEndpoint(t *testing.T) {
	registry := grpcmesh.NewRegistry()

	registry.Register(orderService(), grpcmesh.WithConsumerGroup(shop.OrderTargets.Place, "orders"))

	if got := registry.Endpoints()[0].Metadata[mesh.ConsumerGroupKey]; got != "orders" {
		t.Errorf("Place consumer_group = %q, want orders", got)
	}
}

func TestWithConsumerGroupEmptyRemovesTheGeneratedGroup(t *testing.T) {
	registry := grpcmesh.NewRegistry()

	registry.Register(orderService(), grpcmesh.WithConsumerGroup(shop.OrderTargets.Placed, ""))

	if md := registry.Subscribers()[0].Metadata; md[mesh.ConsumerGroupKey] != "" {
		t.Errorf("Placed metadata = %v, want no consumer_group", md)
	}
}

func TestWithConsumerGroupLastValueForATargetWins(t *testing.T) {
	registry := grpcmesh.NewRegistry()

	registry.Register(orderService(),
		grpcmesh.WithConsumerGroup(shop.OrderTargets.Placed, "first"),
		grpcmesh.WithConsumerGroup(shop.OrderTargets.Placed, "second"))

	if got := registry.Subscribers()[0].Metadata[mesh.ConsumerGroupKey]; got != "second" {
		t.Errorf("Placed consumer_group = %q, want second", got)
	}
}

func TestWithConsumerGroupPanicsForATargetOutsideTheService(t *testing.T) {
	registry := grpcmesh.NewRegistry()
	defer func() {
		p := recover()
		if msg, _ := p.(string); !strings.Contains(msg, "[billing Invoice Made]") {
			t.Errorf("panic = %v, want one naming the target", p)
		}
		if len(registry.Endpoints()) != 0 || len(registry.Subscribers()) != 0 {
			t.Error("the service was registered despite the panic")
		}
	}()

	registry.Register(orderService(), grpcmesh.WithConsumerGroup(topic("mem", "billing", "Invoice", "Made"), "x"))
}
