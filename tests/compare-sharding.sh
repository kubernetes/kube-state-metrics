#!/usr/bin/bash

curl localhost:8080/metrics | grep -v "^#" | grep -v "kube_.*_labels" | sort > all.metrics
curl localhost:8082/metrics | grep -v "^#" | grep -v "kube_.*_labels" > all-sharded.metrics
curl localhost:8084/metrics | grep -v "^#" | grep -v "kube_.*_labels" >> all-sharded.metrics
sort all-sharded.metrics > all-sharded-sorted.metrics

diff all.metrics all-sharded-sorted.metrics

