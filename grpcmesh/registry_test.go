package grpcmesh_test

import (
	"context"
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/testproto"
	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

// bindings is an RPCService built from raw bindings.
type bindings struct {
	endpoints   []mesh.Endpoint
	subscribers []mesh.Subscriber
}

func (b bindings) Endpoints() []mesh.Endpoint     { return b.endpoints }
func (b bindings) Subscribers() []mesh.Subscriber { return b.subscribers }

func route(group string, segments ...string) mesh.Target {
	return mesh.Target{Segments: segments, Kind: mesh.KindRoute, Metadata: map[string]string{mesh.DeploymentGroupKey: group}}
}

func topic(group string, segments ...string) mesh.Target {
	return mesh.Target{Segments: segments, Kind: mesh.KindTopic, Metadata: map[string]string{mesh.DeploymentGroupKey: group}}
}

func TestRegisterAcceptsAGeneratedService(t *testing.T) {
	registry := grpcmesh.NewRegistry()
	svc := testproto.OrderService{
		Place:  func(context.Context, *testproto.Order) (*testproto.Order, error) { return nil, nil },
		Placed: func(context.Context, *testproto.Order) error { return nil },
	}

	registry.Register(svc)

	endpoints, subscribers := registry.Endpoints("shop"), registry.Subscribers("shop")
	if len(endpoints) != 1 || !endpoints[0].Target.Equal(testproto.OrderTargets.Place) {
		t.Errorf("Endpoints = %v, want Place", endpoints)
	}
	if len(subscribers) != 1 || !subscribers[0].Target.Equal(testproto.OrderTargets.Placed) {
		t.Errorf("Subscribers = %v, want Placed", subscribers)
	}
}

func TestRegistryFiltersBindingsByDeploymentGroup(t *testing.T) {
	registry := grpcmesh.NewRegistry()
	registry.Register(bindings{
		endpoints:   []mesh.Endpoint{{Target: route("shop", "shop", "A", "Get")}, {Target: route("billing", "billing", "B", "Get")}},
		subscribers: []mesh.Subscriber{{Target: topic("shop", "shop", "A", "Made")}, {Target: topic("billing", "billing", "B", "Made")}},
	})

	endpoints, subscribers := registry.Endpoints("billing"), registry.Subscribers("billing")

	if len(endpoints) != 1 || endpoints[0].Target.Segments[0] != "billing" {
		t.Errorf("Endpoints(billing) = %v", endpoints)
	}
	if len(subscribers) != 1 || subscribers[0].Target.Segments[0] != "billing" {
		t.Errorf("Subscribers(billing) = %v", subscribers)
	}
	if len(registry.Endpoints("audit")) != 0 || len(registry.Subscribers("audit")) != 0 {
		t.Error("an unregistered group has bindings")
	}
}
