#!/usr/bin/env bash
# Copyright 2024 The Kubernetes Authors All rights reserved.
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

# verify-stable-metrics.sh checks that the golden list of STABLE metrics has
# not changed without a corresponding update to the golden file.
#
# Usage:  hack/verify-stable-metrics.sh

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Verifying stable metrics list..."
cd "${REPO_ROOT}"
go test ./internal/store/ -run TestStableMetrics -count=1

echo ""
echo "PASS: stable metrics list is up to date."
