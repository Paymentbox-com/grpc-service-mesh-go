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
// one mesh.Client per transport. Close releases the clients it built. It is
// safe for concurrent use.
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
func AddTransport(name string, t Transport) {
	DefaultTransportRouter.AddTransport(name, t)
}

// AddTransport adds t under name. Adding under a name already present
// replaces that entry, including any runtime and standalone client it held.
func (r *TransportRouter) AddTransport(name string, t Transport) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[name] = &entry{transport: t}
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

// bindRuntime records rt as the runtime whose Client the router hands out
// for name. The newest runtime bound for a name is the one used.
func (r *TransportRouter) bindRuntime(name string, rt mesh.Runtime) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, err := r.entry(name)
	if err != nil {
		return err
	}
	e.runtime = rt
	return nil
}

func (r *TransportRouter) entry(name string) (*entry, error) {
	e, ok := r.entries[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownTransport, name)
	}
	return e, nil
}

// Close closes every standalone client the router has built and forgets
// them, so a later Client call builds a new one. A client shared with an
// RPCRuntime belongs to that runtime and is released by its Stop. The
// clients' errors are joined.
func (r *TransportRouter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var errs []error
	for name, e := range r.entries {
		if e.standalone == nil {
			continue
		}
		if err := e.standalone.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
		e.standalone = nil
	}
	return errors.Join(errs...)
}
