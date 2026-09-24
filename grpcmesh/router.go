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

// Transport is one entry of a TransportRouter: the Client the application
// built for one transport, the transport's configuration, and the
// constructor of its Runtime. Client holds the connection and the
// transport's ServiceMap. NewRuntime receives that Client, the Config with
// deployment_group set, and the bindings of one deployment group.
type Transport struct {
	Client     mesh.Client
	Config     mesh.Config
	NewRuntime func(mesh.Client, mesh.Config, []mesh.Endpoint, []mesh.Subscriber) (mesh.Runtime, error)
}

// TransportRouter maps transport names to Transport entries and hands out
// each entry's Client. Close closes every entry's Client. It is safe for
// concurrent use.
type TransportRouter struct {
	mu      sync.Mutex
	entries map[string]Transport
}

// DefaultTransportRouter is the process-wide router that generated code and
// NewRPCRuntime use.
var DefaultTransportRouter = NewTransportRouter()

// NewTransportRouter returns an empty router.
func NewTransportRouter() *TransportRouter {
	return &TransportRouter{entries: map[string]Transport{}}
}

// AddTransport adds t under name on DefaultTransportRouter.
func AddTransport(name string, t Transport) {
	DefaultTransportRouter.AddTransport(name, t)
}

// AddTransport adds t under name. Adding under a name already present
// replaces that entry.
func (r *TransportRouter) AddTransport(name string, t Transport) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[name] = t
}

// Get returns the entry added under name.
func (r *TransportRouter) Get(name string) (Transport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.entries[name]
	if !ok {
		return Transport{}, fmt.Errorf("%w: %q", ErrUnknownTransport, name)
	}
	return t, nil
}

// Client returns the Client of the entry added under name.
func (r *TransportRouter) Client(name string) (mesh.Client, error) {
	t, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	return t.Client, nil
}

// Close closes the Client of every entry, joining the clients' errors, each
// wrapped with its transport name. The entries stay, so a later Client call
// returns a closed client.
func (r *TransportRouter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var errs []error
	for name, t := range r.entries {
		if err := t.Client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}
