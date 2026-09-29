# Development

## Specification Protos

`meshoptions/` is package `meshoptions`, the `protoc-gen-go` output of the
specification's `mesh/options.proto`, whose `go_package` names it. It
registers the extensions `mesh.kind`, `mesh.consumer_group`,
`mesh.deployment_group`, `mesh.transport`, and `mesh.root_prefix`. It is compiled from the
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

## Tools and Tests

Tool versions are pinned in `mise.toml` and installed with `mise install`.
`just` lists the recipes. The ones used day to day:

| recipe | what it does |
|---|---|
| `just build` | compile everything |
| `just test` | run the suite with the race detector |
| `just proto` | run `just proto-spec` and `just proto-test` |
| `just proto-spec` | compile `mesh/options.proto` from grpc-service-mesh-api at `spec_tag` into `meshoptions/options.pb.go` |
| `just proto-test` | regenerate `internal/testproto/order.pb.go` with `protoc` and `protoc-gen-go` |
| `just check` | format check, vet, test, vulnerability scan, lint; what CI runs |

The tests run against `internal/memtransport`, an in-process transport
whose `Hub` builds `mesh.Client` values and `mesh.Runtime` values on
them. They deliver to each other and record what they were built with and what they sent. Nothing in this
repository needs a broker. Tests that need a real transport live outside it.

### Updating the Compiled Specification Protos

`meshoptions/options.pb.go` is the compiled form of `mesh/options.proto` in
the specification repository,
[grpc-service-mesh-api](https://github.com/Paymentbox-com/grpc-service-mesh-api),
at the tag `spec_tag` names in the `justfile`. It is never edited here.
`just proto-spec` makes a shallow clone of that tag in a temporary
directory, compiles `mesh/options.proto` from it with `protoc` and
`protoc-gen-go`, and removes the clone. CI runs `just proto` and fails when
the result differs from what is committed, so the compiled form always
matches the stated tag.

The specification's protobuf definitions maintain backwards compatibility, so a new specification tag only adds
options or enum values. To adopt one:

1. Set `spec_tag` in the `justfile` to the new tag.
2. Run `just proto`.
3. Review the diff under `meshoptions/`.
4. Run `just check`.
5. Bump the version, commit, and tag.

The extension numbers in `mesh/options.proto` are part of every definitions
project's compiled descriptors, and the specification never changes or
reuses one.
