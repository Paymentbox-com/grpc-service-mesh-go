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

    "example.com/definitions/servicemaps"
)

// client is the transport's mesh.Client, built from its configuration and servicemaps.Nats.
grpcmesh.AddTransport("nats", client)
```

The application owns each client and the connection it holds. Clients are
added at boot, before any call is made or any `RPCRuntime` is built. Generated
clients and `RPCRuntime` find the process router themselves, so nothing
generated takes a router argument.

`DefaultTransportRouter.Client(name)` returns the client the application added under
`name`, and it is what every generated client method on that transport
sends through. `AddTransport` under a name already present replaces the client.
Looking up a name that was not added returns `ErrUnknownTransport`, listed under
[Library Errors](mesherror.md#library-errors).

A process that only calls builds its clients, adds them, and runs
`DefaultTransportRouter.Close()` before exit, so each transport sends what it has
buffered. `Close` closes every client, even when some of them fail to close. It
returns nil when every client closed, and otherwise one error that joins each
failure, prefixed with its transport name.

An `RPCRuntime` for a transport is built on the router's client for that
transport, and its `Stop` closes that client as well.

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

The registry keeps every endpoint and subscriber registered with it. If two
registered services serve the same `Target`, the transport is given two
endpoints or subscribers for it.

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
rt, err := grpcmesh.NewRPCRuntime("nats", "shop", mesh.Config{}, newRuntime)
if err != nil { /* a library error, or the error newRuntime returns */ }
if err := rt.Start(ctx); err != nil { /* the transport's error */ }

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
<-stop

drain, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = rt.Stop(drain)
```

In the above example, `newRuntime` is a `NewRuntimeFunc` that builds the transport's `mesh.Runtime`. When the
transport's runtime constructor takes the transport's own client type, `newRuntime` asserts the `mesh.Client` the
router passes to it, which is the one added under `nats`.

`NewRPCRuntime` returns a library error when `newRuntime` is nil, when the transport was not added to the router,
or when a given `Target` is outside the runtime. These are listed under
[Library Errors](mesherror.md#library-errors). The error `newRuntime` returns is passed through unchanged.

`WithEndpoints` and `WithSubscribers` give the runtime a list to serve in
place of the registry's list of that kind, so a process can serve only part of a
deployment group. `WithEndpoints()` or `WithSubscribers()` with no arguments serves none of that kind.
Every `Target` in a given list must carry the runtime's `deployment_group` and `transport`.

```go
rt, err := grpcmesh.NewRPCRuntime("nats", "shop", mesh.Config{}, newRuntime,
    grpcmesh.WithEndpoints(shop.OrderService{Place: place}.Endpoints()...))
```

`RPCRuntime` implements `mesh.Runtime`. `Start`, `Stop`, `Running`, and
`Client` delegate to the underlying transport runtime, which `TRuntime()` returns.
`Stop` closes the client. The transport documents what its runtime does,
including what `Stop` returns and how it treats a subscriber handler that returns an error.
`Transport()` and `DeploymentGroup()` return the transport and deployment group the
`RPCRuntime` was built for.
