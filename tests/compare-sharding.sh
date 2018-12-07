#!/usr/bin/bash

trap 'kill 0' SIGTERM

kubectl -n kube-system port-forward deployment/kube-state-metrics 8080:8080 &
kubectl -n kube-system port-forward pod/kube-state-metrics-0 8082:8080 &
kubectl -n kube-system port-forward pod/kube-state-metrics-1 8084:8080 &
kubectl -n kube-system port-forward pod/kube-state-metrics-2 8086:8080 &

sleep 3

RESULT_UNSHARDED="$(mktemp)"
RESULT_SHARDED_UNSORTED="$(mktemp)"
RESULT_SHARDED="$(mktemp)"

curl localhost:8080/metrics | grep -v "^#" | sort | tee "${RESULT_UNSHARDED}"
curl localhost:8082/metrics | grep -v "^#" | tee "${RESULT_SHARDED_UNSORTED}"
curl localhost:8084/metrics | grep -v "^#" | tee -a "${RESULT_SHARDED_UNSORTED}"
curl localhost:8086/metrics | grep -v "^#" | tee -a "${RESULT_SHARDED_UNSORTED}"

sort "${RESULT_SHARDED_UNSORTED}" | tee "${RESULT_SHARDED}"

diff <(echo "${RESULT_UNSHARDED}") <(echo "${RESULT_SHARDED}")

