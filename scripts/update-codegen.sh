#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

set -x
set -e

go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.10.0
$(which controller-gen) object paths=./pkg/customresourcestate
$(which controller-gen) crd:crdVersions=v1 paths=./... output:crd:dir=./pkg/customresourcestate/apis/config

