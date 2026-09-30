# Public API

Every exported name in package `grpcmesh`.

| Name | Role |
|---|---|
| `DefaultTransportRouter` | The process `*TransportRouter`. |
| `AddTransport(name, client)` | A shortcut for `DefaultTransportRouter.AddTransport`. |
| `DefaultRegistry` | The process `*Registry`. |
| `Register(svc, opts...)` | A shortcut for `DefaultRegistry.Register`. |
| `TransportRouter` | Holds one `mesh.Client` per transport name. `NewTransportRouter()` builds an empty one. Its methods are `AddTransport(name, client)`, `Client(name)`, and `Close()`. |
| `Registry` | Holds the registered endpoints and subscribers. `NewRegistry()` builds an empty one. Its methods are `Register(svc, opts...)`, `Endpoints()`, and `Subscribers()`. |
| `RPCService` | The interface a generated service type implements, with `Endpoints()` and `Subscribers()`. |
| `NewRPCRuntime(transport, deploymentGroup, cfg, newRuntime)` | Builds an `*RPCRuntime`, which serves one transport and one deployment group. Its methods are `Start`, `Stop`, `Running`, `Client`, `TRuntime`, `Transport`, and `DeploymentGroup`. |
| `NewRuntimeFunc` | The type of `newRuntime`, `func(mesh.Client, mesh.Config, []mesh.Endpoint, []mesh.Subscriber) (mesh.Runtime, error)`. |
| `RegisterOption`, `WithConsumerGroup(target, group)` | Options for `Register` that set the consumer group of the service's endpoint or subscriber for a target. `Register` panics when a target is not one of the service's. |
| `IncomingMetadata(ctx)` | The inbound message metadata inside a handler, and nil outside one. |
| `SetReplyMetadata(ctx, md)` | Adds metadata to the reply of a `ROUTE` handler. |
| `MeshError` | Described under [MeshError](mesherror.md). `NewMeshError`, `MeshErrorFromProto`, and one `New<Code>Error` per code build one. Its methods are `Code`, `Message`, `Details`, `Proto`, and `Error`. |
| `OK`, `NotFound`, `Internal`, and the rest | The `google.rpc.Code` values, as `code.Code` constants. |
| `OptionPrefix` | `"Mesh-Option-"`, the prefix that marks a metadata key as a transport option. |
| `ContentTypeKey`, `ContentTypeProtobuf` | `"Content-Type"` and `"application/x-protobuf"`, the metadata every sent message carries. |
| `GrpcStatusKey` | `"Grpc-Status"`, the metadata key that marks a reply carrying a `google.rpc.Status`. |
| `TransportKey` | `"transport"`, the `Target` metadata key that names the transport. |
| `ErrUnknownTransport`, `ErrNoRuntimeConstructor` | Listed under [Library Errors](mesherror.md#library-errors). |

## Used by Generated Code

Generated code calls these. They are an implementation detail between the generator and this package, and applications do not call them.
[What Generated Code Relies On](generated-code.md#what-generated-code-relies-on) describes them.

| Name | Role |
|---|---|
| `NewEndpoint(target, consumerGroup, fn)` | Builds the `mesh.Endpoint` for a `ROUTE` handler. Generated code calls it. |
| `NewSubscriber(target, consumerGroup, fn)` | Builds the `mesh.Subscriber` for a `TOPIC` handler. Generated code calls it. |
| `Call(ctx, target, req, md)` | Sends a request to a `ROUTE` target and decodes the reply. Generated clients call it. |
| `Publish(ctx, target, req, md)` | Publishes a message to a `TOPIC` target. Generated clients call it. |
