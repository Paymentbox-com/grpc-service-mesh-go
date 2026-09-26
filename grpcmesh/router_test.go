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

func TestRouterClientReturnsTheClientAdded(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	added := memClient(t, memtransport.NewHub())
	router.AddTransport("mem", added)

	c, err := router.Client("mem")
	if err != nil {
		t.Fatal(err)
	}

	if c != added {
		t.Errorf("Client returned %v, want the client added", c)
	}
}

func TestRouterAddTransportUnderAPresentNameReplacesTheClient(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	hub := memtransport.NewHub()
	router.AddTransport("mem", memClient(t, hub))
	second := memClient(t, hub)
	router.AddTransport("mem", second)

	c, err := router.Client("mem")
	if err != nil {
		t.Fatal(err)
	}

	if c != second {
		t.Errorf("Client returned %v, want the client added last", c)
	}
}

func TestAddTransportAddsToTheDefaultRouter(t *testing.T) {
	freshSingletons(t)
	added := memClient(t, memtransport.NewHub())
	grpcmesh.AddTransport("mem", added)

	c, err := grpcmesh.DefaultTransportRouter.Client("mem")
	if err != nil {
		t.Fatal(err)
	}

	if c != added {
		t.Errorf("Client returned %v, want the client added", c)
	}
}

func TestRouterCloseClosesEveryClientAndJoinsTheirErrors(t *testing.T) {
	router := grpcmesh.NewTransportRouter()
	hub := memtransport.NewHub()
	mem := memClient(t, hub)
	mem.CloseErr = errors.New("mem flush failed")
	other := memClient(t, hub)
	other.CloseErr = errors.New("other flush failed")
	router.AddTransport("mem", mem)
	router.AddTransport("other", other)

	err := router.Close()

	if !mem.Closed() || !other.Closed() {
		t.Error("an added client is still open")
	}
	if !errors.Is(err, mem.CloseErr) || !errors.Is(err, other.CloseErr) {
		t.Errorf("err = %v, want both clients' close errors", err)
	}
	if !strings.Contains(err.Error(), "mem:") || !strings.Contains(err.Error(), "other:") {
		t.Errorf("err = %q, want each transport name in it", err)
	}
}
