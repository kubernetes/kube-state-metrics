#!/usr/bin/env bash
# Copyright 2026 The Kubernetes Authors All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# update-metrics-documentation.sh generates the kube-state-metrics metrics
# reference page for k/website from the stable metrics list.
#
# Usage: hack/update-metrics-documentation.sh
#        hack/update-metrics-documentation.sh --output /path/to/kube-state-metrics.md

set -o errexit
set -o nounset
set -o pipefail

if ((BASH_VERSINFO[0] < 4 || (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] < 2))); then
  for bash in /opt/homebrew/bin/bash /usr/local/bin/bash; do
    if [[ -x "${bash}" ]]; then
      exec "${bash}" "$0" "$@"
    fi
  done
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KUBE_ROOT="${KUBE_ROOT:-"${REPO_ROOT}/../kubernetes"}"
WEBSITE_ROOT="${WEBSITE_ROOT:-"${REPO_ROOT}/../website"}"
INPUT="${REPO_ROOT}/internal/store/testdata/stable-metrics-list.yaml"
OUTPUT="${WEBSITE_ROOT}/content/en/docs/reference/instrumentation/kube-state-metrics.md"
VERSION=""

usage() {
  cat >&2 <<EOF
Usage:
  hack/update-metrics-documentation.sh
  hack/update-metrics-documentation.sh \\
    --output /path/to/kube-state-metrics.md \\
    --version 2.19.1

Environment:
  KUBE_ROOT     Path to a Kubernetes checkout with hack/tools/instrumentation.
  WEBSITE_ROOT  Path to a website checkout.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output)
      if [[ $# -lt 2 ]]; then
        echo "missing value for --output" >&2
        exit 2
      fi
      OUTPUT="$2"
      shift 2
      ;;
    --version)
      if [[ $# -lt 2 ]]; then
        echo "missing value for --version" >&2
        exit 2
      fi
      VERSION="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage
      exit 2
      ;;
  esac
done

if [[ "${OUTPUT}" != /* ]]; then
  OUTPUT="$(pwd)/${OUTPUT}"
fi

if [[ ! -f "${INPUT}" ]]; then
  echo "metrics list does not exist: ${INPUT}" >&2
  exit 1
fi
if [[ ! -f "${KUBE_ROOT}/hack/tools/instrumentation/stability-utils.sh" ]]; then
  echo "KUBE_ROOT does not contain Kubernetes instrumentation tools: ${KUBE_ROOT}" >&2
  echo "Set KUBE_ROOT to a Kubernetes checkout." >&2
  exit 1
fi

if [[ -z "${VERSION}" ]]; then
  VERSION=$(sed -nE 's/^version:[[:space:]]*"([^"]+)".*/\1/p' "${REPO_ROOT}/data.yaml" | head -n 1)
fi
VERSION="${VERSION:-unknown}"

# shellcheck source=/dev/null
source "${KUBE_ROOT}/hack/tools/instrumentation/stability-utils.sh"

kube::update::documentation::from_file \
  "${INPUT}" \
  "${OUTPUT}" \
  --version "${VERSION}" \
  --title "kube-state-metrics Metrics Reference" \
  --description "Details of the metric data that kube-state-metrics exports." \
  --product-name "kube-state-metrics" \
  --intro "This page details the metrics that kube-state-metrics exports. You can query the metrics endpoint using an HTTP scrape, and fetch the current metrics data in Prometheus format."
