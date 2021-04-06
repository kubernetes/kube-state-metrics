#!/usr/bin/env bash

set -e

FILTER=${1:-Builtin}

root_dir="$( cd "$( dirname "$0" )" && pwd )"

(
    cd "${root_dir}"
    mkdir -p ./builtin-benchmark-results
    rm -f ./builtin-benchmark-results/*
    echo "Running Before Test... (10s)"
    go test -bench="${FILTER}" -run=^$ -v -benchtime=10s -jsonnetPath=./jsonnet-old > ./builtin-benchmark-results/before.txt
    echo "Running After Test... (10s)"
    go test -bench="${FILTER}" -run=^$ -v -benchtime=10s -jsonnetPath=./jsonnet > ./builtin-benchmark-results/after.txt
    benchcmp ./builtin-benchmark-results/before.txt ./builtin-benchmark-results/after.txt
)
