# Calling


A [generated client](generated-code.md) is a package-level variable with one method per `rpc`
method. Each method takes the context, the request, and a metadata map. A
`ROUTE` method returns the decoded response, the reply's metadata, and an
error. A `TOPIC` method publishes and returns an error or nil.

```go
md := map[string]string{
    "Tenant": "acme",
    grpcmesh.OptionPrefix + nats.RequestTimeoutKey: "2s",
}

order, reply, err := shop.OrderClient.Place(ctx, &shop.Order{Item: proto.String("book")}, md)

var me *grpcmesh.MeshError
switch {
case err == nil:
    fmt.Println(order.GetId(), reply["Request-Id"])
case errors.As(err, &me):
    // me.Code(), me.Message(), me.Details(); a detail unpacks with UnmarshalTo
    for _, d := range me.Details() {
        var info errdetails.ErrorInfo
        if d.UnmarshalTo(&info) == nil {
            fmt.Println(info.GetReason())
        }
    }
case errors.Is(err, grpcmesh.ErrUnknownTransport):
    // the target's transport is not configured on the router
default:
    // a Service Mesh API or transport error, unchanged: mesh.ErrKindMismatch,
    // nats.ErrNoResponders, nats.ErrTimeout, ...
}

err = shop.OrderClient.Placed(ctx, order, md)
```

## Metadata and Transport Options

The keys of the metadata map that start with `Mesh-Option-`
(`grpcmesh.OptionPrefix`) are transport options. `Call` and `Publish` remove
them from the message metadata and pass each one to the transport's
`Request` or `Publish` options with the prefix removed, so
`Mesh-Option-request_timeout: 2s` reaches the transport as option
`request_timeout: 2s`. The match is exact and case-sensitive. Every other key
is message metadata. The caller's map is not modified. A nil map sends a
message whose only metadata is `Content-Type`, with no options.

Every message the package sends carries `Content-Type: application/x-protobuf`,
and a `Content-Type` in the metadata map is overwritten.

## Reply Metadata

The map a `ROUTE` method returns is a copy of the reply's metadata, on a
successful reply and on a `*MeshError` reply. It is nil when no reply
arrived, such as on a transport error. Reply metadata never carries keys that
start with `Mesh-Option-`. The serving side drops them from what a handler
sets with `SetReplyMetadata`, and the calling side drops them from the reply
it receives.

## Reply Decoding

`Call` reads `Grpc-Status` on the reply before the payload. When it is set,
the payload is decoded as `google.rpc.Status` and returned as a `*MeshError`;
the code is the one inside the `Status`. When it is not set, the payload is
decoded as the response type. A payload that does not decode, and a request
that does not encode, return an `INTERNAL` `*MeshError`. Errors from the
router, the Service Mesh API, and the transport are returned unchanged.
