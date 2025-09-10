#!/bin/bash
set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
REPO_ROOT=$( cd -- "${SCRIPT_DIR}/.." &> /dev/null && pwd )
DATA_FILE="${REPO_ROOT}/data.yaml"
NUM_RELEASES_TO_KEEP=5

check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: Required command '$1' not found. Please install it to continue." >&2
        exit 1
    fi
}

check_command "go"
check_command "gojq"
check_command "sort"

if [ -z "$1" ]; then
    echo "Usage: $0 <new_version>"
    echo "Example: $0 v2.17.0"
    exit 1
fi

if [ ! -f "${DATA_FILE}" ]; then
    echo "Error: Data file not found at ${DATA_FILE}" >&2
    exit 1
fi

NEW_VERSION_WITH_V=$1
CLEAN_NEW_VERSION=${NEW_VERSION_WITH_V#v}

echo "Starting update process for version ${NEW_VERSION_WITH_V}..."

echo "Checking k8s.io/client-go version from go.mod..."
GO_MOD_FILE="${REPO_ROOT}/go.mod"

if [ ! -f "${GO_MOD_FILE}" ]; then
    echo "Error: go.mod file not found at ${GO_MOD_FILE}" >&2
    exit 1
fi

CLIENT_GO_FULL_VERSION=$(go list -m -f '{{.Version}}' k8s.io/client-go)

if [ -z "$CLIENT_GO_FULL_VERSION" ]; then
    echo "Error: Could not find k8s.io/client-go version in go.mod." >&2
    exit 1
fi

K8S_MINOR=$(echo "${CLIENT_GO_FULL_VERSION}" | cut -d. -f2)

K8S_VERSION_FOR_NEW_RELEASE="1.${K8S_MINOR}"
echo "New release ${NEW_VERSION_WITH_V} will be mapped to Kubernetes (N-1): ${K8S_VERSION_FOR_NEW_RELEASE}"


JSON_DATA=$(cat "${DATA_FILE}" | gojq -r --yaml-input '.')

EXISTING_EXACT_MATCH=$(echo "${JSON_DATA}" | gojq -r --arg version "${NEW_VERSION_WITH_V}" --arg k8s "${K8S_VERSION_FOR_NEW_RELEASE}" '.compat[] | select(.version == $version and .kubernetes == $k8s) | .version // empty')

if [ -n "${EXISTING_EXACT_MATCH}" ]; then
    echo "Entry for ${NEW_VERSION_WITH_V} with Kubernetes ${K8S_VERSION_FOR_NEW_RELEASE} already exists. No changes needed."
    exit 0
fi

EXISTING_KSM_VERSION_ENTRY=$(echo "${JSON_DATA}" | gojq -r --arg version "${NEW_VERSION_WITH_V}" '.compat[] | select(.version == $version) | .version // empty')

if [ -n "${EXISTING_KSM_VERSION_ENTRY}" ]; then
    echo "Version ${NEW_VERSION_WITH_V} found with a different K8s mapping. Updating..."
    cat > "${DATA_FILE}" << EOF
# The purpose of this config is to keep all versions in a single place.
#
# Marks the latest release
version: "${CLEAN_NEW_VERSION}"

# List at max 5 releases here + the main branch
compat:
EOF
    
    
    echo "${JSON_DATA}" | gojq -r --arg version "${NEW_VERSION_WITH_V}" --arg k8s "${K8S_VERSION_FOR_NEW_RELEASE}" '
        .compat[] | select(.version != $version) | "  - kubernetes: \"" + .kubernetes + "\"\n    version: " + .version
    ' >> "${DATA_FILE}"
    
    echo "  - kubernetes: \"${K8S_VERSION_FOR_NEW_RELEASE}\"" >> "${DATA_FILE}"
    echo "    version: ${NEW_VERSION_WITH_V}" >> "${DATA_FILE}"
    
    echo "Successfully updated existing entry for ${NEW_VERSION_WITH_V}."
    echo "--- Final ${DATA_FILE} content ---"
    cat "${DATA_FILE}"
    exit 0
fi

echo "Adding new version ${NEW_VERSION_WITH_V} and pruning old releases..."

VERSIONS_LIST=$(echo "${JSON_DATA}" | gojq -r '.compat[] | select(.version != "main") | "\(.version)|\(.kubernetes)"' 2>/dev/null || true)
FULL_VERSIONS_LIST=$(printf "%s\n%s|%s" "${VERSIONS_LIST}" "${NEW_VERSION_WITH_V}" "${K8S_VERSION_FOR_NEW_RELEASE}")

SORTED_VERSIONS=$(echo "${FULL_VERSIONS_LIST}" | grep -v '^$' | sort -t'|' -k1,1 -Vr | head -n "${NUM_RELEASES_TO_KEEP}" | sort -t'|' -k1,1 -V)

cat > "${DATA_FILE}" << EOF
# The purpose of this config is to keep all versions in a single place.

# Marks the latest release
version: "${CLEAN_NEW_VERSION}"

# List at max 5 releases here + the main branch
compat:
EOF

while IFS='|' read -r version k8s_ver; do
    if [ -n "${version}" ]; then
        echo "  - version: \"${version}\"" >> "${DATA_FILE}"
        echo "    kubernetes: \"${k8s_ver}\"" >> "${DATA_FILE}"
    fi
done <<< "${SORTED_VERSIONS}"

echo "  - version: \"main\"" >> "${DATA_FILE}"
echo "    kubernetes: \"${K8S_VERSION_FOR_NEW_RELEASE}\"" >> "${DATA_FILE}"

echo "Successfully updated and pruned ${DATA_FILE}."
echo "New release (${NEW_VERSION_WITH_V}) is mapped to Kubernetes: ${K8S_VERSION_FOR_NEW_RELEASE}"
echo "Main branch is mapped to Kubernetes: ${K8S_VERSION_FOR_NEW_RELEASE}"
echo "--- Final ${DATA_FILE} content ---"
cat "${DATA_FILE}"