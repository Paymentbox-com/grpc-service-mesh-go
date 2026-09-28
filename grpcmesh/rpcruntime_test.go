package grpcmesh_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/memtransport"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/testproto"
	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

func TestNewRPCRuntimePassesTheGroupsBindingsAndConfigToNewRuntime(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	grpcmesh.Register(bindings{
		endpoints:   []mesh.Endpoint{{Target: route("testproto", "testproto", "A", "Get")}, {Target: route("other", "other", "B", "Get")}},
		subscribers: []mesh.Subscriber{{Target: topic("testproto", "testproto", "A", "Made")}, {Target: topic("other", "other", "B", "Made")}},
	})

	rt, err := grpcmesh.NewRPCRuntime("mem", "testproto", memConfig, hub.NewRuntime)
	if err != nil {
		t.Fatal(err)
	}

	built := hub.Runtimes()
	if len(built) != 1 {
		t.Fatalf("newRuntime ran %d times, want 1", len(built))
	}
	if rt.Underlying() != built[0] {
		t.Error("Underlying() is not the runtime newRuntime built")
	}
	if built[0].Config[mesh.DeploymentGroupKey] != "testproto" || built[0].Config["url"] != memConfig["url"] {
		t.Errorf("runtime config = %v, want the given config plus deployment_group", built[0].Config)
	}
	if len(built[0].Endpoints) != 1 || built[0].Endpoints[0].Target.Segments[0] != "testproto" {
		t.Errorf("runtime endpoints = %v, want only the testproto one", built[0].Endpoints)
	}
	if len(built[0].Subscribers) != 1 || built[0].Subscribers[0].Target.Segments[0] != "testproto" {
		t.Errorf("runtime subscribers = %v, want only the testproto one", built[0].Subscribers)
	}
	if rt.Transport() != "mem" || rt.DeploymentGroup() != "testproto" {
		t.Errorf("Transport() = %q, DeploymentGroup() = %q", rt.Transport(), rt.DeploymentGroup())
	}
}

func TestNewRPCRuntimePassesTheRoutersClientToNewRuntime(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	added := memClient(t, hub)
	grpcmesh.AddTransport("mem", added)

	rt, err := grpcmesh.NewRPCRuntime("mem", "testproto", memConfig, hub.NewRuntime)
	if err != nil {
		t.Fatal(err)
	}

	if hub.Runtimes()[0].Client() != added {
		t.Error("newRuntime received a client other than the router's")
	}
	if rt.Client() != added {
		t.Error("Client() is not the router's client")
	}
}

func TestNewRPCRuntimeDeploymentGroupOverridesTheGivenConfig(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	cfg := mesh.Config{mesh.DeploymentGroupKey: "configured"}

	if _, err := grpcmesh.NewRPCRuntime("mem", "testproto", cfg, hub.NewRuntime); err != nil {
		t.Fatal(err)
	}

	if got := hub.Runtimes()[0].Config[mesh.DeploymentGroupKey]; got != "testproto" {
		t.Errorf("deployment_group = %q, want testproto", got)
	}
	if cfg[mesh.DeploymentGroupKey] != "configured" {
		t.Error("the given config was mutated")
	}
}

func TestNewRPCRuntimeUnknownTransport(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()

	_, err := grpcmesh.NewRPCRuntime("mem", "testproto", memConfig, hub.NewRuntime)

	if !errors.Is(err, grpcmesh.ErrUnknownTransport) {
		t.Errorf("err = %v, want ErrUnknownTransport", err)
	}
	if len(hub.Runtimes()) != 0 {
		t.Error("newRuntime ran for an unknown transport")
	}
}

func TestNewRPCRuntimePassesTheNewRuntimeErrorThrough(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	hub.Fail = errors.New("bad url")

	_, err := grpcmesh.NewRPCRuntime("mem", "testproto", memConfig, hub.NewRuntime)

	if err != hub.Fail {
		t.Errorf("err = %v, want newRuntime's error unchanged", err)
	}
}

