/*
Copyright 2020 The Kubernetes Authors All rights reserved.

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

package generator

import (
	"reflect"
	"sort"
	"testing"

	"k8s.io/kube-state-metrics/v2/pkg/allow"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

func TestFilterMetricFamiliesLabels(t *testing.T) {
	tests := []struct {
		name             string
		allowLabels      allow.Labels
		familyGenerators []FamilyGenerator
		results          []FamilyGenerator
	}{
		{
			name:        "Returns all the metric's keys and values if not annotation/label metric by default",
			allowLabels: allow.Labels(map[string][]string{}),
			familyGenerators: []FamilyGenerator{
				{
					Name: "node_info",
					Help: "some help",
					Type: metric.Gauge,
					GenerateFunc: func(obj interface{}) *metric.Family {
						return &metric.Family{
							Metrics: []*metric.Metric{
								{
									LabelKeys:   []string{"one", "two", "three"},
									LabelValues: []string{"value-one", "value-two", "value-three"},
									Value:       1,
								},
							},
						}
					},
				},
			},
			results: []FamilyGenerator{
				{
					Name: "node_info",
					Help: "some help",
					Type: metric.Gauge,
					GenerateFunc: func(obj interface{}) *metric.Family {
						return &metric.Family{
							Metrics: []*metric.Metric{
								{
									LabelKeys:   []string{"one", "two", "three"},
									LabelValues: []string{"value-one", "value-two", "value-three"},
									Value:       1,
								},
							},
						}
					},
				},
			},
		},
		{
			name:        "Returns no labels if it's an annotation metric and no allowed labels specified",
			allowLabels: allow.Labels(map[string][]string{}),
			familyGenerators: []FamilyGenerator{
				{
					Name: "node_annotations",
					Help: "some help",
					Type: metric.Gauge,
					GenerateFunc: func(obj interface{}) *metric.Family {
						return &metric.Family{
							Metrics: []*metric.Metric{
								{
									LabelKeys:   []string{"one", "two", "three"},
									LabelValues: []string{"value-one", "value-two", "value-three"},
									Value:       1,
								},
							},
						}
					},
				},
			},
			results: []FamilyGenerator{
				{
					Name: "node_annotations",
					Help: "some help",
					Type: metric.Gauge,
					GenerateFunc: func(obj interface{}) *metric.Family {
						return &metric.Family{
							Metrics: []*metric.Metric{
								{
									Value: 1,
								},
							},
						}
					},
				},
			},
		},
		{
			name:        "Returns no labels if it's an label metric and no allowed labels specified",
			allowLabels: allow.Labels(map[string][]string{}),
			familyGenerators: []FamilyGenerator{
				{
					Name: "node_labels",
					Help: "some help",
					Type: metric.Gauge,
					GenerateFunc: func(obj interface{}) *metric.Family {
						return &metric.Family{
							Metrics: []*metric.Metric{
								{
									LabelKeys:   []string{"one", "two", "three"},
									LabelValues: []string{"value-one", "value-two", "value-three"},
									Value:       1,
								},
							},
						}
					},
				},
			},
			results: []FamilyGenerator{
				{
					Name: "node_labels",
					Help: "some help",
					Type: metric.Gauge,
					GenerateFunc: func(obj interface{}) *metric.Family {
						return &metric.Family{
							Metrics: []*metric.Metric{
								{
									Value: 1,
								},
							},
						}
					},
				},
			},
		},
		{
			name: "Returns allowed labels for metric and label and value pairs are correct",
			allowLabels: allow.Labels(map[string][]string{
				"node_info": {
					"two",
					"one",
				},
			}),
			familyGenerators: []FamilyGenerator{
				{
					Name: "node_info",
					Help: "some help",
					Type: metric.Gauge,
					GenerateFunc: func(obj interface{}) *metric.Family {
						return &metric.Family{
							Metrics: []*metric.Metric{
								{
									LabelKeys:   []string{"one", "two", "three"},
									LabelValues: []string{"value-one", "value-two", "value-three"},
									Value:       1,
								},
							},
						}
					},
				},
			},
			results: []FamilyGenerator{
				{
					Name: "node_info",
					Help: "some help",
					Type: metric.Gauge,
					GenerateFunc: func(obj interface{}) *metric.Family {
						return &metric.Family{
							Metrics: []*metric.Metric{
								{
									LabelKeys:   []string{"two", "one"},
									LabelValues: []string{"value-two", "value-one"},
									Value:       1,
								},
							},
						}
					},
				},
			},
		},
		{
			name: "Returns allowed labels for metric",
			allowLabels: allow.Labels(map[string][]string{
				"node_labels": {
					"one",
					"two",
				},
			}),
			familyGenerators: []FamilyGenerator{
				{
					Name: "node_labels",
					Help: "some help",
					Type: metric.Gauge,
					GenerateFunc: func(obj interface{}) *metric.Family {
						return &metric.Family{
							Metrics: []*metric.Metric{
								{
									LabelKeys:   []string{"one", "two", "three"},
									LabelValues: []string{"value-one", "value-two", "value-three"},
									Value:       1,
								},
							},
						}
					},
				},
			},
			results: []FamilyGenerator{
				{
					Name: "node_labels",
					Help: "some help",
					Type: metric.Gauge,
					GenerateFunc: func(obj interface{}) *metric.Family {
						return &metric.Family{
							Metrics: []*metric.Metric{
								{
									LabelKeys:   []string{"one", "two"},
									LabelValues: []string{"value-one", "value-two"},
									Value:       1,
								},
							},
						}
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := FilterMetricFamiliesLabels(test.allowLabels, test.familyGenerators)
			if len(results) != len(test.results) {
				t.Fatalf("expected %v, got %v", len(test.results), len(results))
			}

			for i := range results {
				result := results[i].GenerateFunc(nil)
				expected := test.results[i].GenerateFunc(nil)
				for _, resultMetric := range result.Metrics {
					for _, expectedMetric := range expected.Metrics {
						assertEqualSlices(t, expectedMetric.LabelKeys, resultMetric.LabelKeys, "keys")
						assertEqualSlices(t, expectedMetric.LabelValues, resultMetric.LabelValues, "values")

						if expectedMetric.Value != resultMetric.Value {
							t.Fatalf("value - expected %v, got %v", expectedMetric.Value, resultMetric.Value)
						}
					}
				}
			}
		})
	}
}

func assertEqualSlices(t *testing.T, expected, actual []string, kind string) {
	sort.Strings(expected)
	sort.Strings(actual)
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("%s - expected %v, got %v", kind, expected, actual)
	}
}
