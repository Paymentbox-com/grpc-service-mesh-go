package grpcmesh

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

// ErrUnknownTransport is returned, wrapped with the transport name, when a
// transport has not been added to the TransportRouter.
var ErrUnknownTransport = errors.New("grpcmesh: unknown transport")

// ErrDuplicateTransport is returned, wrapped with the name, when a transport
// name is added to a TransportRouter a second time.
var ErrDuplicateTransport = errors.New("grpcmesh: transport already added")

// ErrDuplicateRuntime is returned, wrapped with the transport name, when a
// second RPCRuntime is constructed for a transport on the same router.
var ErrDuplicateRuntime = errors.New("grpcmesh: an RPCRuntime already exists for this transport")

// Transport is one entry of a TransportRouter: the configuration and the
// ServiceMap of one transport together with the constructors of its Runtime
// and Client.
type Transport struct {
	Config     mesh.Config
	ServiceMap mesh.ServiceMap
	NewRuntime func(mesh.Config, mesh.ServiceMap, []mesh.Endpoint, []mesh.Subscriber) (mesh.Runtime, error)
	NewClient  func(mesh.Config, mesh.ServiceMap) (mesh.Client, error)
}

// TransportRouter maps transport names to Transport entries and hands out
// one mesh.Client per transport. It is safe for concurrent use.
type TransportRouter struct {
	mu      sync.Mutex
	entries map[string]*entry
}

type entry struct {
	transport  Transport
	runtime    mesh.Runtime
	standalone mesh.Client
}

// DefaultTransportRouter is the process-wide router that generated code and
// NewRPCRuntime use.
var DefaultTransportRouter = NewTransportRouter()

// NewTransportRouter returns an empty router.
func NewTransportRouter() *TransportRouter {
	return &TransportRouter{entries: map[string]*entry{}}
}

// AddTransport adds t under name on DefaultTransportRouter.
func AddTransport(name string, t Transport) error {
	return DefaultTransportRouter.AddTransport(name, t)
}

// AddTransport adds t under name. A name already added returns
// ErrDuplicateTransport and keeps the first entry.
func (r *TransportRouter) AddTransport(name string, t Transport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[name]; ok {
		return fmt.Errorf("%w: %q", ErrDuplicateTransport, name)
	}
	r.entries[name] = &entry{transport: t}
	return nil
}

// Get returns the entry added under name.
func (r *TransportRouter) Get(name string) (Transport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, err := r.entry(name)
	if err != nil {
		return Transport{}, err
	}
	return e.transport, nil
}

// Client returns the client for transport name. Once an RPCRuntime exists for
// the transport on this router, it is that runtime's Client. Before one
// exists, it is a standalone client built once from the entry's NewClient
// with the entry's Config and ServiceMap and cached.
func (r *TransportRouter) Client(name string) (mesh.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, err := r.entry(name)
	if err != nil {
		return nil, err
	}
	if e.runtime != nil {
		return e.runtime.Client(), nil
	}
	if e.standalone == nil {
		c, err := e.transport.NewClient(e.transport.Config, e.transport.ServiceMap)
		if err != nil {
			return nil, err
		}
		e.standalone = c
	}
	return e.standalone, nil
}

// bindRuntime records rt as the RPCRuntime's underlying runtime for name.
func (r *TransportRouter) bindRuntime(name string, rt mesh.Runtime) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, err := r.entry(name)
	if err != nil {
		return err
	}
	if e.runtime != nil {
		return fmt.Errorf("%w: %s", ErrDuplicateRuntime, name)
	}
	e.runtime = rt
	return nil
}

// reserveRuntime fails when name is unknown or already has a runtime, so a
// constructor can refuse before building anything.
func (r *TransportRouter) reserveRuntime(name string) (Transport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, err := r.entry(name)
	if err != nil {
		return Transport{}, err
	}
	if e.runtime != nil {
		return Transport{}, fmt.Errorf("%w: %s", ErrDuplicateRuntime, name)
	}
	return e.transport, nil
}

func (r *TransportRouter) entry(name string) (*entry, error) {
	e, ok := r.entries[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownTransport, name)
	}
	return e, nil
}
