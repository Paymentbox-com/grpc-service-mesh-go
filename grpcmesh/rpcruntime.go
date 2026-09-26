package grpcmesh

import (
	"context"
	"errors"
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

// NewRPCRuntime builds the runtime for transport and deploymentGroup. It
// takes the transport's Client from DefaultTransportRouter and the Endpoints
// and Subscribers registered in DefaultRegistry whose Targets carry
// deploymentGroup, copies cfg with deployment_group set to deploymentGroup,
// and calls newRuntime with them. The runtime is built here, so Underlying is
// set from this point; Start, Stop, and Running only delegate. Services
// registered afterwards are not served. Errors are ErrNoRuntimeConstructor, ErrUnknownTransport, or
// newRuntime's own.
func NewRPCRuntime(transport, deploymentGroup string, cfg mesh.Config, newRuntime NewRuntimeFunc) (*RPCRuntime, error) {
	return newRPCRuntime(DefaultTransportRouter, DefaultRegistry, transport, deploymentGroup, cfg, newRuntime)
}

func newRPCRuntime(router *TransportRouter, registry *Registry, transport, group string, cfg mesh.Config, newRuntime NewRuntimeFunc) (*RPCRuntime, error) {
	if newRuntime == nil {
		return nil, ErrNoRuntimeConstructor
	}
	client, err := router.Client(transport)
	if err != nil {
		return nil, err
	}

	runtimeCfg := make(mesh.Config, len(cfg)+1)
	maps.Copy(runtimeCfg, cfg)
	runtimeCfg[mesh.DeploymentGroupKey] = group

	rt, err := newRuntime(client, runtimeCfg, registry.Endpoints(group), registry.Subscribers(group))
	if err != nil {
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
