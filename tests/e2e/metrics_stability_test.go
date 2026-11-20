/*
Copyright 2026 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"bytes"
	"testing"
)

func TestMetricsStability(t *testing.T) {
	if framework == nil || framework.KsmClient == nil {
		t.Fatal("e2e framework not initialized, run via TestMain with --ksm-http-metrics-url and --ksm-telemetry-url")
	}

	buf := &bytes.Buffer{}
	if err := framework.KsmClient.Metrics(buf); err != nil {
		t.Fatalf("failed to fetch metrics endpoint: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("metrics endpoint returned empty body")
	}

	ok, err := framework.KsmClient.IsHealthz()
	if err != nil {
		t.Fatalf("healthz check after metrics fetch failed: %v", err)
	}
	if !ok {
		t.Fatal("kube-state-metrics unhealthy after metrics fetch")
	}
}
