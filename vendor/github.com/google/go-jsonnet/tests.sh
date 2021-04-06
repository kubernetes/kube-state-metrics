#!/bin/bash

set -e

PYTHON_COMMAND=${PYTHON_COMMAND:=python}
JSONNET_CPP_DIR=${JSONNET_CPP_DIR:=$PWD/cpp-jsonnet}

set -x

[ "$SKIP_GO_TESTS" == 1 ] || go test ./...

if [ "$SKIP_PYTHON_BINDINGS_TESTS" == 1 ]
then
    c-bindings-tests/build.sh
else
    c-bindings-tests/run.sh

    $PYTHON_COMMAND setup.py build --build-platlib .
    $PYTHON_COMMAND -m pytest python
fi

export IMPLEMENTATION=golang
export OVERRIDE_DIR="$PWD/testdata/cpp-tests-override/"

go build ./cmd/jsonnet
go build ./cmd/jsonnetfmt

export DISABLE_LIB_TESTS=true
export DISABLE_ERROR_TESTS=true
export JSONNETFMT_BIN="$PWD/jsonnetfmt"
export JSONNET_BIN="$PWD/jsonnet"

cd "$JSONNET_CPP_DIR"
exec ./tests.sh
