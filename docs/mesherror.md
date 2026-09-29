# MeshError

`MeshError` is the application failure that travels from a handler to its
caller. A `ROUTE` handler returns one to report a failure, as described under
[Returning an Error](handlers.md#returning-an-error), and the caller receives
it from the generated client method, as described under
[Calling](calling.md). It wraps a `google.rpc.Status`, the standard protobuf
error message, so a caller in any language decodes the same code, message, and
details.

`MeshError` implements `error`.

```go
me := grpcmesh.NewNotFoundError("no such item", &errdetails.ErrorInfo{Reason: "GONE"})
me.Code()    // grpcmesh.NotFound, which is code.Code_NOT_FOUND
me.Message() // "no such item"
me.Details() // []*anypb.Any
me.Proto()   // the wrapped *status.Status
me.Error()   // "NOT_FOUND: no such item"

back := grpcmesh.MeshErrorFromProto(st) // st is a *status.Status; the MeshError wraps it itself
```

## Constructing a MeshError

There is one constructor per `google.rpc.Code`, such as `NewNotFoundError`,
`NewInvalidArgumentError`, and `NewPermissionDeniedError`. Each is
`NewMeshError` with its code fixed, and takes the message and any detail
messages.

`NewMeshError(c, msg, details...)` takes any code, including one with no
constant.

`NewMeshError` packs each detail message into `google.protobuf.Any`. A detail
that cannot be packed, such as a message whose string field holds invalid
UTF-8, makes the result an `INTERNAL` error whose message names the detail type
and the packing failure. The code and message given are dropped in that case,
so a reply never claims details it does not carry.

## Reading a MeshError

The codes are exported as constants of the `code.Code` type, such as
`grpcmesh.OK`, `grpcmesh.NotFound`, and `grpcmesh.Internal`, so handler and
caller code compares codes without importing the `code` package. A caller
finds a `*MeshError` with `errors.As` and tells codes apart with `Code()`.

```go
var me *grpcmesh.MeshError
if errors.As(err, &me) && me.Code() == grpcmesh.NotFound {
    // the order does not exist
}
```

## Library Errors

These are the errors the package itself produces. A `*MeshError` that a
handler returns reaches the caller as described under
[Returning an Error](handlers.md#returning-an-error).

| Error | Returned when |
|---|---|
| `ErrUnknownTransport` | `DefaultTransportRouter.Client`, `NewRPCRuntime`, `Call`, or `Publish` looks up a transport name that was not added to the router. The error is wrapped with the name. |
| `ErrNoRuntimeConstructor` | `NewRPCRuntime` is given a nil `newRuntime`. |
| `ErrTargetOutsideRuntime` | A `Target` given through `WithEndpoints` or `WithSubscribers` carries a different `deployment_group` or `transport` from the runtime. The error is wrapped with the Target's segments and the key that differs. |
| `*MeshError` with `INTERNAL` | A payload fails to encode or decode. `Call` returns it when the reply does not decode, `Call` and `Publish` return it when the request does not encode, and a caller receives it when the serving handler could not decode the request or encode the response. |

Errors from the Service Mesh API, from the transport, and from `newRuntime` are
returned unchanged.
