# grpc-service-mesh-go

The Go library of the
[gRPC Service Mesh API](https://github.com/Paymentbox-com/grpc-service-mesh-api),
module `github.com/Paymentbox-com/grpc-service-mesh-go`, package
`grpcmesh`, imported as `github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh`.
The specification is the authority for everything this package does. It
carries protobuf messages over any transport that implements the
[Service Mesh API Specification](https://github.com/Paymentbox-com/service-mesh-api)
through the Go contract in
[service-mesh-go](https://github.com/Paymentbox-com/service-mesh-go). The
Ruby library is
[grpc-service-mesh-ruby](https://github.com/Paymentbox-com/grpc-service-mesh-ruby).

The package holds the specification's non-generated types, `TransportRouter`,
`Registry`, `RPCRuntime`, and `MeshError`, and the generic helpers that
generated code calls, `NewEndpoint`, `NewSubscriber`, `Call`, and `Publish`.
Generated code comes from `grpc-service-mesh-gen` in the specification
repository and lives in the definitions project. Its dependencies are
`github.com/Paymentbox-com/service-mesh-go/mesh`, `google.golang.org/protobuf`,
and `google.golang.org/genproto/googleapis/rpc`. It imports no transport.

Transports that implement the contract:
[service-mesh-nats-go](https://github.com/Paymentbox-com/service-mesh-nats-go)
and [service-mesh-nats-ruby](https://github.com/Paymentbox-com/service-mesh-nats-ruby)
over NATS, with the Ruby contract in
[service-mesh-ruby](https://github.com/Paymentbox-com/service-mesh-ruby).

## Usage

The examples use the `pbx.ApiKeyService` from the specification: a ROUTE
method `Search` and a TOPIC method `Created`, served over the transport
named `nats` in deployment group `pbx`. The generated package is `pbx` and
the generated per-transport maps are in `servicemaps`.

### Configuring the TransportRouter at boot

`grpcmesh.DefaultTransportRouter` is the process-wide router. The
application builds one transport `Client` per transport name its definitions
use, with the transport's configuration and the generated `ServiceMap` for
that transport, and adds a `Transport` entry holding that client, the
configuration, and the constructor of the transport's `Runtime`. The
transport package is a dependency of the application, not of this library.

```go
import (
    "github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
    "github.com/Paymentbox-com/service-mesh-go/mesh"
    "github.com/Paymentbox-com/service-mesh-nats-go/nats"

    "example.com/definitions/lib/go/servicemaps"
)

cfg := mesh.Config{nats.URLKey: os.Getenv("NATS_URL")}
c, err := nats.NewClient(cfg, servicemaps.Nats)
if err != nil { /* the transport's error */ }

grpcmesh.AddTransport("nats", grpcmesh.Transport{
    Client: c,
    Config: cfg,
    NewRuntime: func(c mesh.Client, cfg mesh.Config, e []mesh.Endpoint, s []mesh.Subscriber) (mesh.Runtime, error) {
        return nats.New(c.(*nats.Client), cfg, e, s)
    },
})
```

`nats.New` takes a `*nats.Client`, so the `NewRuntime` wrapper asserts the
`mesh.Client` the router passes back, which is the one the entry holds.

`router.Client(name)` returns the client the application added under
`name`, the one every generated client method sends through. A process that
only calls builds its clients, adds them, and runs `router.Close()` before
exit, which closes every entry's client and joins their errors, so the
transport flushes what it has buffered. The entries stay after `Close`, and a
`Client` call after it returns the closed client. An `RPCRuntime` for a
transport is built from the entry's client, and its `Stop` closes that client
as well.

`AddTransport` under a name already present replaces the entry. A transport
that was not added yields `ErrUnknownTransport`, wrapped with the name, from `Client`, `Get`,
`NewRPCRuntime`, `Call`, and `Publish`.

### Registering a service

A generated `RPCService` type has one function field per rpc method. The
application sets the ones it serves and registers the value in
`grpcmesh.DefaultRegistry`. A nil field is not served.

```go
grpcmesh.Register(pbx.ApiKeyService{
    Search: func(ctx context.Context, req *pbx.ApiKey) (*pbx.ApiKey, error) {
        key, ok := store.Find(req.GetFirstName())
        if !ok {
            return nil, grpcmesh.NewNotFoundError("no such key")
        }
        return key, nil
    },
    Created: func(ctx context.Context, ev *pbx.ApiKey) error {
        return audit.Record(ev)
    },
})
```

Every registered binding is kept, so two registrations of one `Target` hand
the transport two bindings for it.

### Constructing and starting an RPCRuntime

`NewRPCRuntime(transport, deploymentGroup)` takes from `DefaultRegistry` the
Endpoints and Subscribers whose Targets carry `deploymentGroup`, takes the
`Transport` entry from `DefaultTransportRouter`, and calls the entry's
`NewRuntime` with the entry's `Client`, the entry's `Config` plus
`deployment_group` set to `deploymentGroup`, and those bindings. The
transport's runtime is built in the constructor, so `Underlying()` is set
before `Start`. `Start`, `Stop`, and `Running` only delegate. The value passed
to `NewRPCRuntime` is the deployment group, so a `deployment_group` key in the
entry's `Config` is overwritten. Services registered after the constructor
has run are not served.

```go
rt, err := grpcmesh.NewRPCRuntime("nats", "pbx")
if err != nil { /* ErrUnknownTransport or the transport constructor's error */ }
if err := rt.Start(ctx); err != nil { /* the transport's error */ }

stop := make(chan os.Signal, 1)
signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
<-stop

drain, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
_ = rt.Stop(drain)
```

`RPCRuntime` implements `mesh.Runtime`. `Start`, `Stop`, `Running`, and
`Client` go to the transport's runtime, which `Underlying()` returns, so
`Client()` is the entry's client and `Stop` closes it after the drain.
`Transport()` and `DeploymentGroup()` return what the runtime was built for.

### Handlers

A ROUTE handler receives the decoded request and returns the response or an
error. A `*MeshError` is the application failure the caller receives. Inbound
message metadata is read from the context.

```go
Search: func(ctx context.Context, req *pbx.ApiKey) (*pbx.ApiKey, error) {
    md := grpcmesh.IncomingMetadata(ctx) // the message metadata, nil outside a handler
    if md["Tenant"] == "" {
        return nil, grpcmesh.NewInvalidArgumentError("Tenant is required",
            &errdetails.ErrorInfo{Reason: "MISSING_TENANT", Domain: "pbx"})
    }
    return store.Search(req)
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

A TOPIC handler receives the decoded message and returns an error or nil. Its
error goes to the transport as the `mesh.SubscriberHandler` error, unchanged,
and the transport documents what it does with it; the NATS transport logs
it. A message that does not decode returns the protobuf error the same way.
Panics in a TOPIC handler are not recovered by this package.

### Calling

A generated client is a package-level variable with one method per rpc
method. A ROUTE method returns the decoded response or an error; a TOPIC
method publishes and returns an error or nil. Message metadata and the
transport's per-call options ride on the context.

```go
ctx := grpcmesh.WithOutgoingMetadata(ctx, map[string]string{"Tenant": "acme"})
ctx = grpcmesh.WithTransportOptions(ctx, map[string]string{nats.RequestTimeoutKey: "2s"})

key, err := pbx.ApiKeyClient.Search(ctx, &pbx.ApiKey{FirstName: proto.String("ada")})

var me *grpcmesh.MeshError
switch {
case err == nil:
    fmt.Println(key.GetLastName())
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

err = pbx.ApiKeyClient.Created(ctx, key)
```

Every message the package sends carries `Content-Type: application/x-protobuf`;
a `Content-Type` in the outgoing metadata is overwritten. `Call` reads
`Grpc-Status` on the reply before the payload. When it is set, the payload is
decoded as `google.rpc.Status` and returned as a `*MeshError`; the code is the
one inside the `Status`. When it is not set, the payload is decoded as the
response type. A payload that does not decode, and a request that does not
encode, return an `INTERNAL` `*MeshError`. Errors from the router, the
Service Mesh API, and the transport are returned unchanged.

### MeshError

`MeshError` wraps a `google.rpc.Status` and implements `error`.

```go
me := grpcmesh.NewNotFoundError("no such key", &errdetails.ErrorInfo{Reason: "GONE"})
me.Code()    // grpcmesh.NotFound, which is code.Code_NOT_FOUND
me.Message() // "no such key"
me.Details() // []*anypb.Any
me.Proto()   // the wrapped *status.Status
me.Error()   // "NOT_FOUND: no such key"

back := grpcmesh.MeshErrorFromProto(st) // wraps st itself
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

## Generated code

The generator is `grpc-service-mesh-gen` from the specification repository:

```sh
go install github.com/Paymentbox-com/grpc-service-mesh-api/cmd/grpc-service-mesh-gen@v0.4.0
grpc-service-mesh-gen --definitions definitions --out lib --lang go,ruby
```

The generator reads `mesh/options.proto` from its own module version and
puts that directory on every `protoc` run.

For each service it emits a targets value, an
`RPCService` struct, and a client. This is the reference output for the
`ApiKeyService` in `internal/testproto`, which the tests use in place of a
generated package. A real directory substitutes its own package name,
segments, transport name, and deployment group, and takes its message types
from the standard `protoc-gen-go` output beside it.

```go
// Code generated by grpc-service-mesh-gen. DO NOT EDIT.
// source: testproto/api_key.proto

package testproto

import (
	"context"

	"github.com/Paymentbox-com/grpc-service-mesh-go/grpcmesh"
	"github.com/Paymentbox-com/service-mesh-go/mesh"
)

// ApiKeyTargets holds one Target per rpc method of ApiKeyService.
var ApiKeyTargets = struct {
	Search  mesh.Target
	Created mesh.Target
}{
	Search: mesh.Target{
		Segments: []string{"testproto", "ApiKeyService", "Search"},
		Kind:     mesh.KindRoute,
		Metadata: map[string]string{
			"deployment_group": "testproto",
			"transport":        "mem",
		},
	},
	Created: mesh.Target{
		Segments: []string{"testproto", "ApiKeyService", "Created"},
		Kind:     mesh.KindTopic,
		Metadata: map[string]string{
			"deployment_group": "testproto",
			"transport":        "mem",
			"consumer_group":   "audit",
		},
	},
}

// ApiKeyService is the RPCService for ApiKeyService. A nil field is not
// served.
type ApiKeyService struct {
	Search  func(context.Context, *ApiKey) (*ApiKey, error)
	Created func(context.Context, *ApiKey) error
}

// Endpoints returns the Endpoints of the ROUTE methods that are set.
func (s ApiKeyService) Endpoints() []mesh.Endpoint {
	var out []mesh.Endpoint
	if s.Search != nil {
		out = append(out, grpcmesh.NewEndpoint(ApiKeyTargets.Search, s.Search))
	}
	return out
}

// Subscribers returns the Subscribers of the TOPIC methods that are set.
func (s ApiKeyService) Subscribers() []mesh.Subscriber {
	var out []mesh.Subscriber
	if s.Created != nil {
		out = append(out, grpcmesh.NewSubscriber(ApiKeyTargets.Created, s.Created))
	}
	return out
}

type apiKeyClient struct{}

// ApiKeyClient calls ApiKeyService through grpcmesh.DefaultTransportRouter.
var ApiKeyClient apiKeyClient

// Search calls the ROUTE method Search.
func (apiKeyClient) Search(ctx context.Context, req *ApiKey) (*ApiKey, error) {
	return grpcmesh.Call[*ApiKey, *ApiKey](ctx, ApiKeyTargets.Search, req)
}

// Created publishes to the TOPIC method Created.
func (apiKeyClient) Created(ctx context.Context, req *ApiKey) error {
	return grpcmesh.Publish(ctx, ApiKeyTargets.Created, req)
}
```

The type parameters of `NewEndpoint`, `NewSubscriber`, `Call`, and `Publish`
are pointer types to generated messages. The package obtains a fresh message
for decoding from the type's zero value through
`zero.ProtoReflect().New().Interface()`, which works on a nil pointer to a
generated type and on nothing else.

The per-transport `ServiceMap`s are emitted into `servicemaps/servicemaps.go`
at the output root, one `mesh.ServiceMap` variable per transport named by
PascalCasing the transport string, such as `servicemaps.Nats`.

## Specification protos

`meshoptions/` is package `meshoptions`, the `protoc-gen-go` output of the
specification's `mesh/options.proto`, whose `go_package` names it. It
registers the extensions `mesh.kind`, `mesh.consumer_group`,
`mesh.deployment_group`, and `mesh.transport`. It is compiled from the
[grpc-service-mesh-api](https://github.com/Paymentbox-com/grpc-service-mesh-api)
tag named by `spec_tag` in the `justfile`.

The compiled forms of `google/rpc/*.proto` are the published ones in
`google.golang.org/genproto/googleapis/rpc`.

Every message file `protoc-gen-go` writes from a definitions file that
imports `mesh/options.proto` imports `meshoptions`, so a definitions
project's Go module requires this module. Plain `protoc` takes the
specification's files from the directory `grpc-service-mesh-gen proto-path`
prints:

```sh
protoc \
  -I definitions \
  -I "$(grpc-service-mesh-gen proto-path)" \
  --go_out=lib/go --go_opt=paths=source_relative \
  $(find definitions -name '*.proto')
```

Only files under `definitions/` are listed. The specification's files are
only on the include path. A project that runs this command itself runs the
generator with `--mesh-only`, which writes the mesh code and skips the
message runs.

## Development

Tool versions are pinned in `mise.toml` and installed with `mise install`.
`just` lists the recipes. The ones used day to day:

| recipe        | what it does                                                              |
|---------------|---------------------------------------------------------------------------|
| `just build`  | compile everything                                                        |
| `just test`   | run the suite with the race detector                                      |
| `just proto`  | run `just proto-spec` and `just proto-test`                                 |
| `just proto-spec` | compile `mesh/options.proto` from grpc-service-mesh-api at `spec_tag` into `meshoptions/options.pb.go` |
| `just proto-test` | regenerate `internal/testproto/api_key.pb.go` with `protoc` and `protoc-gen-go` |
| `just check`  | format check, vet, test, vulnerability scan, lint; what CI runs           |

The tests run against `internal/memtransport`, an in-process transport
whose `Hub` builds `mesh.Client` values and `mesh.Runtime` values on
them. They deliver to each other and record what they were built with and
what they sent. Nothing
in this repository needs a broker. Tests that need a transport live outside
it.

### Updating the compiled specification protos

`meshoptions/options.pb.go` is the compiled form of `mesh/options.proto` in
the specification repository,
[grpc-service-mesh-api](https://github.com/Paymentbox-com/grpc-service-mesh-api),
at the tag `spec_tag` names in the `justfile`. It is never edited here.
`just proto-spec` makes a shallow clone of that tag in a temporary
directory, compiles `mesh/options.proto` from it with `protoc` and
`protoc-gen-go`, and removes the clone. CI runs `just proto` and fails when
the result differs from what is committed, so the compiled form always
matches the stated tag.

`mesh/options.proto` changes only by adding, so a new specification tag only
adds options or enum values. To adopt one:

1. Set `spec_tag` in the `justfile` to the new tag.
2. Run `just proto`.
3. Review the diff under `meshoptions/`.
4. Run `just check`.
5. Bump the version, commit, and tag.

The extension numbers in `mesh/options.proto` are part of every definitions
project's compiled descriptors, and the specification never changes or
reuses one.

