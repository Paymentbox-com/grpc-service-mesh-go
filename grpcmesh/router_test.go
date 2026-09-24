package grpcmesh_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"github.com/Paymentbox-com/grpc-service-mesh-go/internal/memtransport"
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
	added := memTransport(t, memtransport.NewHub())
	router.AddTransport("mem", added)

	entry, err := router.Get("mem")
	if err != nil {
		t.Fatal(err)
	}

	if entry.Client != added.Client || entry.Config["url"] != memConfig["url"] {
		t.Errorf("Get returned %+v", entry)
	}
}

func TestRouterClientReturnsTheClientAdded(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	added := memTransport(t, memtransport.NewHub())
	router.AddTransport("mem", added)

	c, err := router.Client("mem")
	if err != nil {
		t.Fatal(err)
	}

	if c != added.Client {
		t.Errorf("Client returned %v, want the client added", c)
	}
}

func TestRouterCloseClosesEveryClientAndJoinsTheirErrors(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	hub := memtransport.NewHub()
	mem := memTransport(t, hub)
	mem.Client.(*memtransport.Client).CloseErr = errors.New("mem flush failed")
	other := memTransport(t, hub)
	other.Client.(*memtransport.Client).CloseErr = errors.New("other flush failed")
	router.AddTransport("mem", mem)
	router.AddTransport("other", other)

	err := router.Close()

	if !mem.Client.(*memtransport.Client).Closed() || !other.Client.(*memtransport.Client).Closed() {
		t.Error("an added client is still open")
	}
	if !errors.Is(err, mem.Client.(*memtransport.Client).CloseErr) || !errors.Is(err, other.Client.(*memtransport.Client).CloseErr) {
		t.Errorf("err = %v, want both clients' close errors", err)
	}
	if !strings.Contains(err.Error(), "mem:") || !strings.Contains(err.Error(), "other:") {
		t.Errorf("err = %q, want each transport name in it", err)
	}
}
