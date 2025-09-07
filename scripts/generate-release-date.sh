#!/bin/bash
set -exuo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
REPO_ROOT=$( cd -- "${SCRIPT_DIR}/.." &> /dev/null && pwd )
CHANGELOG_FILE="${REPO_ROOT}/CHANGELOG.md"

if [ -z "$1" ]; then
    echo "Error: A version argument is required (e.g., v2.10.0)." >&2
    exit 1
fi
new_version=$1

# Determine the OS to use the correct version of sed.
# shellcheck disable=SC2209
SED=sed
if [[ $(uname) == "Darwin" ]]; then
  # Check if gnu-sed is installed.
  if ! command -v gsed &> /dev/null; then
      echo "gnu-sed is not installed. Please install it using 'brew install gnu-sed'." >&2
      exit 1
  fi
  SED=gsed
fi

# Extract content between "## Unreleased" and "## Released"
UNRELEASED_CONTENT=$($SED -n '/^## Unreleased/,/^## Released/{/^## Unreleased/d; /^## Released/d; p;}' "${CHANGELOG_FILE}")

# Clear the Unreleased section (remove everything between "## Unreleased" and "## Released")
$SED -i '/^## Unreleased/,/^## Released/{/^## Unreleased/!{/^## Released/!d;}}' "${CHANGELOG_FILE}"

# Add an empty line after the Unreleased section
$SED -i "/^## Unreleased/a\\"$'\n' "${CHANGELOG_FILE}"

# Add the new version section with date after "## Released"
if [[ -n "$UNRELEASED_CONTENT" ]]; then
    # Create a temporary file to avoid issues with quotes and special characters
    TEMP_FILE=$(mktemp)
    {
        echo ""
        echo "## $new_version / $(date +'%Y-%m-%d')"
        echo "$UNRELEASED_CONTENT"
    } > "$TEMP_FILE"
    
    $SED -i "/^## Released/r $TEMP_FILE" "${CHANGELOG_FILE}"
    
    rm "$TEMP_FILE"
else
    # If no content in Unreleased, just add the version header
    $SED -i "/^## Released/a\\
\\
## $new_version / $(date +'%Y-%m-%d')" "${CHANGELOG_FILE}"
fi

echo "CHANGELOG.md updated successfully. Moved unreleased content to $new_version section."