func TestRPCRuntimeDelegatesTheLifecycle(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	rt, err := grpcmesh.NewRPCRuntime("mem", "testproto", memConfig, hub.NewRuntime)
	if err != nil {
		t.Fatal(err)
	}
	underlying := hub.Runtimes()[0]

	if rt.Running() || underlying.Running() {
		t.Error("running before Start")
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !rt.Running() || !underlying.Running() {
		t.Error("not running after Start")
	}
	if err := rt.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if rt.Running() || underlying.Running() {
		t.Error("running after Stop")
	}
}

func TestRPCRuntimeStopLeavesTheClientClosed(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	added := memClient(t, hub)
	grpcmesh.AddTransport("mem", added)
	rt, err := grpcmesh.NewRPCRuntime("mem", "testproto", memConfig, hub.NewRuntime)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	if err := rt.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !added.Closed() {
		t.Error("the router's client is open after Stop")
	}
}

func TestRPCRuntimeServesAGeneratedService(t *testing.T) {
	var got *testproto.Order
	serveMem(t, testproto.OrderService{
		Place: func(_ context.Context, req *testproto.Order) (*testproto.Order, error) {
			got = req
			return order("reply"), nil
		},
	})

	resp, err := testproto.OrderClient.Place(context.Background(), order("ask"))
	if err != nil {
		t.Fatal(err)
	}

	if got.GetId() != "ask" {
		t.Errorf("handler received %v", got)
	}
	if resp.GetId() != "reply" {
		t.Errorf("caller received %v", resp)
	}
}

func TestNewRPCRuntimeWithoutARuntimeConstructor(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))

	_, err := grpcmesh.NewRPCRuntime("mem", "testproto", memConfig, nil)

	if !errors.Is(err, grpcmesh.ErrNoRuntimeConstructor) {
		t.Errorf("err = %v, want ErrNoRuntimeConstructor", err)
	}
}

// shopTarget returns a Target in deployment group shop over transport mem,
// the RPCRuntime's in the override tests.
func shopTarget(kind mesh.Kind, segments ...string) mesh.Target {
	return mesh.Target{Segments: segments, Kind: kind, Metadata: map[string]string{mesh.DeploymentGroupKey: "shop", grpcmesh.TransportKey: "mem"}}
}

func TestNewRPCRuntimeWithEndpointsReplacesTheRegistrysEndpoints(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	grpcmesh.Register(bindings{
		endpoints:   []mesh.Endpoint{{Target: shopTarget(mesh.KindRoute, "shop", "Registered", "Get")}},
		subscribers: []mesh.Subscriber{{Target: shopTarget(mesh.KindTopic, "shop", "Registered", "Made")}},
	})

	_, err := grpcmesh.NewRPCRuntime("mem", "shop", memConfig, hub.NewRuntime,
		grpcmesh.WithEndpoints(mesh.Endpoint{Target: shopTarget(mesh.KindRoute, "shop", "Given", "Get")}))
	if err != nil {
		t.Fatal(err)
	}

	built := hub.Runtimes()[0]
	if len(built.Endpoints) != 1 || built.Endpoints[0].Target.Segments[1] != "Given" {
		t.Errorf("runtime endpoints = %v, want only the given one", built.Endpoints)
	}
	if len(built.Subscribers) != 1 || built.Subscribers[0].Target.Segments[1] != "Registered" {
		t.Errorf("runtime subscribers = %v, want the registered one", built.Subscribers)
	}
}

func TestNewRPCRuntimeWithSubscribersReplacesTheRegistrysSubscribers(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	grpcmesh.Register(bindings{
		endpoints:   []mesh.Endpoint{{Target: shopTarget(mesh.KindRoute, "shop", "Registered", "Get")}},
		subscribers: []mesh.Subscriber{{Target: shopTarget(mesh.KindTopic, "shop", "Registered", "Made")}},
	})

	_, err := grpcmesh.NewRPCRuntime("mem", "shop", memConfig, hub.NewRuntime,
		grpcmesh.WithSubscribers(mesh.Subscriber{Target: shopTarget(mesh.KindTopic, "shop", "Given", "Made")}))
	if err != nil {
		t.Fatal(err)
	}

	built := hub.Runtimes()[0]
	if len(built.Endpoints) != 1 || built.Endpoints[0].Target.Segments[1] != "Registered" {
		t.Errorf("runtime endpoints = %v, want the registered one", built.Endpoints)
	}
	if len(built.Subscribers) != 1 || built.Subscribers[0].Target.Segments[1] != "Given" {
		t.Errorf("runtime subscribers = %v, want only the given one", built.Subscribers)
	}
}

