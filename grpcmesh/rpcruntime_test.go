package grpcmesh_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/memtransport"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/testproto"
	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

func TestNewRPCRuntimeHandsTheTransportItsBindingsMapAndConfig(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memTransport(hub))
	if err := grpcmesh.Register(bindings{
		endpoints:   []mesh.Endpoint{{Target: route("testproto", "testproto", "A", "Get")}, {Target: route("other", "other", "B", "Get")}},
		subscribers: []mesh.Subscriber{{Target: topic("testproto", "testproto", "A", "Made")}, {Target: topic("other", "other", "B", "Made")}},
	}); err != nil {
		t.Fatal(err)
	}

	rt, err := grpcmesh.NewRPCRuntime("mem", "testproto")
	if err != nil {
		t.Fatal(err)
	}

	built := hub.Runtimes()
	if len(built) != 1 {
		t.Fatalf("NewRuntime ran %d times, want 1", len(built))
	}
	if rt.Underlying() != built[0] {
		t.Error("Underlying() is not the runtime the transport built")
	}
	if built[0].Config[mesh.DeploymentGroupKey] != "testproto" || built[0].Config["url"] != memConfig["url"] {
		t.Errorf("runtime config = %v, want the entry config plus deployment_group", built[0].Config)
	}
	if len(built[0].ServiceMap.Targets) != len(memServiceMap.Targets) {
		t.Errorf("runtime service map = %v, want the entry's", built[0].ServiceMap)
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

func TestNewRPCRuntimeDeploymentGroupOverridesTheEntryConfig(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	entry := memTransport(hub)
	entry.Config = mesh.Config{mesh.DeploymentGroupKey: "configured"}
	grpcmesh.AddTransport("mem", entry)

	if _, err := grpcmesh.NewRPCRuntime("mem", "testproto"); err != nil {
		t.Fatal(err)
	}

	if got := hub.Runtimes()[0].Config[mesh.DeploymentGroupKey]; got != "testproto" {
		t.Errorf("deployment_group = %q, want testproto", got)
	}
	if entry.Config[mesh.DeploymentGroupKey] != "configured" {
		t.Error("the entry's config was mutated")
	}
}

func TestNewRPCRuntimeUnknownTransport(t *testing.T) {
	freshSingletons(t)

	_, err := grpcmesh.NewRPCRuntime("mem", "testproto")

	if !errors.Is(err, grpcmesh.ErrUnknownTransport) {
		t.Errorf("err = %v, want ErrUnknownTransport", err)
	}
}

func TestNewRPCRuntimeRefusesASecondRuntimeForTheTransport(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memTransport(hub))
	if _, err := grpcmesh.NewRPCRuntime("mem", "testproto"); err != nil {
		t.Fatal(err)
	}

	_, err := grpcmesh.NewRPCRuntime("mem", "other")

	if !errors.Is(err, grpcmesh.ErrRuntimeExists) {
		t.Errorf("err = %v, want ErrRuntimeExists", err)
	}
	if len(hub.Runtimes()) != 1 {
		t.Errorf("NewRuntime ran %d times, want 1", len(hub.Runtimes()))
	}
}

func TestNewRPCRuntimePropagatesTheTransportConstructorError(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	hub.Fail = errors.New("bad url")
	grpcmesh.AddTransport("mem", memTransport(hub))

	_, err := grpcmesh.NewRPCRuntime("mem", "testproto")

	if !errors.Is(err, hub.Fail) {
		t.Errorf("err = %v, want the constructor's error", err)
	}
}

func TestRPCRuntimeDelegatesTheLifecycle(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memTransport(hub))
	rt, err := grpcmesh.NewRPCRuntime("mem", "testproto")
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
	if rt.Client() != underlying.Client() {
		t.Error("Client() is not the underlying runtime's client")
	}
	if err := rt.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if rt.Running() || underlying.Running() {
		t.Error("running after Stop")
	}
}

func TestRPCRuntimeServesAGeneratedService(t *testing.T) {
	var got *testproto.ApiKey
	serveMem(t, testproto.ApiKeyService{
		Search: func(_ context.Context, req *testproto.ApiKey) (*testproto.ApiKey, error) {
			got = req
			return apiKey("reply"), nil
		},
	})

	resp, err := testproto.ApiKeyClient.Search(context.Background(), apiKey("ask"))
	if err != nil {
		t.Fatal(err)
	}

	if got.GetFirstName() != "ask" {
		t.Errorf("handler received %v", got)
	}
	if resp.GetFirstName() != "reply" {
		t.Errorf("caller received %v", resp)
	}
}
