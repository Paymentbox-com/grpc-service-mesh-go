# grpc-service-mesh-go

The Go implementation of the
[gRPC Service Mesh API](https://github.com/Paymentbox-com/grpc-service-mesh-api),
module `github.com/Paymentbox-com/grpc-service-mesh-go`, package
`grpcmesh`, imported as `github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh`.
The specification is the authority for everything this package does. It
carries protobuf messages over any transport that implements the
[Service Mesh API Specification](https://github.com/Paymentbox-com/service-mesh-api)
through the Go contract in
[service-mesh-go](https://github.com/Paymentbox-com/service-mesh-go).

The `grpcmesh` package holds the specification's non-generated types:
* `TransportRouter`
* `Registry`
* `RPCRuntime`
* `MeshError`

It also holds the generic helpers that generated code calls:
* `NewEndpoint`
* `NewSubscriber`
* `Call`
* `Publish`

Generated code comes from `grpc-service-mesh-gen` in the specification
repository and lives in a definitions project. Its dependencies are
`github.com/Paymentbox-com/service-mesh-go/mesh`, `google.golang.org/protobuf`,
and `google.golang.org/genproto/googleapis/rpc`.

## Usage

An application uses this library through the generated code that depends on it. The
[gRPC Service Mesh API](https://github.com/Paymentbox-com/grpc-service-mesh-api) describes how Go code is
generated from `.proto` files.

### Reference Examples

The examples in this README use the `shop.OrderService` from the specification: a `ROUTE` method `Place` and a
`TOPIC` method `Placed`, served over the transport named `nats` in deployment group `shop`. The generated package is
`shop`, and the generated per-transport maps are in `servicemaps`. The library's own reference copy of that generated
code is in `internal/testproto/`, described under [Generated Code](docs/generated-code.md).

## Documentation

- [Setup](docs/setup.md): configuring the `TransportRouter`, registering services, and running an `RPCRuntime`
- [Handlers](docs/handlers.md): writing `ROUTE` and `TOPIC` handlers, returning errors, and setting reply metadata
- [Calling](docs/calling.md): calling generated clients, metadata and transport options, and reply metadata
- [MeshError](docs/mesherror.md): constructing and reading `MeshError`
- [Generated Code](docs/generated-code.md): what the generator emits for Go, with the reference file
- [Development](docs/development.md): the specification protos, the recipes, and updating the compiled protos
