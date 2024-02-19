#!/bin/bash
set -exuo pipefail

# Determine the OS to use the correct version of sed and awk.
# shellcheck disable=SC2209
SED=sed
# shellcheck disable=SC2209
AWK=awk
if [[ $(uname) == "Darwin" ]]; then
  # Check if gnu-sed and gawk are installed.
  if ! command -v gsed &> /dev/null; then
      echo "gnu-sed is not installed. Please install it using 'brew install gnu-sed'." >&2
      exit 1
  fi
  if ! command -v gawk &> /dev/null; then
      echo "gawk is not installed. Please install it using 'brew install gawk'." >&2
      exit 1
  fi
  AWK=gawk
  SED=gsed
fi

# Fetch the latest tag.
git fetch --tags
latest_tag=$(git describe --tags "$(git rev-list --tags --max-count=1)")

# Determine if it's a patch release or not (minor and major releases are handled the same way in the compatibility matrix).
if [[ $latest_tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    IFS='.' read -ra TAG <<< "$latest_tag"
    if [[ ${#TAG[@]} -eq 2 || ${#TAG[@]} -eq 3 && ${TAG[2]} -eq 0 ]]; then
        # Prep for a non-patch release.
        # shellcheck disable=SC2016
        main_client_go_version=$($AWK '/\| \*\*main\*\*/ {print $4}' README.md)
        $SED -i "/|\s*\*\*main\*\*\s*|\s*$main_client_go_version\s*|/i| \*\*$latest_tag\*\* | $main_client_go_version |" README.md
        # shellcheck disable=SC2016
        oldest_supported_client_go_version=$($AWK '/\| kube-state-metrics \| Kubernetes client-go Version \|/ {getline; getline; print $4; exit}' README.md)
        # Remove the first row (i.e., the oldest supported client-go version).
        $SED -i "/|\s*\*\*.*\*\*\s*|\s*$oldest_supported_client_go_version\s*|/d" README.md
    else
        # Prep for a patch release.
        minor_release="${TAG[0]}.${TAG[1]}"
        # Get the client-go version of the corresponding minor release row (that needs to be updated with the patch release version).
        # shellcheck disable=SC2016
        last_client_go_version=$($AWK '/\| kube-state-metrics \| Kubernetes client-go Version \|/ {getline; getline; getline; getline; getline; getline; print $4; exit}' README.md)
        # Update the row with the latest tag and client-go version.
        $SED -i "s/|\s*\*\*$minor_release.*\*\*\s*|\s*$last_client_go_version\s*|/| \*\*$latest_tag\*\* | $last_client_go_version |/" README.md
    fi
else
    echo -e "Bad tag format: $latest_tag, expected one of the following formats:\n
      * vMAJOR.MINOR (non-patch release)\n
      * vMAJOR.MINOR.PATCH (patch release)"
    exit 1
fi
