# Development

This page is for working on the library itself. It covers the compiled
specification protos the library ships, how to update them, and the recipes
and tests used during development.

## Specification Protos

`meshoptions/` is package `meshoptions`, the `protoc-gen-go` output of the
specification's `mesh/options.proto`, whose `go_package` names it. It registers
the extensions `mesh.kind`, `mesh.consumer_group`, `mesh.deployment_group`,
`mesh.transport`, and `mesh.root_prefix`. It is compiled from the
[grpc-service-mesh-api](https://github.com/Paymentbox-com/grpc-service-mesh-api)
tag named by `spec_tag` in the `justfile`.

The compiled forms of `google/rpc/*.proto` are the published ones in
`google.golang.org/genproto/googleapis/rpc`.

A definitions file that uses the mesh options imports `mesh/options.proto`.
The Go message file that `protoc-gen-go` writes from it then imports
`meshoptions`, so a definitions project's Go module requires this module.

A project that runs plain `protoc` itself takes the specification's files from
the directory that `grpc-service-mesh-gen proto-path` prints:

```sh
protoc \
  -I definitions \
  -I "$(grpc-service-mesh-gen proto-path)" \
  --go_out=lib/go --go_opt=paths=source_relative \
  $(find definitions -name '*.proto')
```

Only files under `definitions/` are listed. The specification's files are
only on the include path. A project that runs this command itself runs the
generator with `--mesh-only`, which writes the mesh code and skips compiling
the message types.

### Updating the Compiled Specification Protos

`meshoptions/options.pb.go` is the compiled form of `mesh/options.proto` in the
specification repository,
[grpc-service-mesh-api](https://github.com/Paymentbox-com/grpc-service-mesh-api),
at the tag `spec_tag` names in the `justfile`. It is never edited here. `just
proto-spec` makes a shallow clone of that tag in a temporary directory,
compiles `mesh/options.proto` from it with `protoc` and `protoc-gen-go`, and
removes the clone. CI runs `just proto` and fails when the result differs from
what is committed, so the compiled form always matches the stated tag.

The specification's protobuf definitions maintain backwards compatibility, so a
new specification tag only adds options or enum values. To adopt one:

1. Set `spec_tag` in the `justfile` to the new tag.
2. Run `just proto`.
3. Review the diff under `meshoptions/`.
4. Run `just check`.
5. Run `just bump patch`, `just bump minor`, or `just bump major`, which commits
   the new `VERSION`, then push `master`, wait for CI to pass, and run `just release`.

The extension numbers in `mesh/options.proto` are part of every definitions
project's compiled descriptors, and the specification never changes or
reuses one.

## Tools and Tests

```
mise install
just check      # format check, vet, test, vulnerability scan, lint
```

Tool versions are pinned in `mise.toml`. `just` with no arguments lists the
recipes.

| Recipe | What it does |
|---|---|
| `just build` | Compiles everything. |
| `just test` | Runs the test suite with the race detector. |
| `just proto` | Runs `just proto-spec` and `just proto-test`. |
| `just proto-spec` | Compiles `mesh/options.proto` from grpc-service-mesh-api at `spec_tag` into `meshoptions/options.pb.go`. |
| `just proto-test` | Regenerates `internal/testproto/` from `internal/testproto/definitions/` with `grpc-service-mesh-gen` at `gen_version`. |
| `just vet` | Runs `go vet`. |
| `just fmt` | Formats the code in place with `gofmt`. |
| `just tidy` | Reconciles `go.mod` and `go.sum`. |
| `just vuln` | Reports reachable vulnerabilities with `govulncheck`. |
| `just lint` | Reports lint findings with `golangci-lint`. |
| `just check` | Runs the format check, `vet`, `test`, `vuln`, and `lint`, in the order CI runs them. |
| `just release` | Tags the current commit with the version in `VERSION`, pushes the tag, and asks the Go module proxy to fetch it. It refuses a working tree with changes. |
| `just bump patch`, `just bump minor`, `just bump major` | Raises the version in `VERSION` by one step and commits that file alone. A minor bump resets the patch number, and a major bump resets both. |

A Go version is released by its tag alone. Nothing is built or uploaded, and
the proxy fetch only makes the new version resolve for others right away.

The tests run against `internal/memtransport`, an in-process transport. Its
`Hub` builds clients, and runtimes on those clients, that deliver messages to
each other in memory. Each client and runtime records what it was built with
and what it sent, so nothing in this repository needs a broker.

`internal/testproto/` holds the reference output shown under
[Generated Code](generated-code.md), generated from the definitions in
`internal/testproto/definitions/`. `just proto-test` deletes the generated
files and runs `grpc-service-mesh-gen` at `gen_version` to write them again.
CI runs `just proto` and fails when the result differs from what is committed.
Adopting a new generator version means setting `gen_version`, running
`just proto`, and reviewing the diff.

`meshoptions/meshoptions_test.go` checks that the five extensions are
registered by their full names.

Tests that use the process router and registry call `freshSingletons`, which
installs an empty `DefaultTransportRouter` and `DefaultRegistry` for the test
and restores the previous ones when the test ends.
