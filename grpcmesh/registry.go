package grpcmesh

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

// ErrDuplicateTarget is returned, wrapped with the target, when a binding is
// registered for a Target that already has one.
var ErrDuplicateTarget = errors.New("grpcmesh: target already registered")

// RPCService is what generated service types implement: the Endpoints and
// Subscribers of the rpc methods an application serves.
type RPCService interface {
	Endpoints() []mesh.Endpoint
	Subscribers() []mesh.Subscriber
}

// Registry collects every Endpoint and Subscriber a process serves. It is
// safe for concurrent use.
type Registry struct {
	mu          sync.Mutex
	endpoints   []mesh.Endpoint
	subscribers []mesh.Subscriber
}

// DefaultRegistry is the process-wide registry that Register and
// NewRPCRuntime use.
var DefaultRegistry = NewRegistry()

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds svc's bindings to DefaultRegistry.
func Register(svc RPCService) error {
	return DefaultRegistry.Register(svc)
}

// Register adds svc's Endpoints and Subscribers. When any of their Targets is
// already registered, or appears twice in svc, nothing is added and the
// error wraps ErrDuplicateTarget with the target.
func (r *Registry) Register(svc RPCService) error {
	endpoints, subscribers := svc.Endpoints(), svc.Subscribers()

	r.mu.Lock()
	defer r.mu.Unlock()

	var seen []mesh.Target
	for _, e := range r.endpoints {
		seen = append(seen, e.Target)
	}
	for _, s := range r.subscribers {
		seen = append(seen, s.Target)
	}
	for _, e := range endpoints {
		if err := checkNew(seen, e.Target); err != nil {
			return err
		}
		seen = append(seen, e.Target)
	}
	for _, s := range subscribers {
		if err := checkNew(seen, s.Target); err != nil {
			return err
		}
		seen = append(seen, s.Target)
	}

	r.endpoints = append(r.endpoints, endpoints...)
	r.subscribers = append(r.subscribers, subscribers...)
	return nil
}

// Endpoints returns the registered Endpoints whose Target carries
// deployment_group metadata equal to group.
func (r *Registry) Endpoints(group string) []mesh.Endpoint {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []mesh.Endpoint
	for _, e := range r.endpoints {
		if e.Target.Metadata[mesh.DeploymentGroupKey] == group {
			out = append(out, e)
		}
	}
	return out
}

// Subscribers returns the registered Subscribers whose Target carries
// deployment_group metadata equal to group.
func (r *Registry) Subscribers(group string) []mesh.Subscriber {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []mesh.Subscriber
	for _, s := range r.subscribers {
		if s.Target.Metadata[mesh.DeploymentGroupKey] == group {
			out = append(out, s)
		}
	}
	return out
}

func checkNew(seen []mesh.Target, t mesh.Target) error {
	for _, s := range seen {
		if s.Equal(t) {
			return fmt.Errorf("%w: %s %s", ErrDuplicateTarget, t.Kind, strings.Join(t.Segments, "."))
		}
	}
	return nil
}
