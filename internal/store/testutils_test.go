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

package store

import (
	"strings"
	"testing"

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestValidateFamilyLabels(t *testing.T) {
	tests := []struct {
		name        string
		metricName  string
		declared    []string
		emittedSets [][]string
		wantErr     string
	}{
		{
			name:       "exact static label set matches regardless of order",
			metricName: "kube_cronjob_info",
			declared:   []string{"namespace", "cronjob", "schedule"},
			emittedSets: [][]string{
				{"schedule", "namespace", "cronjob"},
			},
		},
		{
			name:       "missing declared static label fails",
			metricName: "kube_cronjob_info",
			declared:   []string{"namespace", "cronjob", "schedule"},
			emittedSets: [][]string{
				{"namespace", "cronjob"},
			},
			wantErr: `missing declared label "schedule"`,
		},
		{
			name:       "undeclared emitted label fails",
			metricName: "kube_cronjob_info",
			declared:   []string{"namespace", "cronjob"},
			emittedSets: [][]string{
				{"namespace", "cronjob", "extra"},
			},
			wantErr: `output undeclared label "extra"`,
		},
		{
			name:       "duplicate emitted label fails",
			metricName: "kube_cronjob_info",
			declared:   []string{"namespace", "cronjob"},
			emittedSets: [][]string{
				{"namespace", "cronjob", "namespace"},
			},
			wantErr: `output duplicate label "namespace"`,
		},
		{
			name:       "dynamic label placeholder allows zero or multiple dynamic labels",
			metricName: "kube_cronjob_labels",
			declared:   []string{"namespace", "cronjob", "label_CRONJOB_LABEL"},
			emittedSets: [][]string{
				{"namespace", "cronjob"},
				{"namespace", "cronjob", "label_app", "label_env"},
			},
		},
		{
			name:       "dynamic label metric still requires all static declared labels",
			metricName: "kube_cronjob_labels",
			declared:   []string{"namespace", "cronjob", "label_CRONJOB_LABEL"},
			emittedSets: [][]string{
				{"namespace", "label_app"},
			},
			wantErr: `missing declared label "cronjob"`,
		},
		{
			name:       "legacy variable label metric accepts allowed variants",
			metricName: "kube_job_status_failed",
			declared:   []string{"namespace", "job_name", "reason"},
			emittedSets: [][]string{
				{"namespace", "job_name"},
				{"namespace", "job_name", "reason"},
			},
		},
		{
			name:       "legacy variable label metric rejects unexpected label set",
			metricName: "kube_job_status_failed",
			declared:   []string{"namespace", "job_name", "reason"},
			emittedSets: [][]string{
				{"namespace"},
			},
			wantErr: "does not match any allowed set",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metrics := make([]*metric.Metric, 0, len(tc.emittedSets))
			for _, keys := range tc.emittedSets {
				metrics = append(metrics, &metric.Metric{
					LabelKeys: keys,
					Value:     1,
				})
			}
			fam := metric.Family{
				Name:    tc.metricName,
				Metrics: metrics,
			}
			gen := generator.FamilyGenerator{
				Name:           tc.metricName,
				StabilityLevel: basemetrics.STABLE,
				Labels:         tc.declared,
			}

			err := validateFamilyLabels(fam, gen)
			if tc.wantErr == "" && err != nil {
				t.Fatalf("validateFamilyLabels() unexpected error: %v", err)
			}
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("validateFamilyLabels() succeeded, want error containing %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("validateFamilyLabels() error = %q, want substring %q", err.Error(), tc.wantErr)
				}
			}
		})
	}
}
