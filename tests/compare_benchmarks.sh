#!/usr/bin/env bash
# exit immediately when a command fails
set -e
# only exit with zero if all commands of the pipeline exit successfully
set -o pipefail
# error on unset variables
set -u

[[ "$#" -le 2 ]] || echo "At least one argument required, $# provided."

REF_CURRENT="$(git rev-parse --abbrev-ref HEAD)"
REF_TO_COMPARE="${1}"

COUNT=${2:-"1"}

echo "Running benchmarks ${COUNT} time(s)"

RESULT_CURRENT="$(mktemp)-${REF_CURRENT}"
RESULT_TO_COMPARE="$(mktemp)-${REF_TO_COMPARE}"

echo ""
echo "### Testing ${REF_CURRENT}"

go test -benchmem -run=NONE -bench=. -count="${COUNT}" ./... | tee "${RESULT_CURRENT}"

echo ""
echo "### Done testing ${REF_CURRENT}"

echo ""
echo "### Testing ${REF_TO_COMPARE}"

git checkout "${REF_TO_COMPARE}"

go test -benchmem -run=NONE -bench=. -count="${COUNT}" ./... | tee "${RESULT_TO_COMPARE}"

echo ""
echo "### Done testing ${REF_TO_COMPARE}"

git checkout -

echo ""
echo "### Result"
echo "old=${REF_TO_COMPARE} new=${REF_CURRENT}"

if [[ -z "${BENCHSTAT_OUTPUT_FILE}" ]]; then
	go tool golang.org/x/perf/cmd/benchstat "${REF_TO_COMPARE}=${RESULT_TO_COMPARE}" "${REF_CURRENT}=${RESULT_CURRENT}"
else
	go tool golang.org/x/perf/cmd/benchstat "${REF_TO_COMPARE}=${RESULT_TO_COMPARE}" "${REF_CURRENT}=${RESULT_CURRENT}" | tee -a "${BENCHSTAT_OUTPUT_FILE}"
fi
