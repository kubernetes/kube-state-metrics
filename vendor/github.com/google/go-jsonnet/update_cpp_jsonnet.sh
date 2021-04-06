#!/bin/bash

# Updates cpp-jsonnet repo and regenerates dependent files

set -e
set -x

cd cpp-jsonnet
git checkout master
git pull
cd ..
go run cmd/dumpstdlibast/dumpstdlibast.go cpp-jsonnet/stdlib/std.jsonnet > astgen/stdast.go

set +x
echo
echo -e "\033[1mUpdate completed. Please check if any tests are broken and fix any encountered issues.\033[0m"
