package grpcmesh_test

import (
	"context"
	"errors"
	"strings"
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

	if err := registry.Register(svc); err != nil {
		t.Fatal(err)
	}

	endpoints, subscribers := registry.Endpoints("testproto"), registry.Subscribers("testproto")
	if len(endpoints) != 1 || !endpoints[0].Target.Equal(testproto.ApiKeyTargets.Search) {
		t.Errorf("Endpoints = %v, want Search", endpoints)
	}
	if len(subscribers) != 1 || !subscribers[0].Target.Equal(testproto.ApiKeyTargets.Created) {
		t.Errorf("Subscribers = %v, want Created", subscribers)
	}
}

func TestRegisterRejectsATargetRegisteredEarlierAndAddsNothing(t *testing.T) {
	registry := grpcmesh.NewRegistry()
	search := route("pbx", "pbx", "ApiKeyService", "Search")
	if err := registry.Register(bindings{endpoints: []mesh.Endpoint{{Target: search}}}); err != nil {
		t.Fatal(err)
	}

	err := registry.Register(bindings{
		endpoints:   []mesh.Endpoint{{Target: search}},
		subscribers: []mesh.Subscriber{{Target: topic("pbx", "pbx", "ApiKeyService", "Created")}},
	})

	if !errors.Is(err, grpcmesh.ErrDuplicateTarget) {
		t.Fatalf("err = %v, want ErrDuplicateTarget", err)
	}
	if !strings.Contains(err.Error(), "pbx.ApiKeyService.Search") {
		t.Errorf("err = %q, want the target in it", err)
	}
	if len(registry.Subscribers("pbx")) != 0 {
		t.Error("the rejected service's other binding was added")
	}
}

func TestRegisterRejectsATargetAppearingTwiceInOneService(t *testing.T) {
	registry := grpcmesh.NewRegistry()
	created := topic("pbx", "pbx", "ApiKeyService", "Created")

	err := registry.Register(bindings{subscribers: []mesh.Subscriber{{Target: created}, {Target: created}}})

	if !errors.Is(err, grpcmesh.ErrDuplicateTarget) {
		t.Fatalf("err = %v, want ErrDuplicateTarget", err)
	}
	if len(registry.Subscribers("pbx")) != 0 {
		t.Error("bindings were added")
	}
}

func TestRegisterAllowsTheSameSegmentsWithAnotherKind(t *testing.T) {
	registry := grpcmesh.NewRegistry()

	err := registry.Register(bindings{
		endpoints:   []mesh.Endpoint{{Target: route("pbx", "pbx", "Svc", "M")}},
		subscribers: []mesh.Subscriber{{Target: topic("pbx", "pbx", "Svc", "M")}},
	})

	if err != nil {
		t.Fatal(err)
	}
}

func TestRegistryFiltersBindingsByDeploymentGroup(t *testing.T) {
	registry := grpcmesh.NewRegistry()
	if err := registry.Register(bindings{
		endpoints:   []mesh.Endpoint{{Target: route("pbx", "pbx", "A", "Get")}, {Target: route("billing", "billing", "B", "Get")}},
		subscribers: []mesh.Subscriber{{Target: topic("pbx", "pbx", "A", "Made")}, {Target: topic("billing", "billing", "B", "Made")}},
	}); err != nil {
		t.Fatal(err)
	}

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
