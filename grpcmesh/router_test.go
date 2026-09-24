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
	if err := router.AddTransport("mem", memTransport(hub)); err != nil {
		t.Fatal(err)
	}

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
	if err := router.AddTransport("mem", memTransport(hub)); err != nil {
		t.Fatal(err)
	}

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
	if err := router.AddTransport("mem", memTransport(hub)); err != nil {
		t.Fatal(err)
	}

	_, err := router.Client("mem")

	if !errors.Is(err, hub.Fail) {
		t.Errorf("err = %v, want the constructor's error", err)
	}
}

func TestRouterClientSwitchesToTheRuntimeClientOnceAnRPCRuntimeExists(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	if err := grpcmesh.AddTransport("mem", memTransport(hub)); err != nil {
		t.Fatal(err)
	}

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

func TestAddTransportRejectsANameAlreadyAddedAndKeepsTheFirstEntry(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	first, second := memtransport.NewHub(), memtransport.NewHub()
	if err := router.AddTransport("mem", memTransport(first)); err != nil {
		t.Fatal(err)
	}

	err := router.AddTransport("mem", grpcmesh.Transport{
		Config:     mesh.Config{"url": "mem://second"},
		ServiceMap: mesh.ServiceMap{Targets: []mesh.Target{testproto.ApiKeyTargets.Search}},
		NewRuntime: second.NewRuntime,
		NewClient:  second.NewClient,
	})

	if !errors.Is(err, grpcmesh.ErrDuplicateTransport) {
		t.Fatalf("err = %v, want ErrDuplicateTransport", err)
	}
	if !strings.Contains(err.Error(), "mem") {
		t.Errorf("err = %q, want the transport name in it", err)
	}
	if _, err := router.Client("mem"); err != nil {
		t.Fatal(err)
	}
	if len(first.Clients()) != 1 || len(second.Clients()) != 0 {
		t.Errorf("first hub built %d clients and second %d, want the first entry kept", len(first.Clients()), len(second.Clients()))
	}
}

func TestRouterCloseClosesTheStandaloneClientsAndForgetsThem(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	hub := memtransport.NewHub()
	if err := router.AddTransport("mem", memTransport(hub)); err != nil {
		t.Fatal(err)
	}
	if err := router.AddTransport("other", memTransport(hub)); err != nil {
		t.Fatal(err)
	}
	if _, err := router.Client("mem"); err != nil {
		t.Fatal(err)
	}
	if _, err := router.Client("other"); err != nil {
		t.Fatal(err)
	}

	if err := router.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := router.Client("mem")
	if err != nil {
		t.Fatal(err)
	}

	clients := hub.Clients()
	if len(clients) != 3 {
		t.Fatalf("NewClient ran %d times, want 2 before Close and 1 after", len(clients))
	}
	if !clients[0].Closed() || !clients[1].Closed() {
		t.Error("a standalone client built before Close is still open")
	}
	if again.(*memtransport.Client).Closed() {
		t.Error("the client built after Close is closed")
	}
}

func TestRouterCloseLeavesTheRuntimeClientToItsRuntime(t *testing.T) {
	freshSingletons(t)
	hub := memtransport.NewHub()
	if err := grpcmesh.AddTransport("mem", memTransport(hub)); err != nil {
		t.Fatal(err)
	}
	if _, err := grpcmesh.NewRPCRuntime("mem", "testproto"); err != nil {
		t.Fatal(err)
	}
	owned, err := grpcmesh.DefaultTransportRouter.Client("mem")
	if err != nil {
		t.Fatal(err)
	}

	if err := grpcmesh.DefaultTransportRouter.Close(); err != nil {
		t.Fatal(err)
	}

	if owned.(*memtransport.Client).Closed() {
		t.Error("Close closed the runtime-owned client")
	}
}

func TestRouterCloseJoinsClientErrors(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	hub := memtransport.NewHub()
	if err := router.AddTransport("mem", memTransport(hub)); err != nil {
		t.Fatal(err)
	}
	c, err := router.Client("mem")
	if err != nil {
		t.Fatal(err)
	}
	c.(*memtransport.Client).CloseErr = errors.New("flush failed")

	err = router.Close()

	if !errors.Is(err, c.(*memtransport.Client).CloseErr) {
		t.Errorf("err = %v, want the client's close error", err)
	}
}
