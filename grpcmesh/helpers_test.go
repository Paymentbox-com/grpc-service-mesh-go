package grpcmesh_test

import (
	"context"
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/memtransport"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/testproto"
	"github.com/Paymentbox-com/service-mesh-go/mesh"
	"google.golang.org/protobuf/proto"
)

var memConfig = mesh.Config{"url": "mem://hub"}

var memServiceMap = mesh.ServiceMap{Targets: []mesh.Target{testproto.OrderTargets.Place, testproto.OrderTargets.Placed}}

// freshSingletons installs an empty DefaultTransportRouter and
// DefaultRegistry for the test and restores the previous ones afterwards.
func freshSingletons(t *testing.T) {
	t.Helper()
	router, registry := grpcmesh.DefaultTransportRouter, grpcmesh.DefaultRegistry
	grpcmesh.DefaultTransportRouter = grpcmesh.NewTransportRouter()
	grpcmesh.DefaultRegistry = grpcmesh.NewRegistry()
	t.Cleanup(func() {
		grpcmesh.DefaultTransportRouter = router
		grpcmesh.DefaultRegistry = registry
	})
}

// memClient returns a client served by hub, the one an application would
// add to the router.
func memClient(t *testing.T, hub *memtransport.Hub) *memtransport.Client {
	t.Helper()
	c, err := hub.NewClient(memConfig, memServiceMap)
	if err != nil {
		t.Fatal(err)
	}
	return c.(*memtransport.Client)
}

// serveMem configures the singletons with transport "mem" on a new hub,
// registers svc, and starts an RPCRuntime for it. It returns the hub.
func serveMem(t *testing.T, svc grpcmesh.RPCService) *memtransport.Hub {
	t.Helper()
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	grpcmesh.Register(svc)
	rt, err := grpcmesh.NewRPCRuntime("mem", "shop", memConfig, hub.NewRuntime)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rt.Stop(context.Background()) })
	return hub
}

// serveRaw configures the singletons with transport "mem" on a new hub and
// starts a hub runtime with the given raw bindings. It returns the hub.
func serveRaw(t *testing.T, endpoints []mesh.Endpoint, subscribers []mesh.Subscriber) *memtransport.Hub {
	t.Helper()
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memClient(t, hub))
	cfg := mesh.Config{mesh.DeploymentGroupKey: "shop"}
	c, err := hub.NewClient(cfg, memServiceMap)
	if err != nil {
		t.Fatal(err)
	}
	rt, err := hub.NewRuntime(c, cfg, endpoints, subscribers)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	return hub
}

func mustMarshal(t *testing.T, m proto.Message) []byte {
	t.Helper()
	b, err := proto.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func order(id string) *testproto.Order {
	return &testproto.Order{Id: proto.String(id)}
}
