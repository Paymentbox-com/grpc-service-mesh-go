# Setup

A process that uses this library sets it up at boot, in three steps. It adds
one transport `Client` per transport name to the process router, registers the
services it implements, and builds and starts one `RPCRuntime` per transport
and deployment group it serves. A process that only calls needs only the first
step.

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

// client is the transport's mesh.Client, built from its configuration and servicemaps.Mem.
grpcmesh.AddTransport("mem", client)
```

The application owns each client and the connection it holds. Clients are
added at boot, before any call is made or any `RPCRuntime` is built. Generated
clients and `RPCRuntime` find the process router themselves, so nothing
generated takes a router argument.

`DefaultTransportRouter.Client(name)` returns the client the application added
under `name`, and it is what every generated client method on that transport
sends through. `AddTransport` under a name already present replaces the client.
Looking up a name that was not added returns `ErrUnknownTransport`, listed
under [Library Errors](mesherror.md#library-errors).

A process that only calls builds its clients, adds them, and runs
`DefaultTransportRouter.Close()` before exit, so each transport sends what it
has buffered. `Close` closes every client, even when some of them fail to
close. It returns nil when every client closed, and otherwise one error that
joins each failure, prefixed with its transport name.

An `RPCRuntime` for a transport is built on the router's client for that
transport, and its `Stop` closes that client as well.

## Registering a Service

The application implements a generated `RPCService` as described under
[Handlers](handlers.md), and registers the value in `grpcmesh.DefaultRegistry`.
Each rpc method it implements becomes a Service Mesh API `Endpoint` or
`Subscriber`, which an `RPCRuntime` passes to the transport's `Runtime`. An rpc
method whose field is nil is not served.

```go
orders := &Orders{store: store, audit: audit}
grpcmesh.Register(shop.OrderService{Place: orders.Place, Placed: orders.Placed})
```

The registry keeps every endpoint and subscriber registered with it. If two
registered services serve the same `Target`, the transport is given two
endpoints or subscribers for it.

## Constructing and Starting an RPCRuntime

`NewRPCRuntime(transport, deploymentGroup, cfg, newRuntime, opts...)` builds
the runtime for one transport and one deployment group.

It takes the transport's `Client` from `DefaultTransportRouter`. It takes the
`Endpoints` and `Subscribers` whose `Targets` carry `deploymentGroup` from
`DefaultRegistry`. It copies `cfg` and sets `deployment_group` in the copy to
`deploymentGroup`, overwriting any value `cfg` has, and leaves `cfg` itself
unchanged. It then calls `newRuntime` with the client, the copied
configuration, and the endpoints and subscribers, so the transport's runtime
is built in the constructor and `TRuntime()` is set before `Start`.

`newRuntime` is a `NewRuntimeFunc`, `func(mesh.Client, mesh.Config,
[]mesh.Endpoint, []mesh.Subscriber) (mesh.Runtime, error)`, that builds the
transport's `mesh.Runtime`. When the transport's runtime constructor takes the
transport's own client type, `newRuntime` asserts the `mesh.Client` the router
passes to it, which is the one added under the transport's name.

Services registered after the constructor has run are not served.

```go
rt, err := grpcmesh.NewRPCRuntime("mem", "shop", mesh.Config{}, newRuntime)
if err != nil { /* a library error, or the error newRuntime returns */ }
if err := rt.Start(ctx); err != nil { /* the transport's error */ }

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
<-stop

drain, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = rt.Stop(drain)
```

`NewRPCRuntime` returns a library error when `newRuntime` is nil, when the
transport was not added to the router, or when a given `Target` is outside the
runtime. These are listed under [Library Errors](mesherror.md#library-errors).
The error `newRuntime` returns is passed through unchanged.

`WithEndpoints` and `WithSubscribers` give the runtime a list to serve in
place of the registry's list of that kind, so a process can serve only part of
a deployment group. `WithEndpoints()` or `WithSubscribers()` with no arguments
serves none of that kind. Every `Target` in a given list must carry the
runtime's `deployment_group` and `transport`.

```go
rt, err := grpcmesh.NewRPCRuntime("mem", "shop", mesh.Config{}, newRuntime,
    grpcmesh.WithEndpoints(shop.OrderService{Place: orders.Place}.Endpoints()...))
```

`RPCRuntime` implements `mesh.Runtime`. `Start`, `Stop`, `Running`, and
`Client` delegate to the underlying transport runtime, which `TRuntime()`
returns.

`Stop` closes the client. The transport documents what its runtime does,
including what `Stop` returns and how it treats a subscriber handler that
returns an error.

`Transport()` and `DeploymentGroup()` return the transport and deployment group
the `RPCRuntime` was built for.
