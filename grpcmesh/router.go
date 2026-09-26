package grpcmesh

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

// ErrUnknownTransport is returned, wrapped with the transport name, when no
// client has been added to the TransportRouter under a transport name.
var ErrUnknownTransport = errors.New("grpcmesh: unknown transport")

// TransportRouter holds one mesh.Client per transport name: the
// transport-specific client the application built. Close closes every
// client. It is safe for concurrent use.
type TransportRouter struct {
	mu      sync.Mutex
	clients map[string]mesh.Client
}

// DefaultTransportRouter is the process-wide router that generated code and
// NewRPCRuntime use.
var DefaultTransportRouter = NewTransportRouter()

// NewTransportRouter returns an empty router.
func NewTransportRouter() *TransportRouter {
	return &TransportRouter{clients: map[string]mesh.Client{}}
}

// AddTransport adds c under name on DefaultTransportRouter.
func AddTransport(name string, c mesh.Client) {
	DefaultTransportRouter.AddTransport(name, c)
}

// AddTransport adds c under name. Adding under a name already present
// replaces that client.
func (r *TransportRouter) AddTransport(name string, c mesh.Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[name] = c
}

// Client returns the client added under name.
func (r *TransportRouter) Client(name string) (mesh.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.clients[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownTransport, name)
	}
	return c, nil
}

// Close closes every client, joining their errors, each wrapped with its
// transport name. The clients stay, so a later Client call returns a closed
// client.
func (r *TransportRouter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var errs []error
	for name, c := range r.clients {
		if err := c.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}
