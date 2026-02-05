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

package customresourcestate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

func mustCompileMetric(t *testing.T, m Metric) compiledMetric {
	t.Helper()
	compiled, err := newCompiledMetric(m)
	if err != nil {
		t.Fatalf("compile metric: %v", err)
	}
	return compiled
}

func Test_extractValues(t *testing.T) {
	type tc struct {
		name       string
		each       compiledEach
		wantResult []eachValue
		wantErrors []error
	}

	tests := []tc{
		{
			name: "single",
			each: mustCompileMetric(t,
				Metric{
					Type: metric.Gauge,
					Gauge: &MetricGauge{
						MetricMeta: MetricMeta{
							Path: []string{"spec", "replicas"},
						},
					},
				},
			),
			wantResult: []eachValue{newEachValue(t, 1)},
		},

		{
			name: "obj",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"status", "active"},
					},
					LabelFromKey: "type",
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 1, "type", "type-a"),
				newEachValue(t, 3, "type", "type-b"),
			},
		},

		{
			name: "deep obj",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"status", "sub"},
						LabelsFromPath: map[string][]string{
							"active": {"active"},
						},
					},
					ValueFrom:    ValueFrom{PathValueFrom: []string{"ready"}},
					LabelFromKey: "type",
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 2, "type", "type-a", "active", "1"),
				newEachValue(t, 4, "type", "type-b", "active", "3"),
			},
		},

		{
			name: "path-relative valueFrom value",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"metadata"},
						LabelsFromPath: map[string][]string{
							"name": {"name"},
						},
					},
					ValueFrom: ValueFrom{PathValueFrom: []string{"creationTimestamp"}},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 1.6563744e+09, "name", "foo"),
			},
		},

		{
			name: "non-existent path",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"foo"},
						LabelsFromPath: map[string][]string{
							"name": {"name"},
						},
					},
					ValueFrom: ValueFrom{PathValueFrom: []string{"creationTimestamp"}},
				},
			}),
			wantResult: nil, wantErrors: []error{
				errors.New("[foo]: got nil while resolving path"),
			},
		},

		{
			name: "exist path but valueFrom path is non-existent single",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"spec", "replicas"},
					},
					ValueFrom: ValueFrom{PathValueFrom: []string{"non-existent"}},
				},
			}),
			wantResult: nil, wantErrors: nil,
		},

		{
			name: "exist path but valueFrom path non-existent array",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"status", "condition_values"},
						LabelsFromPath: map[string][]string{
							"name": {"name"},
						},
					},
					ValueFrom: ValueFrom{PathValueFrom: []string{"non-existent"}},
				},
			}),
			wantResult: nil, wantErrors: nil,
		},

		{
			name: "array",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"status", "condition_values"},
						LabelsFromPath: map[string][]string{
							"name": {"name"},
						},
					},
					ValueFrom: ValueFrom{PathValueFrom: []string{"value"}},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 45, "name", "a"),
				newEachValue(t, 66, "name", "b"),
			},
		},

		{
			name: "timestamp",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"metadata", "creationTimestamp"},
					},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 1656374400),
			},
		},

		{
			name: "quantity_milli",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"status", "quantity_milli"},
					},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 0.25),
			},
		},

		{
			name: "quantity_binarySI",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"status", "quantity_binarySI"},
					},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 5.36870912e+09),
			},
		},

		{
			name: "percentage",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"status", "percentage"},
					},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 0.28),
			},
		},

		{
			name: "path-relative valueFrom percentage",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"metadata"},
						LabelsFromPath: map[string][]string{
							"name": {"name"},
						},
					},
					ValueFrom: ValueFrom{PathValueFrom: []string{"percentage"}},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 0.39, "name", "foo"),
			},
		},

		{
			name: "boolean_string",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"spec", "paused"},
					},
					NilIsZero: true,
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 0),
			},
		},

		{
			name: "info",
			each: mustCompileMetric(t, Metric{
				Type: metric.Info,
				Info: &MetricInfo{
					MetricMeta: MetricMeta{
						LabelsFromPath: map[string][]string{
							"version": []string{"spec", "version"},
						},
					},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 1, "version", "v0.0.0"),
			},
		},

		{
			name: "info nil path",
			each: mustCompileMetric(t, Metric{
				Type: metric.Info,
				Info: &MetricInfo{
					MetricMeta: MetricMeta{
						Path: []string{"does", "not", "exist"},
					},
				},
			}),
			wantResult: nil,
		},

		{
			name: "info label from key",
			each: mustCompileMetric(t, Metric{
				Type: metric.Info,
				Info: &MetricInfo{
					MetricMeta: MetricMeta{
						Path: []string{"status", "active"},
					},
					LabelFromKey: "type",
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 1, "type", "type-a"),
				newEachValue(t, 1, "type", "type-b"),
			},
		},

		{
			name: "info label from path",
			each: mustCompileMetric(t, Metric{
				Type: metric.Info,
				Info: &MetricInfo{
					MetricMeta: MetricMeta{
						Path: []string{"status", "sub"},
						LabelsFromPath: map[string][]string{
							"active": {"active"},
						},
					},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 1, "active", "1"),
				newEachValue(t, 1, "active", "3"),
			},
		},

		{
			name: "stateset",
			each: mustCompileMetric(t, Metric{
				Type: metric.StateSet,
				StateSet: &MetricStateSet{
					MetricMeta: MetricMeta{
						Path: []string{"status", "phase"},
					},
					LabelName: "phase",
					List:      []string{"foo", "bar"},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 0, "phase", "bar"),
				newEachValue(t, 1, "phase", "foo"),
			},
		},

		{
			name: "status_conditions",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"status", "conditions", "[type=Ready]", "status"},
					},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 1),
			},
		},

		{
			name: "status_conditions_all",
			each: mustCompileMetric(t, Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"status", "conditions"},
						LabelsFromPath: map[string][]string{
							"type": {"type"},
						},
					},
					ValueFrom: ValueFrom{PathValueFrom: []string{"status"}},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 0, "type", "Provisioned"),
				newEachValue(t, 1, "type", "Ready"),
			},
		},

		{
			name: "= expression matching",
			each: mustCompileMetric(t, Metric{
				Type: metric.Info,
				Info: &MetricInfo{
					MetricMeta: MetricMeta{
						LabelsFromPath: map[string][]string{
							"bar": {"metadata", "annotations", "bar=baz"},
						},
					},
				},
			}),
			wantResult: []eachValue{
				newEachValue(t, 1, "bar", "baz"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotErrors := scrapeValuesFor(tt.each, cr)
			assert.Equal(t, tt.wantResult, gotResult)
			assert.Equal(t, tt.wantErrors, gotErrors)
		})
	}
}
