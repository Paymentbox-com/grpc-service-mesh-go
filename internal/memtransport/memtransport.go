// Package memtransport is an in-process Service Mesh API transport for the
// grpcmesh tests. A Hub stands in for the broker: every Client it builds, and
// every Runtime bound on one of those clients, delivers through it, and it
// records what it built.
//
// Request finds the first endpoint with an equal target across the hub's
// running runtimes and returns the handler's reply or error. Publish delivers
// to every subscriber with an equal target across the running runtimes and
// returns the handlers' errors joined. Consumer groups are not modelled.
package memtransport

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

// ErrNoReceiver is returned by Request when no running runtime serves the
// target.
var ErrNoReceiver = errors.New("memtransport: no receiver for target")

// ErrNotRestartable is returned by Start after Stop.
var ErrNotRestartable = errors.New("memtransport: runtime is not restartable")

// Sent is one Request or Publish a Client made.
type Sent struct {
	Message mesh.Message
	Options map[string]string
}

// Hub delivers messages between the clients it builds and the runtimes bound
// on them, and records the constructor arguments it received.
type Hub struct {
	// Fail, when set, is returned by NewRuntime.
	Fail error

	mu       sync.Mutex
	runtimes []*Runtime
	clients  []*Client
}

// NewHub returns an empty hub.
func NewHub() *Hub {
	return &Hub{}
}

// NewClient builds the client an application would add to the
// grpcmesh.TransportRouter.
func (h *Hub) NewClient(cfg mesh.Config, sm mesh.ServiceMap) (mesh.Client, error) {
	c := &Client{Config: cfg, ServiceMap: sm, hub: h}
	h.mu.Lock()
	h.clients = append(h.clients, c)
	h.mu.Unlock()
	return c, nil
}

// NewRuntime is a grpcmesh.NewRuntimeFunc. The client
// is one this package built, and the runtime binds on that client's hub.
func (h *Hub) NewRuntime(client mesh.Client, cfg mesh.Config, endpoints []mesh.Endpoint, subscribers []mesh.Subscriber) (mesh.Runtime, error) {
	if h.Fail != nil {
		return nil, h.Fail
	}
	c, ok := client.(*Client)
	if !ok {
		return nil, fmt.Errorf("memtransport: client is %T, want *memtransport.Client", client)
	}
	if cfg[mesh.DeploymentGroupKey] == "" {
		return nil, mesh.ErrNoDeploymentGroup
	}
	for _, e := range endpoints {
		if e.Target.Kind != mesh.KindRoute {
			return nil, mesh.ErrKindMismatch
		}
	}
	for _, s := range subscribers {
		if s.Target.Kind != mesh.KindTopic {
			return nil, mesh.ErrKindMismatch
		}
	}
	rt := &Runtime{Config: cfg, Endpoints: endpoints, Subscribers: subscribers, client: c}
	c.hub.mu.Lock()
	c.hub.runtimes = append(c.hub.runtimes, rt)
	c.hub.mu.Unlock()
	return rt, nil
}

// Runtimes returns the runtimes bound on this hub so far, in order.
func (h *Hub) Runtimes() []*Runtime {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*Runtime(nil), h.runtimes...)
}

// Clients returns the clients built so far, in order.
func (h *Hub) Clients() []*Client {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*Client(nil), h.clients...)
}

func (h *Hub) request(ctx context.Context, msg mesh.Message) (mesh.Message, error) {
	for _, rt := range h.Runtimes() {
		if !rt.Running() {
			continue
		}
		for _, e := range rt.Endpoints {
			if e.Target.Equal(msg.Target) {
				return e.Handler(ctx, msg)
			}
		}
	}
	return mesh.Message{}, ErrNoReceiver
}

func (h *Hub) publish(ctx context.Context, msg mesh.Message) error {
	var errs []error
	for _, rt := range h.Runtimes() {
		if !rt.Running() {
			continue
		}
		for _, s := range rt.Subscribers {
			if s.Target.Equal(msg.Target) {
				errs = append(errs, s.Handler(ctx, msg))
			}
		}
	}
	return errors.Join(errs...)
}

// Runtime is an in-process mesh.Runtime. The exported fields are what it was
// built with.
type Runtime struct {
	Config      mesh.Config
	Endpoints   []mesh.Endpoint
	Subscribers []mesh.Subscriber

	client *Client

	mu      sync.Mutex
	running bool
	stopped bool
}

var _ mesh.Runtime = (*Runtime)(nil)

// Client returns the client the runtime was built on.
func (r *Runtime) Client() mesh.Client {
	return r.client
}

// Start begins receiving.
func (r *Runtime) Start(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return ErrNotRestartable
	}
	r.running = true
	return nil
}

// Stop stops receiving and closes the client.
func (r *Runtime) Stop(context.Context) error {
	r.mu.Lock()
	r.running = false
	r.stopped = true
	r.mu.Unlock()
	return r.client.Close()
}

// Running reports whether Start has run and Stop has not.
func (r *Runtime) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Client is an in-process mesh.Client. The exported fields are what it was
// built with.
type Client struct {
	// CloseErr is what Close returns.
	CloseErr   error
	Config     mesh.Config
	ServiceMap mesh.ServiceMap

	hub *Hub

	mu        sync.Mutex
	requests  []Sent
	publishes []Sent
	closed    bool
}

var _ mesh.Client = (*Client)(nil)

// Request records the call and delivers msg to a running endpoint.
func (c *Client) Request(ctx context.Context, msg mesh.Message, opts map[string]string) (mesh.Message, error) {
	if msg.Target.Kind != mesh.KindRoute {
		return mesh.Message{}, mesh.ErrKindMismatch
	}
	c.mu.Lock()
	c.requests = append(c.requests, Sent{Message: msg, Options: opts})
	c.mu.Unlock()
	return c.hub.request(ctx, msg)
}

// Publish records the call and delivers msg to the running subscribers.
func (c *Client) Publish(ctx context.Context, msg mesh.Message, opts map[string]string) error {
	if msg.Target.Kind != mesh.KindTopic {
		return mesh.ErrKindMismatch
	}
	c.mu.Lock()
	c.publishes = append(c.publishes, Sent{Message: msg, Options: opts})
	c.mu.Unlock()
	return c.hub.publish(ctx, msg)
}

// Close marks the client closed.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return c.CloseErr
}

// Requests returns the Request calls made so far, in order.
func (c *Client) Requests() []Sent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Sent(nil), c.requests...)
}

// Publishes returns the Publish calls made so far, in order.
func (c *Client) Publishes() []Sent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Sent(nil), c.publishes...)
}

// Closed reports whether Close has run.
func (c *Client) Closed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}
