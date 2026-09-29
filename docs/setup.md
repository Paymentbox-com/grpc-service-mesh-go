# Setup

## Configuring the TransportRouter at Boot

`grpcmesh.DefaultTransportRouter` is the process-wide router. It holds one
`mesh.Client` per transport name. The application builds one transport
`Client` per transport name its definitions use, with the transport's
configuration and the generated `ServiceMap` for that transport, and adds it
under that name. The transport package is a dependency of the application,
not of this library.

```go
import (
    "github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
    "github.com/Paymentbox-com/service-mesh-go/mesh"
    "github.com/Paymentbox-com/service-mesh-nats-go/nats"

    "example.com/definitions/servicemaps"
)

client, err := nats.NewClient(mesh.Config{nats.URLKey: os.Getenv("NATS_URL")}, servicemaps.Nats)
if err != nil { /* the transport's error */ }
grpcmesh.AddTransport("nats", client)
```

`DefaultTransportRouter.Client(name)` returns the client the application added under
`name`, and it is what every generated client method on that transport
sends through. A process that only calls builds its clients, adds them, and
runs `DefaultTransportRouter.Close()` before exit, which closes every client and joins their
errors, so the transport flushes what it has buffered. An `RPCRuntime` for a
transport is built on the router's client for that transport, and its `Stop` closes that
client as well.

`AddTransport` under a name already present replaces the client. A lookup for a
transport that was not added yields `ErrUnknownTransport`, wrapped with the name, from
`Client`, `NewRPCRuntime`, `Call`, and `Publish`.

## Registering a Service

A [generated](generated-code.md) `RPCService` type has one function field per `rpc` method. The
application sets the ones it serves and registers the value in
`grpcmesh.DefaultRegistry`. Implemented `rpc` methods are turned into
Service Mesh API `Endpoints` and `Subscribers` and passed through to the
transport's `Runtime`. An unimplemented (nil) field is excluded.

```go
grpcmesh.Register(shop.OrderService{
    Place: func(ctx context.Context, req *shop.Order) (*shop.Order, error) {
        if !store.InStock(req.GetItem()) {
            return nil, grpcmesh.NewNotFoundError("no such item")
        }
        return &shop.Order{Id: proto.String(store.Place(req.GetItem())), Item: req.Item}, nil
    },
    Placed: func(ctx context.Context, ev *shop.Order) error {
        return audit.Record(ev)
    },
})
```

Every registered binding is kept, so two registrations that have identical `Targets` hand
the transport two bindings for it.

## Constructing and Starting an RPCRuntime

`NewRPCRuntime(transport, deploymentGroup, cfg, newRuntime, opts...)` takes the
transport's `Client` from `DefaultTransportRouter` and the `Endpoints` and
`Subscribers` whose `Targets` carry `deploymentGroup` from `DefaultRegistry`,
copies `cfg` with `deployment_group` set to the specified `deploymentGroup`, and
calls `newRuntime`.

`newRuntime` is a `NewRuntimeFunc`, `func(mesh.Client, mesh.Config, []mesh.Endpoint, []mesh.Subscriber)
(mesh.Runtime, error)`. The transport's runtime is built in the constructor, so `TRuntime()` is set before `Start`.

A `deployment_group` key in `cfg` is overwritten by `deploymentGroup`, and `cfg` itself is left unchanged. Services
registered after the constructor has run are not served.

```go
rt, err := grpcmesh.NewRPCRuntime("nats", "shop", mesh.Config{},
    func(c mesh.Client, cfg mesh.Config, e []mesh.Endpoint, s []mesh.Subscriber) (mesh.Runtime, error) {
        return nats.New(c.(*nats.Client), cfg, e, s)
    })
if err != nil { /* ErrNoRuntimeConstructor, ErrUnknownTransport, ErrTargetOutsideRuntime, or the transport constructor's error */ }
if err := rt.Start(ctx); err != nil { /* the transport's error */ }

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
<-stop

drain, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = rt.Stop(drain)
```

In the above example, `nats.New` takes a `*nats.Client`, so the function asserts the `mesh.Client`
the router passes to it, which is the one added under `nats`.

`WithEndpoints` and `WithSubscribers` give the runtime a list to serve in
place of the registry's list of that kind, so a process can serve only part of a
deployment group. `WithEndpoints()` or `WithSubscribers()` with no arguments serves none of that kind.

Every `Target` in a given list must carry the runtime's `deployment_group` and `transport`, and
passing one that does not yields `ErrTargetOutsideRuntime`, wrapped with the Target's segments
and the key that differs.

```go
rt, err := grpcmesh.NewRPCRuntime("nats", "shop", mesh.Config{}, newRuntime,
    grpcmesh.WithEndpoints(shop.OrderService{Place: place}.Endpoints()...))
```

`RPCRuntime` implements `mesh.Runtime`. `Start`, `Stop`, `Running`, and
`Client` delegate to the underlying transport runtime, which `TRuntime()` returns.
`Transport()` and `DeploymentGroup()` return the transport and deployment group the
`RPCRuntime` was built for.
