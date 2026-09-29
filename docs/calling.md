# Calling

An application calls a service through its [generated
client](generated-code.md), a package-level variable with one method per rpc
method. Every call looks up the transport's `Client` through the router, so the
same calling code works in a process that serves and in one that only calls,
once the router holds a client for the transport, as described under
[Setup](setup.md).

Each method takes the context, the request, and a metadata map. A `ROUTE`
method returns the decoded response, the reply's metadata, and an error. A
`TOPIC` method publishes and returns an error or nil.

```go
md := map[string]string{
    "Tenant": "acme",
    grpcmesh.OptionPrefix + "request_timeout": "2s",
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
    // a Service Mesh API or transport error, unchanged, such as
    // mesh.ErrKindMismatch or the transport's timeout error
}

event := &shop.Order{Id: proto.String("o-1"), Item: proto.String("book")}
err = shop.OrderClient.Placed(ctx, event, md)
```

## Metadata and Transport Options

The keys of the metadata map that start with `Mesh-Option-`
(`grpcmesh.OptionPrefix`) are transport options. `Call` and `Publish` remove
them from the message metadata and pass each one to the transport's `Request`
or `Publish` options with the prefix removed. The match is exact and
case-sensitive.

For example, `Mesh-Option-request_timeout: 2s` reaches the transport as the
option `request_timeout: 2s`.

Every other key is sent as message metadata. The caller's map is not modified.
A nil map sends a message whose only metadata is `Content-Type`, with no
options.

## Reply Metadata

The map a `ROUTE` method returns is a copy of the reply's metadata, on a
successful reply and on a `*MeshError` reply. It is nil when no reply
arrived, such as on a transport error. Reply metadata never carries keys that
start with `Mesh-Option-`. The serving side drops them from what a handler
sets with `SetReplyMetadata`, and the calling side drops them from the reply
it receives.

## Reply Decoding

`Call` reads `Grpc-Status` on the reply before the payload. When it is set,
the payload is decoded as `google.rpc.Status` and returned as a `*MeshError`,
whose code is the one inside the `Status`. When it is not set, the payload is
decoded as the response type.

A payload that does not decode returns an `INTERNAL` (13) `*MeshError` whose
message names the expected type, such as
`reply does not decode as shop.Order: ...`. A request that does not encode
returns an `INTERNAL` `*MeshError` the same way, from `Call` and from
`Publish`.

A transport name the router does not hold returns `ErrUnknownTransport`, a
library error listed under [Library Errors](mesherror.md#library-errors).
Errors from the Service Mesh API and from the transport are returned
unchanged.

## Wire Format

Every message the package sends carries the metadata
`Content-Type: application/x-protobuf`. A `Content-Type` in the metadata map is
overwritten.

A reply that reports a `*MeshError` also carries `Grpc-Status`, the code as a
decimal integer string, and its payload is the encoded `google.rpc.Status`.
Every other payload is the message's binary protobuf encoding.
