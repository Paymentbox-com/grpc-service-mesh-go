# Handlers

A handler is the application code that serves one rpc method. The generator
writes an `RPCService` struct for each proto service, with one function field
per rpc, and the struct serves nothing until those fields are set. The
application implements its handlers by setting a function on each field for an
rpc it serves. It then registers the struct value, as described under
[Registering a Service](setup.md#registering-a-service).

The package turns each set field into a Service Mesh API `Endpoint`, for a
`ROUTE` rpc, or `Subscriber`, for a `TOPIC` rpc. Each one decodes the inbound
payload, calls the function, and for a `ROUTE` rpc encodes the reply. The
sections below describe what a handler function receives, what it returns, and
how its failures reach the caller.

## Implementing an RPCService

The generated `shop.OrderService` has the `ROUTE` field `Place` and the `TOPIC`
field `Placed`, typed with the message each one takes and returns:

```go
type OrderService struct {
    Place  func(context.Context, *Order) (*Order, error)
    Placed func(context.Context, *Order) error
}
```

The application sets a function on each field it serves. Each function takes
the context and the decoded request. The functions are ordinary Go closures or
method values, so they can use whatever dependencies the application has in
scope.

```go
type Orders struct {
    store *Store
    audit *Audit
}

// ROUTE: returns a *shop.Order
func (o *Orders) Place(ctx context.Context, req *shop.Order) (*shop.Order, error) {
    return &shop.Order{Id: proto.String(o.store.Place(req.GetItem())), Item: req.Item}, nil
}

// TOPIC: returns an error or nil
func (o *Orders) Placed(ctx context.Context, ev *shop.Order) error {
    return o.audit.Record(ev)
}

orders := &Orders{store: store, audit: audit}
grpcmesh.Register(shop.OrderService{Place: orders.Place, Placed: orders.Placed})
```

A nil field is not served, so a process can implement only some of a
service's rpcs.

## Endpoint Handlers

A `ROUTE` handler receives the decoded request and returns the response or an
error. The inbound message metadata is on the context, read with
`IncomingMetadata`, described under [Message Metadata](#message-metadata).

```go
func (o *Orders) Place(ctx context.Context, req *shop.Order) (*shop.Order, error) {
    if grpcmesh.IncomingMetadata(ctx)["Tenant"] == "" {
        return nil, grpcmesh.NewInvalidArgumentError("Tenant is required",
            &errdetails.ErrorInfo{Reason: "MISSING_TENANT", Domain: "shop"})
    }
    return &shop.Order{Id: proto.String(o.store.Place(req.GetItem())), Item: req.Item}, nil
}
```

## Returning an Error

A handler reports an application failure by returning a [`*MeshError`](mesherror.md) as its
error. The per-code constructors, such as `NewNotFoundError` and
`NewInvalidArgumentError`, take the message and any detail messages.
`NewMeshError(code, msg, details...)` takes any `google.rpc.Code`.

```go
func (o *Orders) Place(ctx context.Context, req *shop.Order) (*shop.Order, error) {
    if !o.store.InStock(req.GetItem()) {
        return nil, grpcmesh.NewNotFoundError("no such item",
            &errdetails.ErrorInfo{Reason: "ITEM_MISSING", Domain: "shop"})
    }
    return &shop.Order{Id: proto.String(o.store.Place(req.GetItem())), Item: req.Item}, nil
}
```

A `*MeshError` becomes a reply whose payload is the encoded
`google.rpc.Status` and whose metadata carries `Grpc-Status`, the code as a
decimal integer string, beside `Content-Type`.

Any other error, and a panic, is reported the same way as `UNKNOWN` (2), with
the failure's text as the message.

A request that does not decode, and a response that does not encode, are
reported as `INTERNAL` (13), the code every decoding failure carries, and the
function is not called for a request that does not decode. The message names
the message type, such as `request does not decode as shop.Order: ...`.

In the Service Mesh API, an endpoint handler returns either a reply message or
an error, and a transport reports a returned error to the caller in its own
way. The endpoint that `NewEndpoint` builds always returns a reply message,
even for a failure. A caller therefore receives every handler failure as a
`*MeshError`, and never as the transport's own handler error.

## Reply Metadata

A `ROUTE` handler sets metadata on its reply with `SetReplyMetadata`. The reply
carries that metadata on success and on a `*MeshError` reply, including the
`UNKNOWN` reply for another error or a panic.

The package writes `Content-Type` and `Grpc-Status` after the application's
values, so its own values win, and a successful reply carries no
`Grpc-Status`. Keys that start with `Mesh-Option-` are removed from the reply's
metadata.

```go
func (o *Orders) Place(ctx context.Context, req *shop.Order) (*shop.Order, error) {
    if !o.store.InStock(req.GetItem()) {
        grpcmesh.SetReplyMetadata(ctx, map[string]string{"Retry-After": "30"})
        return nil, grpcmesh.NewNotFoundError("no such item")
    }
    grpcmesh.SetReplyMetadata(ctx, map[string]string{"Request-Id": "7"})
    return &shop.Order{Id: proto.String(o.store.Place(req.GetItem())), Item: req.Item}, nil
}
```

## Subscriber Handlers

A `TOPIC` handler receives the decoded message and returns an error or nil. A
`TOPIC` message has no reply, so `SetReplyMetadata` in a `TOPIC` handler has no
effect.

The subscriber that `NewSubscriber` builds returns the handler's error to the
transport unchanged, as the `mesh.SubscriberHandler` error. The transport
documents what it does with it.

A message that does not decode is handled the same way. The function is not
called, and the transport receives an error that names the message type, such
as `request does not decode as shop.Order: ...`.

Panics in a `TOPIC` handler are not recovered by this package.

## Message Metadata

Message metadata travels on the handler's context. The package puts the
inbound message's metadata on the context it passes to every handler, and two
functions read and write it.

| Function | Behavior |
|---|---|
| `IncomingMetadata(ctx)` | Returns the metadata of the message the handler is serving, in a `ROUTE` or a `TOPIC` handler. It returns nil outside a handler. |
| `SetReplyMetadata(ctx, md)` | Merges `md` into the metadata of the reply a `ROUTE` handler sends. A later call adds keys and overwrites the ones already set. The keys and values are copied, so the handler may reuse `md`. Outside a `ROUTE` handler it has no effect. |

The metadata a caller sends and the metadata it receives on a reply are plain
maps, described under [Calling](calling.md).
