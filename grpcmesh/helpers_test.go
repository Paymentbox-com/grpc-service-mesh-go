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

var memServiceMap = mesh.ServiceMap{Targets: []mesh.Target{testproto.ApiKeyTargets.Search, testproto.ApiKeyTargets.Created}}

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

// memTransport returns a Transport entry served by hub.
func memTransport(hub *memtransport.Hub) grpcmesh.Transport {
	return grpcmesh.Transport{
		Config:     memConfig,
		ServiceMap: memServiceMap,
		NewRuntime: hub.NewRuntime,
		NewClient:  hub.NewClient,
	}
}

// serveMem configures the singletons with transport "mem" on a new hub,
// registers svc, and starts an RPCRuntime for it. It returns the hub.
func serveMem(t *testing.T, svc grpcmesh.RPCService) *memtransport.Hub {
	t.Helper()
	freshSingletons(t)
	hub := memtransport.NewHub()
	grpcmesh.AddTransport("mem", memTransport(hub))
	if err := grpcmesh.Register(svc); err != nil {
		t.Fatal(err)
	}
	rt, err := grpcmesh.NewRPCRuntime("mem", "testproto")
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
	grpcmesh.AddTransport("mem", memTransport(hub))
	cfg := mesh.Config{mesh.DeploymentGroupKey: "testproto"}
	rt, err := hub.NewRuntime(cfg, memServiceMap, endpoints, subscribers)
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

func apiKey(first string) *testproto.ApiKey {
	return &testproto.ApiKey{FirstName: proto.String(first)}
}
