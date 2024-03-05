#!/bin/bash
set -exuo pipefail

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

# Get the latest tag from the git repository.
latest_tag=$(git describe --tags --abbrev=0)

# Replace the first line of the CHANGELOG.md file with the expected metadata.
# NOTE: This is done to ensure the latest release header is consistent, and will replace the manual placeholder.
$SED -i "1s/.*/## v$latest_tag / $(date +'%Y-%m-%d')/" CHANGELOG.md
