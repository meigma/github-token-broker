set shell := ["bash", "-euo", "pipefail", "-c"]

default:
    @just --list

fmt:
    gofmt -w ./cmd ./internal

test:
    go test ./cmd/... ./internal/...

build:
    rm -rf build dist
    mkdir -p build dist
    GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -tags lambda.norpc -trimpath -ldflags="-s -w" -o build/bootstrap ./cmd/github-token-broker
    (cd build && zip -X ../dist/github-token-broker.zip bootstrap)

check:
    go test ./cmd/... ./internal/...
    just build

integration:
    go test -tags integration -count=1 ./internal/integration
