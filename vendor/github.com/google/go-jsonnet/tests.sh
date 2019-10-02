#!/bin/bash

set -e

[ "$1" = "--skip-go-test" ] || go test ./...

c-bindings-tests/run.sh

export IMPLEMENTATION=golang

go build ./cmd/jsonnet

export DISABLE_LIB_TESTS=true
export DISABLE_FMT_TESTS=true
export DISABLE_ERROR_TESTS=true
export JSONNET_BIN="$PWD/jsonnet"

git submodule update --recursive cpp-jsonnet
cd cpp-jsonnet
exec ./tests.sh
