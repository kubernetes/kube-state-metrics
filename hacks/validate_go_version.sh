#!/bin/bash

set -e

which go > /dev/null
if [ $? -ne 0 ]; then
    echo "No golang environment found"
    exit 1
fi

# We should only compile kube-state-metrics with released golang versions.
go_version=$(go version|awk '{print $3}'|awk -Fgo '{print $2}')
if [[ $go_version = *"beta"* ]] || [[ $go_version = *"rc"* ]]; then
    echo "Using a beta/rc golang version: $go_version"
    exit 1
fi

major=$(echo $go_version|cut -d. -f1)
minor=$(echo $go_version|cut -d. -f2)
patch=$(echo $go_version|cut -d. -f3)

# We should only compile kube-state-metrics with golang whose version is greater than 1.9.5 or 1.10.1.
# For more info: https://github.com/kubernetes/kube-state-metrics/issues/416
if ([ ! -z "$patch" ] && [ $major -ge 1 -a $minor -eq 9 -a $patch -ge 5 ]) || ([ ! -z "$patch" ] && [ $major -ge 1 -a $minor -eq 10 -a $patch -ge 1 ]) || [ $major -ge 1 -a $minor -gt 10 ]; then
    echo "Compile kube-state-metrics with golang $go_version"
    exit 0
else
    echo "Kube-state-metrics should only be compiled with golang whose version is greater than 1.9.5 or 1.10.1"
    exit 1
fi

