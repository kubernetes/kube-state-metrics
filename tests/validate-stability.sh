#!/usr/bin/env bash

set -o errexit

temp_file=$(mktemp)
metric_file='internal/store/pod.go'

go run \
		"tests/stablemetrics/main.go" \
		"tests/stablemetrics/decode_metric.go" \
		"tests/stablemetrics/find_stable_metric.go" \
		"tests/stablemetrics/error.go" \
		"tests/stablemetrics/metric.go" \
		-- \
		"${metric_file}" >"${temp_file}"


if diff -u "tests/stablemetrics/testdata/test-stable-metrics-list.yaml" "$temp_file"; then
	echo -e "PASS metrics stability verification"
fi
