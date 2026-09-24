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
	svc := testproto.ApiKeyService{
		Search:  func(context.Context, *testproto.ApiKey) (*testproto.ApiKey, error) { return nil, nil },
		Created: func(context.Context, *testproto.ApiKey) error { return nil },
	}

	registry.Register(svc)

	endpoints, subscribers := registry.Endpoints("testproto"), registry.Subscribers("testproto")
	if len(endpoints) != 1 || !endpoints[0].Target.Equal(testproto.ApiKeyTargets.Search) {
		t.Errorf("Endpoints = %v, want Search", endpoints)
	}
	if len(subscribers) != 1 || !subscribers[0].Target.Equal(testproto.ApiKeyTargets.Created) {
		t.Errorf("Subscribers = %v, want Created", subscribers)
	}
}

func TestRegistryFiltersBindingsByDeploymentGroup(t *testing.T) {
	registry := grpcmesh.NewRegistry()
	registry.Register(bindings{
		endpoints:   []mesh.Endpoint{{Target: route("pbx", "pbx", "A", "Get")}, {Target: route("billing", "billing", "B", "Get")}},
		subscribers: []mesh.Subscriber{{Target: topic("pbx", "pbx", "A", "Made")}, {Target: topic("billing", "billing", "B", "Made")}},
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
