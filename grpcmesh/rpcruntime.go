package grpcmesh

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

// RPCRuntime serves one deployment group over one transport. It wraps the
// transport's mesh.Runtime and implements mesh.Runtime by delegation.
type RPCRuntime struct {
	transport  string
	group      string
	underlying mesh.Runtime
}

var _ mesh.Runtime = (*RPCRuntime)(nil)

// NewRuntimeFunc builds a transport's mesh.Runtime from the transport's
// Client, the Config with deployment_group set, and the Endpoints and
// Subscribers of one deployment group.
type NewRuntimeFunc func(mesh.Client, mesh.Config, []mesh.Endpoint, []mesh.Subscriber) (mesh.Runtime, error)

// ErrNoRuntimeConstructor is returned by NewRPCRuntime when newRuntime is nil.
var ErrNoRuntimeConstructor = errors.New("grpcmesh: NewRPCRuntime needs a runtime constructor; newRuntime is nil")

// ErrTargetOutsideRuntime is returned by NewRPCRuntime, wrapped with the
// Target's segments and the metadata key that differs, when an Endpoint or
// Subscriber given with WithEndpoints or WithSubscribers has a Target whose
// deployment_group or transport is not the runtime's.
var ErrTargetOutsideRuntime = errors.New("grpcmesh: target is outside the RPCRuntime's deployment group or transport")

// RPCRuntimeOption configures NewRPCRuntime.
type RPCRuntimeOption func(*rpcRuntimeOptions)

type rpcRuntimeOptions struct {
	endpoints      []mesh.Endpoint
	endpointsSet   bool
	subscribers    []mesh.Subscriber
	subscribersSet bool
}

// WithEndpoints makes the runtime serve e in place of the Endpoints in
// DefaultRegistry. WithEndpoints() with no arguments serves no Endpoints.
func WithEndpoints(e ...mesh.Endpoint) RPCRuntimeOption {
	return func(o *rpcRuntimeOptions) {
		o.endpoints, o.endpointsSet = e, true
	}
}

// WithSubscribers makes the runtime serve s in place of the Subscribers in
// DefaultRegistry. WithSubscribers() with no arguments serves no Subscribers.
func WithSubscribers(s ...mesh.Subscriber) RPCRuntimeOption {
	return func(o *rpcRuntimeOptions) {
		o.subscribers, o.subscribersSet = s, true
	}
}

// NewRPCRuntime builds the runtime for transport and deploymentGroup. It
// takes the transport's Client from DefaultTransportRouter and the Endpoints
// and Subscribers registered in DefaultRegistry whose Targets carry
// deploymentGroup, copies cfg with deployment_group set to deploymentGroup,
// and calls newRuntime with them. WithEndpoints and WithSubscribers each
// replace the registry's list of that kind, and every Target in a list given
// that way must carry deploymentGroup and transport. The runtime is built
// here, so Underlying is set from this point; Start, Stop, and Running only
// delegate. Services registered afterwards are not served. Errors are
// ErrNoRuntimeConstructor, ErrUnknownTransport, ErrTargetOutsideRuntime, or
// newRuntime's own.
func NewRPCRuntime(transport, deploymentGroup string, cfg mesh.Config, newRuntime NewRuntimeFunc, opts ...RPCRuntimeOption) (*RPCRuntime, error) {
	return newRPCRuntime(DefaultTransportRouter, DefaultRegistry, transport, deploymentGroup, cfg, newRuntime, opts)
}

func newRPCRuntime(router *TransportRouter, registry *Registry, transport, group string, cfg mesh.Config, newRuntime NewRuntimeFunc, opts []RPCRuntimeOption) (*RPCRuntime, error) {
	if newRuntime == nil {
		return nil, ErrNoRuntimeConstructor
	}
	client, err := router.Client(transport)
	if err != nil {
		return nil, err
	}

	var o rpcRuntimeOptions
	for _, opt := range opts {
		opt(&o)
	}
	endpoints := registry.Endpoints(group)
	if o.endpointsSet {
		for _, e := range o.endpoints {
			if err := checkTarget("endpoint", e.Target, transport, group); err != nil {
				return nil, err
			}
		}
		endpoints = o.endpoints
	}
	subscribers := registry.Subscribers(group)
	if o.subscribersSet {
		for _, s := range o.subscribers {
			if err := checkTarget("subscriber", s.Target, transport, group); err != nil {
				return nil, err
			}
		}
		subscribers = o.subscribers
	}

	runtimeCfg := make(mesh.Config, len(cfg)+1)
	maps.Copy(runtimeCfg, cfg)
	runtimeCfg[mesh.DeploymentGroupKey] = group

	rt, err := newRuntime(client, runtimeCfg, endpoints, subscribers)
	if err != nil {
		return nil, err
	}
	return &RPCRuntime{transport: transport, group: group, underlying: rt}, nil
}

// checkTarget reports a Target whose deployment_group or transport metadata
// is not group or transport.
func checkTarget(kind string, t mesh.Target, transport, group string) error {
	if got := t.Metadata[mesh.DeploymentGroupKey]; got != group {
		return fmt.Errorf("%w: %s %v has %s %q, want %q", ErrTargetOutsideRuntime, kind, t.Segments, mesh.DeploymentGroupKey, got, group)
	}
	if got := t.Metadata[TransportKey]; got != transport {
		return fmt.Errorf("%w: %s %v has %s %q, want %q", ErrTargetOutsideRuntime, kind, t.Segments, TransportKey, got, transport)
	}
	return nil
}

// Underlying returns the transport's runtime.
func (r *RPCRuntime) Underlying() mesh.Runtime {
	return r.underlying
}

// Transport returns the transport name the runtime was built for.
func (r *RPCRuntime) Transport() string {
	return r.transport
}

// DeploymentGroup returns the deployment group the runtime serves.
func (r *RPCRuntime) DeploymentGroup() string {
	return r.group
}

// Client returns the underlying runtime's client, the one the router holds.
func (r *RPCRuntime) Client() mesh.Client {
	return r.underlying.Client()
}

// Start starts the underlying runtime.
func (r *RPCRuntime) Start(ctx context.Context) error {
	return r.underlying.Start(ctx)
}

// Stop stops the underlying runtime, draining until ctx is done. The
// underlying runtime closes its client.
func (r *RPCRuntime) Stop(ctx context.Context) error {
	return r.underlying.Stop(ctx)
}

// Running reports the underlying runtime's state.
func (r *RPCRuntime) Running() bool {
	return r.underlying.Running()
}
