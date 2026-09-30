package grpcmesh_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/memtransport"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/testproto/shop"
	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

func TestNewRPCRuntimePassesItsTransportsEndpointsSubscribersAndConfigToNewRuntime(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	grpcmesh.Register(served{
		endpoints:   []mesh.Endpoint{{Target: route("mem", "shop", "A", "Get")}, {Target: route("nats", "billing", "B", "Get")}},
		subscribers: []mesh.Subscriber{{Target: topic("mem", "shop", "A", "Made")}, {Target: topic("nats", "billing", "B", "Made")}},
	})

	rt, err := grpcmesh.NewRPCRuntime("mem", "billing", memConfig, hub.NewRuntime)
	if err != nil {
		t.Fatal(err)
	}

	built := hub.Runtimes()
	if len(built) != 1 {
		t.Fatalf("newRuntime ran %d times, want 1", len(built))
	}
	if rt.TRuntime() != built[0] {
		t.Error("TRuntime() is not the runtime newRuntime built")
	}
	if built[0].Config[mesh.DeploymentGroupKey] != "billing" || built[0].Config["url"] != memConfig["url"] {
		t.Errorf("runtime config = %v, want the given config plus deployment_group", built[0].Config)
	}
	if len(built[0].Endpoints) != 1 || built[0].Endpoints[0].Target.Segments[0] != "shop" {
		t.Errorf("runtime endpoints = %v, want only the mem one, whatever its namespace", built[0].Endpoints)
	}
	if len(built[0].Subscribers) != 1 || built[0].Subscribers[0].Target.Segments[0] != "shop" {
		t.Errorf("runtime subscribers = %v, want only the mem one, whatever its namespace", built[0].Subscribers)
	}
	if rt.Transport() != "mem" || rt.DeploymentGroup() != "billing" {
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
	var got *shop.Order
	serveMem(t, shop.OrderService{
		Place: func(_ context.Context, req *shop.Order) (*shop.Order, error) {
			got = req
			return order("reply"), nil
		},
	})

	resp, _, err := shop.OrderClient.Place(context.Background(), order("ask"), nil)
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
