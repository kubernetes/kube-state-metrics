/*
Copyright 2024 The Kubernetes Authors All rights reserved.

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

package store

// TestStableMetrics verifies the golden list of stable metrics has not changed.
//
// It collects all FamilyGenerators from every store resource, filters for
// StabilityLevel == STABLE, and compares the sorted list to a golden YAML file.
// The fixture uses BETA as the serialized stability level because KSM stable
// metrics are consumed as beta metrics in k/k's stable metrics tooling.
//
// Workflow:
//   - To verify (CI presubmit):  go test ./internal/store/ -run TestStableMetrics
//   - To regenerate golden file: UPDATE_STABLE_METRICS=true go test ./internal/store/ -run TestStableMetrics
//     or equivalently:           hack/update-stable-metrics.sh

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	basemetrics "k8s.io/component-base/metrics"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

// stableMetricEntry matches the schema in the k/k stable-metrics-list.yaml
// golden file format for easy cross-project comparison.
type stableMetricEntry struct {
	Name           string    `json:"name" yaml:"name"`
	Help           string    `json:"help" yaml:"help"`
	Type           string    `json:"type" yaml:"type"`
	StabilityLevel string    `json:"stabilityLevel" yaml:"stabilityLevel"`
	Labels         []string  `json:"labels,omitempty" yaml:"labels,omitempty"`
	Buckets        []float64 `json:"buckets,omitempty" yaml:"buckets,omitempty"`
}

// k8sMetricType maps KSM's lowercase OpenMetrics type names to the capitalized
// type names used by k/k's stable-metrics-list.yaml format (Counter, Gauge,
// Histogram, Summary). KSM's info and stateset metrics are exposed on the wire
// as gauges, so they map to "Gauge".
var k8sMetricType = map[metric.Type]string{
	metric.Gauge:    "Gauge",
	metric.Counter:  "Counter",
	metric.Info:     "Gauge",
	metric.StateSet: "Gauge",
}

const goldenFile = "testdata/stable-metrics-list.yaml"

func TestStableMetrics(t *testing.T) {
	// Locate testdata/ relative to this source file so the test works when
	// run from any working directory.
	_, thisFile, _, _ := runtime.Caller(0)
	goldenPath := filepath.Join(filepath.Dir(thisFile), goldenFile)
	updateGolden := os.Getenv("UPDATE_STABLE_METRICS") == "true"

	raw, err := os.ReadFile(goldenPath) //nolint:gosec // G304: goldenPath is derived from this source file's directory and a constant testdata path.
	if err != nil {
		if updateGolden && os.IsNotExist(err) {
			raw = nil
		} else {
			t.Fatalf("Cannot read golden file %s: %v\n"+
				"Run hack/update-stable-metrics.sh to generate it.", goldenPath, err)
		}
	}

	var golden []stableMetricEntry
	if len(raw) > 0 {
		if err := yaml.Unmarshal(raw, &golden); err != nil {
			t.Fatalf("Cannot parse golden file: %v", err)
		}
	}

	entries := collectStableMetrics(t)

	if updateGolden {
		writeGolden(t, goldenPath, entries)
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	if len(raw) == 0 {
		t.Fatalf("Cannot read golden file %s: %v\n"+
			"Run hack/update-stable-metrics.sh to generate it.", goldenPath, os.ErrNotExist)
	}

	current, err := yaml.Marshal(entries)
	if err != nil {
		t.Fatalf("Cannot marshal current metrics: %v", err)
	}
	expected, err := yaml.Marshal(golden)
	if err != nil {
		t.Fatalf("Cannot marshal golden metrics: %v", err)
	}

	if diff := cmp.Diff(string(expected), string(current)); diff != "" {
		t.Errorf("Stable metrics list has changed.\n\n"+
			"Run hack/update-stable-metrics.sh to update the golden file if this is intentional.\n\n"+
			"diff (- expected, + current):\n%s",
			diff)
	}
}

// collectStableMetrics gathers all FamilyGenerators with StabilityLevel == STABLE
// across every store resource and returns them sorted by metric name.
func collectStableMetrics(t *testing.T) []stableMetricEntry {
	t.Helper()
	empty := []string{} // allow-lists don't affect stability metadata

	var all []generator.FamilyGenerator
	all = append(all, csrMetricFamilies(empty, empty)...)
	all = append(all, clusterRoleMetricFamilies(empty, empty)...)
	all = append(all, clusterRoleBindingMetricFamilies(empty, empty)...)
	all = append(all, configMapMetricFamilies(empty, empty)...)
	all = append(all, cronJobMetricFamilies(empty, empty)...)
	all = append(all, daemonSetMetricFamilies(empty, empty)...)
	all = append(all, deploymentMetricFamilies(empty, empty)...)
	all = append(all, endpointMetricFamilies(empty, empty)...)
	all = append(all, endpointSliceMetricFamilies(empty, empty)...)
	all = append(all, hpaMetricFamilies(empty, empty)...)
	all = append(all, ingressMetricFamilies(empty, empty)...)
	all = append(all, ingressClassMetricFamilies(empty, empty)...)
	all = append(all, jobMetricFamilies(empty, empty)...)
	all = append(all, leaseMetricFamilies...)
	all = append(all, limitRangeMetricFamilies...)
	all = append(all, mutatingWebhookConfigurationMetricFamilies...)
	all = append(all, namespaceMetricFamilies(empty, empty)...)
	all = append(all, networkPolicyMetricFamilies(empty, empty)...)
	all = append(all, nodeMetricFamilies(empty, empty)...)
	all = append(all, persistentVolumeMetricFamilies(empty, empty)...)
	all = append(all, persistentVolumeClaimMetricFamilies(empty, empty)...)
	all = append(all, podMetricFamilies(empty, empty)...)
	all = append(all, podDisruptionBudgetMetricFamilies(empty, empty)...)
	all = append(all, replicaSetMetricFamilies(empty, empty)...)
	all = append(all, replicationControllerMetricFamilies...)
	all = append(all, resourceQuotaMetricFamilies(empty, empty)...)
	all = append(all, roleMetricFamilies(empty, empty)...)
	all = append(all, roleBindingMetricFamilies(empty, empty)...)
	all = append(all, secretMetricFamilies(empty, empty)...)
	all = append(all, serviceMetricFamilies(empty, empty)...)
	all = append(all, serviceAccountMetricFamilies(empty, empty)...)
	all = append(all, statefulSetMetricFamilies(empty, empty)...)
	all = append(all, storageClassMetricFamilies(empty, empty)...)
	all = append(all, validatingWebhookConfigurationMetricFamilies...)
	all = append(all, volumeAttachmentMetricFamilies...)

	var stable []stableMetricEntry
	for _, fg := range all {
		if fg.StabilityLevel != basemetrics.STABLE {
			continue
		}
		// Strip the "[STABLE] " annotation prefix that KSM prepends to help text
		// in the rendered header; we store the clean help text in the golden file.
		help := strings.TrimPrefix(fg.Help, "[STABLE] ")
		typeName, ok := k8sMetricType[fg.Type]
		if !ok {
			t.Errorf("metric %q has unknown type %q; add it to k8sMetricType", fg.Name, fg.Type)
			typeName = string(fg.Type)
		}
		if fg.Labels == nil {
			t.Errorf("STABLE metric %q is missing an explicit label schema. Use NewFamilyGeneratorWithLabels to define its labels.", fg.Name)
			continue
		}
		stable = append(stable, stableMetricEntry{
			Name:           fg.Name,
			Help:           help,
			Type:           typeName,
			StabilityLevel: string(basemetrics.BETA),
			Labels:         fg.Labels,
		})
	}

	sort.Slice(stable, func(i, j int) bool {
		return stable[i].Name < stable[j].Name
	})

	// Detect duplicates (same metric name from multiple stores would be a bug).
	for i := 1; i < len(stable); i++ {
		if stable[i].Name == stable[i-1].Name {
			t.Errorf("Duplicate stable metric name: %s", stable[i].Name)
		}
	}

	return stable
}

func writeGolden(t *testing.T, path string, entries []stableMetricEntry) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatalf("Cannot create testdata dir: %v", err)
	}
	raw, err := yaml.Marshal(entries)
	if err != nil {
		t.Fatalf("Cannot marshal entries: %v", err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("Cannot write golden file: %v", err)
	}
}
