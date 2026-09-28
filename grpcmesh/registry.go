package grpcmesh

import (
	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

// RPCService is what generated service types implement: the Endpoints and
// Subscribers of the rpc methods an application serves.
type RPCService interface {
	Endpoints() []mesh.Endpoint
	Subscribers() []mesh.Subscriber
}

// Registry collects every Endpoint and Subscriber a process serves. Services
// are registered at boot, before any RPCRuntime is built.
type Registry struct {
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
func Register(svc RPCService) {
	DefaultRegistry.Register(svc)
}

// Register adds svc's Endpoints and Subscribers. Every binding is kept, so
// two registrations of one Target hand the transport two bindings for it.
func (r *Registry) Register(svc RPCService) {
	endpoints, subscribers := svc.Endpoints(), svc.Subscribers()

	r.endpoints = append(r.endpoints, endpoints...)
	r.subscribers = append(r.subscribers, subscribers...)
}

// Endpoints returns the registered Endpoints whose Target carries
// deployment_group metadata equal to group.
func (r *Registry) Endpoints(group string) []mesh.Endpoint {
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
	var out []mesh.Subscriber
	for _, s := range r.subscribers {
		if s.Target.Metadata[mesh.DeploymentGroupKey] == group {
			out = append(out, s)
		}
	}
	return out
}
