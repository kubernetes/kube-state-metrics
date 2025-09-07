#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &>/dev/null && pwd )
PROJECT_ROOT=$( cd -- "${SCRIPT_DIR}/../../.." &>/dev/null && pwd )

INPUT_DIR="${SCRIPT_DIR}/inputs"
OUTPUT_DIR="${SCRIPT_DIR}/outputs"
mkdir -p "${OUTPUT_DIR}"

# Backup root data.yaml once
ROOT_DATA="${PROJECT_ROOT}/data.yaml"
TMP_DATA=$(mktemp)
cp "${ROOT_DATA}" "${TMP_DATA}"

for input_file in "${INPUT_DIR}"/*.yaml; do
    base=$(basename "${input_file}" .yaml)
    echo "Processing ${base}.yaml ..."

    cp "${input_file}" "${ROOT_DATA}"
    "${PROJECT_ROOT}/scripts/update-data-yaml.sh" v2.18.0
    cp "${ROOT_DATA}" "${OUTPUT_DIR}/${base}.out.yaml"
done

# Restore original data.yaml
mv "${TMP_DATA}" "${ROOT_DATA}"

echo "All done. Results in ${OUTPUT_DIR}"
