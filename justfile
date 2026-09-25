# Task runner for grpc-service-mesh-go. Run `just` with no arguments to see the menu.
#
# Every Go recipe runs through `mise exec` so it uses the toolchain pinned in
# mise.toml without relying on the shell's mise activation. If you ever run just
# from a bare environment where `mise` is not on PATH, change this to
# "/opt/homebrew/bin/mise exec -- go".
go := "mise exec -- go"

# List all recipes
default:
    @just --list

# Compile everything
[group('build')]
build:
    {{go}} build ./...

# Run the full test suite with the race detector
[group('build')]
test:
    {{go}} test -race ./...

# The grpc-service-mesh-api tag whose mesh/options.proto meshoptions is compiled from
spec_tag := "v0.4.0"

# Regenerate meshoptions/options.pb.go from the specification at {{spec_tag}}
# and regenerate the test message code. protoc is found on PATH;
# protoc-gen-go comes from the pinned Go toolchain.
[group('build')]
proto: proto-spec proto-test

# Compile mesh/options.proto from a shallow clone of grpc-service-mesh-api at {{spec_tag}} into meshoptions/options.pb.go
[group('build')]
proto-spec:
    #!/usr/bin/env bash
    set -euo pipefail
    spec="$(mktemp -d)"
    trap 'rm -rf "$spec"' EXIT
    git -c advice.detachedHead=false clone --quiet --depth 1 --branch {{spec_tag}} https://github.com/Paymentbox-com/grpc-service-mesh-api "$spec"
    mise exec -- protoc --proto_path="$spec" --go_out=. --go_opt=module=github.com/Paymentbox-com/grpc-service-mesh-go "$spec/mesh/options.proto"

# Regenerate the test message code from internal/testproto/order.proto
[group('build')]
proto-test:
    mise exec -- protoc --proto_path=internal/testproto --go_out=internal/testproto --go_opt=paths=source_relative internal/testproto/order.proto

# Run go vet
[group('checks')]
vet:
    {{go}} vet ./...

# Format the code in place
[group('checks')]
fmt:
    gofmt -w .

# Fail if any file is not gofmt-formatted
[private]
[group('checks')]
fmt-check:
    test -z "$(gofmt -l .)"

# Reconcile go.mod and go.sum
[group('checks')]
tidy:
    {{go}} mod tidy

# Report reachable vulnerabilities (matches CI)
[group('checks')]
vuln:
    {{go}} run golang.org/x/vuln/cmd/govulncheck@latest ./...

# There is no .golangci.yml, so lint runs golangci-lint's default linters. CI
# installs golangci-lint through its GitHub action rather than with `go run`,
# so the two can differ by a release.

# Report lint findings (matches CI)
[group('checks')]
lint:
    {{go}} run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...

# Everything the pre-push gate checks, in the order CI runs them
[group('checks')]
check: fmt-check vet test vuln lint
