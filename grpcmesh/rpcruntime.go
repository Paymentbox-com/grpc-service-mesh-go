package grpcmesh

import (
	"context"
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

// NewRPCRuntime builds the runtime for transport and deploymentGroup from the
// process singletons. It takes the Endpoints and Subscribers registered in
// DefaultRegistry whose Targets carry deploymentGroup, takes the Transport
// entry from DefaultTransportRouter, and calls the entry's NewRuntime with
// the entry's Config plus deployment_group set to deploymentGroup, the
// entry's ServiceMap, and those bindings. The runtime is built here, so
// Underlying is set and the router hands out its Client from this point;
// Start, Stop, and Running only delegate. Services registered afterwards are
// not served. Errors are ErrUnknownTransport, ErrDuplicateRuntime, or the
// transport constructor's own.
func NewRPCRuntime(transport, deploymentGroup string) (*RPCRuntime, error) {
	return newRPCRuntime(DefaultTransportRouter, DefaultRegistry, transport, deploymentGroup)
}

func newRPCRuntime(router *TransportRouter, registry *Registry, transport, group string) (*RPCRuntime, error) {
	t, err := router.reserveRuntime(transport)
	if err != nil {
		return nil, err
	}

	cfg := make(mesh.Config, len(t.Config)+1)
	maps.Copy(cfg, t.Config)
	cfg[mesh.DeploymentGroupKey] = group

	rt, err := t.NewRuntime(cfg, t.ServiceMap, registry.Endpoints(group), registry.Subscribers(group))
	if err != nil {
		return nil, err
	}
	if err := router.bindRuntime(transport, rt); err != nil {
		return nil, err
	}
	return &RPCRuntime{transport: transport, group: group, underlying: rt}, nil
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

// Client returns the underlying runtime's client.
func (r *RPCRuntime) Client() mesh.Client {
	return r.underlying.Client()
}

// Start starts the underlying runtime.
func (r *RPCRuntime) Start(ctx context.Context) error {
	return r.underlying.Start(ctx)
}

// Stop stops the underlying runtime, draining until ctx is done.
func (r *RPCRuntime) Stop(ctx context.Context) error {
	return r.underlying.Stop(ctx)
}

// Running reports the underlying runtime's state.
func (r *RPCRuntime) Running() bool {
	return r.underlying.Running()
}
