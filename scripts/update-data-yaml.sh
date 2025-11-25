#!/bin/bash
set -euo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
REPO_ROOT=$( cd -- "${SCRIPT_DIR}/.." &> /dev/null && pwd )
DATA_FILE="${REPO_ROOT}/data.yaml"
NUM_RELEASES_TO_KEEP=5

GOJQ="go tool github.com/itchyny/gojq/cmd/gojq"

check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: Required command '$1' not found. Please install it to continue." >&2
        exit 1
    fi
}

check_command "go"
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
NEW_VERSION_WITHOUT_V=${NEW_VERSION_WITH_V#v}

echo "Starting update process for version ${NEW_VERSION_WITH_V}..."

echo "Checking k8s.io/client-go version from go.mod..."
GO_MOD="${REPO_ROOT}/go.mod"

if [ ! -f "${GO_MOD}" ]; then
    echo "Error: go.mod file not found at ${GO_MOD}" >&2
    exit 1
fi

CLIENT_GO_VERSION=$(go list -m -f '{{.Version}}' k8s.io/client-go)

if [ -z "$CLIENT_GO_VERSION" ]; then
    echo "Error: Could not find k8s.io/client-go version in go.mod." >&2
    exit 1
fi

K8S_MINOR=$(echo "${CLIENT_GO_VERSION}" | cut -d. -f2)

K8S_VERSION_FOR_NEW_RELEASE="1.${K8S_MINOR}"
echo "New release ${NEW_VERSION_WITH_V} will be mapped to Kubernetes: ${K8S_VERSION_FOR_NEW_RELEASE}"

# Convert YAML data file to JSON format
DATA_FILE_JSON_DATA=$(${GOJQ} -r --yaml-input '.' "${DATA_FILE}")

EXISTING_EXACT_MATCH=$(\
echo "${DATA_FILE_JSON_DATA}" |\
# Query for existing entry with same version and k8s mapping
${GOJQ} -r --arg version "${NEW_VERSION_WITH_V}" --arg k8s "${K8S_VERSION_FOR_NEW_RELEASE}" ".compat[] | select(.version == \$version and .kubernetes == \$k8s) | .version // empty"\
)

if [ -n "${EXISTING_EXACT_MATCH}" ]; then
    echo "Entry for ${NEW_VERSION_WITH_V} with Kubernetes ${K8S_VERSION_FOR_NEW_RELEASE} already exists. No changes needed."
    exit 0
fi

EXISTING_KSM_VERSION_ENTRY=$(\
echo "${DATA_FILE_JSON_DATA}" |\
# Check if KSM version already exists (with different k8s mapping)
${GOJQ} -r --arg version "${NEW_VERSION_WITH_V}" ".compat[] | select(.version == \$version) | .version // empty"\
)

if [ -n "${EXISTING_KSM_VERSION_ENTRY}" ]; then
    echo "Version ${NEW_VERSION_WITH_V} found with a different K8s mapping. Updating..."
    cat > "${DATA_FILE}" << EOF
# This configuration tracks the last five releases, and acts as the source of truth for those entries in the README.
#
# This marks the latest release.
version: "${NEW_VERSION_WITHOUT_V}"

# List KSM-to-K8s version mapping for the last five releases here, and the default branch.
compat:
EOF
    
    
    {
        echo "${DATA_FILE_JSON_DATA}" |\
        # Filter out the version being updated and format as YAML
        ${GOJQ} -r --arg version "${NEW_VERSION_WITH_V}" --arg k8s "${K8S_VERSION_FOR_NEW_RELEASE}" \
            ".compat[] | select(.version != \$version) | \"  - kubernetes: \\\"\" + .kubernetes + \"\\\"\\n    version: \" + .version"
        
        echo "  - kubernetes: \"${K8S_VERSION_FOR_NEW_RELEASE}\""
        echo "    version: ${NEW_VERSION_WITH_V}"
    } >> "${DATA_FILE}"
    
    echo "Successfully updated existing entry for ${NEW_VERSION_WITH_V}."
    echo "--- Final ${DATA_FILE} content ---"
    cat "${DATA_FILE}"
    exit 0
fi

echo "Adding new version ${NEW_VERSION_WITH_V} and pruning old releases..."

VERSIONS_LIST_WITHOUT_MAIN=$(\
echo "${DATA_FILE_JSON_DATA}" |\
# Extract version|kubernetes pairs, excluding main branch
${GOJQ} -r ".compat[] | select(.version != \"main\") | \"\(.version)|\(.kubernetes)\"" 2>/dev/null || true\
)

FULL_VERSIONS_LIST=$(printf "%s\n%s|%s" "${VERSIONS_LIST_WITHOUT_MAIN}" "${NEW_VERSION_WITH_V}" "${K8S_VERSION_FOR_NEW_RELEASE}")

SORTED_VERSIONS=$(\
echo "${FULL_VERSIONS_LIST}" |\
# Remove empty lines
grep -v '^$' |\
# Sort by version (ascending) using version sort
sort -t'|' -k1,1 -V |\
# Keep only the most recent N releases
tail -n "${NUM_RELEASES_TO_KEEP}"\
)

cat > "${DATA_FILE}" << EOF
# This configuration tracks the last five releases, and acts as the source of truth for those entries in the README.
#
# This marks the latest release.
version: "${NEW_VERSION_WITHOUT_V}"

# List KSM-to-K8s version mapping for the last five releases here, and the default branch.
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
