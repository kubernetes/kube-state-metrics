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
	"encoding/json"
	"reflect"
	"testing"

	"k8s.io/kube-state-metrics/v2/pkg/metric"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

func TestNewCustomResourceMetrics(t *testing.T) {

	tests := []struct {
		r          Resource
		wantErr    bool
		wantResult *customResourceMetrics
		name       string
	}{
		{
			// https://github.com/kubernetes/kube-state-metrics/issues/1886
			name: "cr metric with dynamic metric type",
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
					CommonLabels: map[string]string{
						"hello": "world",
					},
				},
				Metrics: []Generator{
					{
						Name: "test_metrics",
						Help: "metrics for testing",
						Each: Metric{
							Type: metric.Info,
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
			wantResult: &customResourceMetrics{
				MetricNamePrefix: "kube_customresource",
				GroupVersionKind: schema.GroupVersionKind{
					Group:   "apps",
					Version: "v1",
					Kind:    "Deployment",
				},
				ResourceName: "deployments",
				Families: []compiledFamily{
					{
						Name: "kube_customresource_test_metrics",
						Help: "metrics for testing",
						Each: &compiledInfo{},
						Labels: map[string]string{
							"customresource_group":   "apps",
							"customresource_kind":    "Deployment",
							"customresource_version": "v1",
							"hello":                  "world",
						},
						LabelFromPath: map[string]valuePath{
							"name": mustCompilePath(t, "metadata", "name"),
						},
					},
				},
			},
		},
		{
			name: "cr metric with custom prefix",
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
					CommonLabels: map[string]string{
						"hello": "world",
					},
				},
				Metrics: []Generator{
					{
						Name: "test_metrics",
						Help: "metrics for testing",
						Each: Metric{
							Type: metric.Info,
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
				MetricNamePrefix: ptr.To("apps_deployment"),
			},
			wantErr: false,
			wantResult: &customResourceMetrics{
				MetricNamePrefix: "apps_deployment",
				GroupVersionKind: schema.GroupVersionKind{
					Group:   "apps",
					Version: "v1",
					Kind:    "Deployment",
				},
				ResourceName: "deployments",
				Families: []compiledFamily{
					{
						Name: "apps_deployment_test_metrics",
						Help: "metrics for testing",
						Each: &compiledInfo{},
						Labels: map[string]string{
							"customresource_group":   "apps",
							"customresource_kind":    "Deployment",
							"customresource_version": "v1",
							"hello":                  "world",
						},
						LabelFromPath: map[string]valuePath{
							"name": mustCompilePath(t, "metadata", "name"),
						},
					},
				},
			},
		},
		{
			name: "cr metric with custom prefix - expect error",
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
					CommonLabels: map[string]string{
						"hello": "world",
					},
				},
				Metrics: []Generator{
					{
						Name: "test_metrics",
						Help: "metrics for testing",
						Each: Metric{
							Type: metric.Info,
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
				MetricNamePrefix: ptr.To("apps_deployment"),
			},
			wantErr: true,
			wantResult: &customResourceMetrics{
				GroupVersionKind: schema.GroupVersionKind{
					Group:   "apps",
					Version: "v1",
					Kind:    "Deployment",
				},
				ResourceName: "deployments",
				Families: []compiledFamily{
					{
						Name: "apps_deployment_test_metrics",
						Help: "metrics for testing",
						Each: &compiledInfo{},
						Labels: map[string]string{
							"customresource_group":   "apps",
							"customresource_kind":    "Deployment",
							"customresource_version": "v1",
							"hello":                  "world",
						},
						LabelFromPath: map[string]valuePath{
							"name": mustCompilePath(t, "metadata", "name"),
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewCustomResourceMetrics(tt.r)
			if err != nil {
				t.Error(err.Error())
			}

			// convert to JSON for easier nil comparison
			ttWantJSON, _ := json.Marshal(tt.wantResult)
			customResourceMetricJSON, _ := json.Marshal(v.(*customResourceMetrics))

			if !tt.wantErr && !reflect.DeepEqual(ttWantJSON, customResourceMetricJSON) {
				t.Errorf("NewCustomResourceMetrics: error expected %v", tt.wantErr)
				t.Errorf("NewCustomResourceMetrics:\n %#v\n is not deep equal\n %#v", v, tt.wantResult)
			}

			if tt.wantErr && reflect.DeepEqual(ttWantJSON, customResourceMetricJSON) {
				t.Errorf("NewCustomResourceMetrics: error expected %v", tt.wantErr)
				t.Errorf("NewCustomResourceMetrics:\n %#v\n is not deep equal\n %#v", v, tt.wantResult)
			}
		})
	}
}
