#!/usr/bin/env bash

set -o errexit

out_file='tests/stablemetrics/testdata/test-stable-metrics-list.yaml'
metric_file='internal/store/pod.go'

go run \
		"tests/stablemetrics/main.go" \
		"tests/stablemetrics/decode_metric.go" \
		"tests/stablemetrics/find_stable_metric.go" \
		"tests/stablemetrics/error.go" \
		"tests/stablemetrics/metric.go" \
		-- \
		"${metric_file}" >"${out_file}"


