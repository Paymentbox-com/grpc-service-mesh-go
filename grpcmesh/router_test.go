package grpcmesh_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/memtransport"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/testproto"
	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

func TestRouterClientUnknownTransport(t *testing.T) {
	router := grpcmesh.NewTransportRouter()

	_, err := router.Client("http")

	if !errors.Is(err, grpcmesh.ErrUnknownTransport) {
		t.Fatalf("err = %v, want ErrUnknownTransport", err)
	}
	if !strings.Contains(err.Error(), "http") {
		t.Errorf("err = %q, want the transport name in it", err)
	}
}

func TestRouterGetUnknownTransport(t *testing.T) {
	router := grpcmesh.NewTransportRouter()

	_, err := router.Get("http")

	if !errors.Is(err, grpcmesh.ErrUnknownTransport) {
		t.Fatalf("err = %v, want ErrUnknownTransport", err)
	}
}

func TestRouterGetReturnsTheEntry(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	hub := memtransport.NewHub()
	router.AddTransport("mem", memTransport(hub))

	entry, err := router.Get("mem")
	if err != nil {
		t.Fatal(err)
	}

	if entry.Config["url"] != memConfig["url"] || len(entry.ServiceMap.Targets) != len(memServiceMap.Targets) {
		t.Errorf("Get returned %+v", entry)
	}
}

func TestRouterClientBuildsOneStandaloneClientFromTheEntry(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	hub := memtransport.NewHub()
	router.AddTransport("mem", memTransport(hub))

	first, err := router.Client("mem")
	if err != nil {
		t.Fatal(err)
	}
	second, err := router.Client("mem")
	if err != nil {
		t.Fatal(err)
	}

	if first != second {
		t.Error("two calls returned different clients")
	}
	clients := hub.Clients()
	if len(clients) != 1 {
		t.Fatalf("NewClient ran %d times, want 1", len(clients))
	}
	if clients[0].Config["url"] != memConfig["url"] {
		t.Errorf("client built with config %v", clients[0].Config)
	}
	if len(clients[0].ServiceMap.Targets) != len(memServiceMap.Targets) {
		t.Errorf("client built with service map %v", clients[0].ServiceMap)
	}
	if clients[0].Owner != nil {
		t.Error("client is runtime-owned, want standalone")
	}
}

func TestRouterClientPropagatesNewClientError(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	hub := memtransport.NewHub()
	hub.Fail = errors.New("connect refused")
	router.AddTransport("mem", memTransport(hub))

	_, err := router.Client("mem")

	if !errors.Is(err, hub.Fail) {
		t.Errorf("err = %v, want the constructor's error", err)
	}
}

func TestRouterClientSwitchesToTheRuntimeClientOnceAnRPCRuntimeExists(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memTransport(hub))

	standalone, err := grpcmesh.DefaultTransportRouter.Client("mem")
	if err != nil {
		t.Fatal(err)
	}
	rt, err := grpcmesh.NewRPCRuntime("mem", "testproto")
	if err != nil {
		t.Fatal(err)
	}
	owned, err := grpcmesh.DefaultTransportRouter.Client("mem")
	if err != nil {
		t.Fatal(err)
	}

	if owned == standalone {
		t.Error("router still returns the standalone client")
	}
	if owned != rt.Underlying().Client() {
		t.Error("router does not return the runtime's client")
	}
}

func TestAddTransportAgainReplacesTheEntry(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	first, second := memtransport.NewHub(), memtransport.NewHub()
	router.AddTransport("mem", memTransport(first))
	if _, err := router.Client("mem"); err != nil {
		t.Fatal(err)
	}

	router.AddTransport("mem", grpcmesh.Transport{
		Config:     mesh.Config{"url": "mem://second"},
		ServiceMap: mesh.ServiceMap{Targets: []mesh.Target{testproto.ApiKeyTargets.Search}},
		NewRuntime: second.NewRuntime,
		NewClient:  second.NewClient,
	})
	c, err := router.Client("mem")
	if err != nil {
		t.Fatal(err)
	}

	if len(second.Clients()) != 1 || c != second.Clients()[0] {
		t.Error("client does not come from the replacement entry")
	}
	if len(first.Clients()) != 1 {
		t.Errorf("first hub built %d clients, want the one from before the replacement", len(first.Clients()))
	}
}
