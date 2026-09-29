# Handlers


## Endpoint Handlers

A `ROUTE` handler receives the decoded request and returns the response or an
error. A `*MeshError` is the application failure the caller receives. Inbound
message metadata is read from the context.

```go
Place: func(ctx context.Context, req *shop.Order) (*shop.Order, error) {
    md := grpcmesh.IncomingMetadata(ctx) // the message metadata, nil outside a handler
    if md["Tenant"] == "" {
        return nil, grpcmesh.NewInvalidArgumentError("Tenant is required",
            &errdetails.ErrorInfo{Reason: "MISSING_TENANT", Domain: "shop"})
    }
    return &shop.Order{Id: proto.String(store.Place(req.GetItem())), Item: req.Item}, nil
}
```

## Returning an Error

A handler reports an application failure by returning a [`*MeshError`](mesherror.md) as its
error. The per-code constructors, such as `NewNotFoundError` and
`NewInvalidArgumentError`, take the message and any detail messages;
`NewMeshError(code, msg, details...)` takes any `google.rpc.Code`.

```go
Place: func(ctx context.Context, req *shop.Order) (*shop.Order, error) {
    if !store.InStock(req.GetItem()) {
        return nil, grpcmesh.NewNotFoundError("no such item",
            &errdetails.ErrorInfo{Reason: "ITEM_MISSING", Domain: "shop"})
    }
    return &shop.Order{Id: proto.String(store.Place(req.GetItem())), Item: req.Item}, nil
}
```

A `*MeshError` becomes a reply whose payload is the encoded
`google.rpc.Status` and whose metadata carries `Grpc-Status`, the code as a
decimal integer string, beside `Content-Type`. Any other error, and a panic,
is reported the same way as `UNKNOWN` (2) with the error's text or the panic
value as the message. A request that does not decode, and a response that
does not encode, are reported as `INTERNAL` (13) with the protobuf error's
text. The endpoint handler never returns an error to the transport, so a
transport's own handler-failure reporting is not involved.

## Reply Metadata

A `ROUTE` handler sets metadata on its reply with `SetReplyMetadata`. A later
call adds keys and overwrites the ones already set. The reply carries the
metadata on success and on a `*MeshError` reply, including the `UNKNOWN`
reply for another error or a panic. The package writes `Content-Type` and
`Grpc-Status` after the application's values, so its own values win, and a
successful reply carries no `Grpc-Status`. Outside a `ROUTE` handler, such as
in a `TOPIC` handler, `SetReplyMetadata` has no effect.

```go
Place: func(ctx context.Context, req *shop.Order) (*shop.Order, error) {
    grpcmesh.SetReplyMetadata(ctx, map[string]string{"Request-Id": "7"})
    return &shop.Order{Id: proto.String(store.Place(req.GetItem())), Item: req.Item}, nil
}
```

## Subscriber Handlers

A `TOPIC` handler receives the decoded message and returns an error or nil. Its
error goes to the transport as the `mesh.SubscriberHandler` error, unchanged,
and the transport documents what it does with it. A message that does not decode
returns the protobuf error the same way. Panics in a `TOPIC` handler are not recovered by
this package.
