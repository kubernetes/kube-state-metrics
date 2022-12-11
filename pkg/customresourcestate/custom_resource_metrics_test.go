/*
Copyright 2022 The Kubernetes Authors All rights reserved.

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

package customresourcestate

import (
	"testing"
)

func TestNewCustomResourceMetrics(t *testing.T) {
	tests := []struct {
		r       Resource
		wantErr bool
		name    string
	}{
		{
			// https://github.com/kubernetes/kube-state-metrics/issues/1886
			name: "dynamic metric type (not just hardcoded to gauge)",
			r: Resource{
				GroupVersionKind: GroupVersionKind{
					Group:   "apps",
					Version: "v1",
					Kind:    "Deployment",
				},
				Labels: Labels{
					LabelsFromPath: map[string][]string{
						"name": {"metadata", "name"},
					},
				},
				Metrics: []Generator{
					{
						Name: "test_metrics",
						Help: "metrics for testing",
						Each: Metric{
							Type: MetricTypeInfo,
							Info: &MetricInfo{
								MetricMeta: MetricMeta{
									Path: []string{
										"metadata",
										"annotations",
									},
								},
								LabelFromKey: "test",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewCustomResourceMetrics(tt.r)
			expectedError := v.(*customResourceMetrics).Families[0].Each.Type() != "info"
			if (err != nil) != tt.wantErr || expectedError {
				t.Errorf("NewCustomResourceMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