func TestNewRPCRuntimeWithEndpointsAndSubscribersReplacesBoth(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	grpcmesh.Register(bindings{
		endpoints:   []mesh.Endpoint{{Target: shopTarget(mesh.KindRoute, "shop", "Registered", "Get")}},
		subscribers: []mesh.Subscriber{{Target: shopTarget(mesh.KindTopic, "shop", "Registered", "Made")}},
	})

	_, err := grpcmesh.NewRPCRuntime("mem", "shop", memConfig, hub.NewRuntime,
		grpcmesh.WithEndpoints(mesh.Endpoint{Target: shopTarget(mesh.KindRoute, "shop", "Given", "Get")}),
		grpcmesh.WithSubscribers(mesh.Subscriber{Target: shopTarget(mesh.KindTopic, "shop", "Given", "Made")}))
	if err != nil {
		t.Fatal(err)
	}

	built := hub.Runtimes()[0]
	if len(built.Endpoints) != 1 || built.Endpoints[0].Target.Segments[1] != "Given" {
		t.Errorf("runtime endpoints = %v, want only the given one", built.Endpoints)
	}
	if len(built.Subscribers) != 1 || built.Subscribers[0].Target.Segments[1] != "Given" {
		t.Errorf("runtime subscribers = %v, want only the given one", built.Subscribers)
	}
}

func TestNewRPCRuntimeWithNoEndpointsServesNone(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	grpcmesh.Register(bindings{
		endpoints:   []mesh.Endpoint{{Target: shopTarget(mesh.KindRoute, "shop", "Registered", "Get")}},
		subscribers: []mesh.Subscriber{{Target: shopTarget(mesh.KindTopic, "shop", "Registered", "Made")}},
	})

	_, err := grpcmesh.NewRPCRuntime("mem", "shop", memConfig, hub.NewRuntime, grpcmesh.WithEndpoints())
	if err != nil {
		t.Fatal(err)
	}

	built := hub.Runtimes()[0]
	if len(built.Endpoints) != 0 {
		t.Errorf("runtime endpoints = %v, want none", built.Endpoints)
	}
	if len(built.Subscribers) != 1 || built.Subscribers[0].Target.Segments[1] != "Registered" {
		t.Errorf("runtime subscribers = %v, want the registered one", built.Subscribers)
	}
}

func TestNewRPCRuntimeRejectsAGivenTargetInAnotherDeploymentGroup(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	target := shopTarget(mesh.KindRoute, "billing", "Invoice", "Get")
	target.Metadata[mesh.DeploymentGroupKey] = "billing"

	_, err := grpcmesh.NewRPCRuntime("mem", "shop", memConfig, hub.NewRuntime, grpcmesh.WithEndpoints(mesh.Endpoint{Target: target}))

	if !errors.Is(err, grpcmesh.ErrTargetOutsideRuntime) {
		t.Fatalf("err = %v, want ErrTargetOutsideRuntime", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "[billing Invoice Get]") || !strings.Contains(msg, `deployment_group "billing"`) {
		t.Errorf("err = %q, want the segments and deployment_group named", msg)
	}
	if len(hub.Runtimes()) != 0 {
		t.Error("newRuntime ran for a target outside the runtime")
	}
}

func TestNewRPCRuntimeRejectsAGivenTargetOnAnotherTransport(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	target := shopTarget(mesh.KindTopic, "shop", "Order", "Made")
	target.Metadata[grpcmesh.TransportKey] = "nats"

	_, err := grpcmesh.NewRPCRuntime("mem", "shop", memConfig, hub.NewRuntime, grpcmesh.WithSubscribers(mesh.Subscriber{Target: target}))

	if !errors.Is(err, grpcmesh.ErrTargetOutsideRuntime) {
		t.Fatalf("err = %v, want ErrTargetOutsideRuntime", err)
	}
	if msg := err.Error(); !strings.Contains(msg, "[shop Order Made]") || !strings.Contains(msg, `transport "nats"`) {
		t.Errorf("err = %q, want the segments and transport named", msg)
	}
	if len(hub.Runtimes()) != 0 {
		t.Error("newRuntime ran for a target outside the runtime")
	}
}
