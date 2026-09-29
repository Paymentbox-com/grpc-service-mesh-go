# MeshError

`MeshError` wraps a `google.rpc.Status` and implements `error`.

```go
me := grpcmesh.NewNotFoundError("no such item", &errdetails.ErrorInfo{Reason: "GONE"})
me.Code()    // grpcmesh.NotFound, which is code.Code_NOT_FOUND
me.Message() // "no such item"
me.Details() // []*anypb.Any
me.Proto()   // the wrapped *status.Status
me.Error()   // "NOT_FOUND: no such item"

back := grpcmesh.MeshErrorFromProto(st) // st is a *status.Status; the MeshError wraps it itself
```

There is one constructor per `google.rpc.Code`, `NewNotFoundError`,
`NewInvalidArgumentError`, `NewPermissionDeniedError`, and so on, each
`NewMeshError` with its code fixed. The codes themselves are exported as
constants of the `code.Code` type, `grpcmesh.OK`, `grpcmesh.NotFound`,
`grpcmesh.Internal`, and the rest, so handler and caller code compares codes
without importing the `code` package. `NewMeshError(c, msg, details...)` takes
any code, including one with no constant.

`NewMeshError` packs each detail message into `google.protobuf.Any`. A detail
that cannot be packed, such as a message whose string field holds invalid
UTF-8, makes the result an `INTERNAL` error whose message names the detail
type and the packing failure, and the code and message given are dropped, so
a reply never claims details it does not carry.

## Library Errors

These are the errors the package itself returns, apart from a `*MeshError`.

| Error | Returned when |
|---|---|
| `ErrUnknownTransport` | `DefaultTransportRouter.Client`, `NewRPCRuntime`, `Call`, or `Publish` looks up a transport name that was not added to the router. The error is wrapped with the name. |
| `ErrNoRuntimeConstructor` | `NewRPCRuntime` is given a nil `newRuntime`. |
| `ErrTargetOutsideRuntime` | A `Target` given through `WithEndpoints` or `WithSubscribers` carries a different `deployment_group` or `transport` from the runtime. The error is wrapped with the Target's segments and the key that differs. |
| `*MeshError` with `INTERNAL` | A payload fails to encode or decode. `Call` returns it when the reply does not decode, `Call` and `Publish` return it when the request does not encode, and a caller receives it when the serving handler could not decode the request or encode the response. |

Errors from the Service Mesh API, from the transport, and from `newRuntime` are
returned unchanged.
