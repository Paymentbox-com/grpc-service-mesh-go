package grpcmesh

import (
	"fmt"
	"maps"

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

// RegisterOption adjusts the Endpoints and Subscribers of one Register call.
type RegisterOption func(*registration)

type registration struct {
	groups []targetGroup // one per WithConsumerGroup, in the order given
}

type targetGroup struct {
	target mesh.Target
	group  string
}

// WithConsumerGroup sets the consumer group of the service's Endpoint or
// Subscriber for target, replacing the one the generated code carries from
// the method's consumer_group option. "" removes it, so the runtime's
// deployment group applies, and mesh.ConsumerGroupNone means no group. When
// one target is given more than once, the last value wins.
func WithConsumerGroup(target mesh.Target, group string) RegisterOption {
	return func(r *registration) {
		r.groups = append(r.groups, targetGroup{target: target, group: group})
	}
}

// Register adds svc's Endpoints and Subscribers to DefaultRegistry, with the
// consumer groups opts set. It panics when an option names a target that is
// not one of svc's, since registering happens at boot.
func Register(svc RPCService, opts ...RegisterOption) {
	DefaultRegistry.Register(svc, opts...)
}

// Register adds svc's Endpoints and Subscribers, with the consumer groups
// opts set. Every Endpoint and Subscriber is kept, so two registrations of
// one Target hand the transport two of them. It panics when an option names
// a target that is not one of svc's, since registering happens at boot.
func (r *Registry) Register(svc RPCService, opts ...RegisterOption) {
	var reg registration
	for _, opt := range opts {
		opt(&reg)
	}
	endpoints, subscribers := svc.Endpoints(), svc.Subscribers()
	for _, tg := range reg.groups {
		found := false
		for i := range endpoints {
			if endpoints[i].Target.Equal(tg.target) {
				endpoints[i].Metadata = withConsumerGroup(endpoints[i].Metadata, tg.group)
				found = true
			}
		}
		for i := range subscribers {
			if subscribers[i].Target.Equal(tg.target) {
				subscribers[i].Metadata = withConsumerGroup(subscribers[i].Metadata, tg.group)
				found = true
			}
		}
		if !found {
			panic(fmt.Sprintf("grpcmesh: WithConsumerGroup names target %v, which is not an rpc method of the registered service", tg.target.Segments))
		}
	}
	r.endpoints = append(r.endpoints, endpoints...)
	r.subscribers = append(r.subscribers, subscribers...)
}

// withConsumerGroup returns a copy of md with consumer_group set to group, or
// removed when group is "".
func withConsumerGroup(md map[string]string, group string) map[string]string {
	out := maps.Clone(md)
	if out == nil {
		out = map[string]string{}
	}
	if group == "" {
		delete(out, mesh.ConsumerGroupKey)
	} else {
		out[mesh.ConsumerGroupKey] = group
	}
	return out
}

// Endpoints returns every registered Endpoint.
func (r *Registry) Endpoints() []mesh.Endpoint {
	return r.endpoints
}

// Subscribers returns every registered Subscriber.
func (r *Registry) Subscribers() []mesh.Subscriber {
	return r.subscribers
}
