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
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

// Test_CEL_Custom_CELResult_Type tests the custom CELResult type that allows
// CEL expressions to return both a value and additional labels.
func Test_CEL_Custom_CELResult_Type(t *testing.T) {
	tests := []struct {
		name       string
		expr       string
		value      interface{}
		wantValue  float64
		wantLabels map[string]string
	}{
		{
			name:       "CELResult with value only",
			expr:       "CELResult(100.0, {})",
			value:      nil,
			wantValue:  100.0,
			wantLabels: map[string]string{},
		},
		{
			name:       "CELResult with int value is converted to float",
			expr:       "CELResult(42, {})",
			value:      nil,
			wantValue:  42.0,
			wantLabels: map[string]string{},
		},
		{
			name:       "CELResult with additional labels",
			expr:       "CELResult(42.0, {'severity': 'high', 'component': 'api'})",
			value:      nil,
			wantValue:  42.0,
			wantLabels: map[string]string{"severity": "high", "component": "api"},
		},
		{
			name:       "CELResult with computed value and labels",
			expr:       "CELResult(double(value) * 10.0, {'multiplied': 'true'})",
			value:      5.0,
			wantValue:  50.0,
			wantLabels: map[string]string{"multiplied": "true"},
		},
		{
			name:       "CELResult with conditional logic",
			expr:       "value > 10 ? CELResult(1.0, {'status': 'high'}) : CELResult(0.0, {'status': 'low'})",
			value:      15.0,
			wantValue:  1.0,
			wantLabels: map[string]string{"status": "high"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := newCELValueExtractor(tt.expr, nil, nil, false)
			assert.NoError(t, err)

			results, errs := extractor.extractValues(tt.value)
			assert.Empty(t, errs)
			assert.Len(t, results, 1)
			assert.Equal(t, tt.wantValue, results[0].Value)
			assert.Equal(t, tt.wantLabels, results[0].Labels)
		})
	}
}

// Test_CEL_Value_Type_Conversions tests that CEL expressions can return
// numeric values directly (without CELResult wrapper) and they're properly converted to float64.
func Test_CEL_Value_Type_Conversions(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		value     interface{}
		wantValue float64
	}{
		{
			name:      "direct int value",
			expr:      "42",
			value:     nil,
			wantValue: 42.0,
		},
		{
			name:      "direct double value",
			expr:      "3.14",
			value:     nil,
			wantValue: 3.14,
		},
		{
			name:      "value variable passthrough",
			expr:      "value",
			value:     99.5,
			wantValue: 99.5,
		},
		{
			name:      "arithmetic on value",
			expr:      "double(value) * 2.0",
			value:     21.5,
			wantValue: 43.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor, err := newCELValueExtractor(tt.expr, nil, nil, false)
			assert.NoError(t, err)

			results, errs := extractor.extractValues(tt.value)
			assert.Empty(t, errs)
			assert.Len(t, results, 1)
			assert.Equal(t, tt.wantValue, results[0].Value)
			assert.Empty(t, results[0].Labels) // Direct values have no labels
		})
	}
}

// Test_CEL_With_Real_CR_Data tests CEL extractor against cr imitation data
func Test_CEL_With_Real_CR_Data(t *testing.T) {
	tests := []struct {
		name    string
		metric  Metric
		want    []eachValue
		wantErr bool
	}{
		{
			name: "transform numeric value",
			metric: Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"status", "uptime"},
					},
					ValueFrom: ValueFrom{CelExpr: "double(value) * 2.0"},
				},
			},
			want: []eachValue{{Value: 86.42, Labels: map[string]string{}}},
		},
		{
			name: "conditional expression",
			metric: Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"spec", "replicas"},
					},
					ValueFrom: ValueFrom{CelExpr: "value > 0 ? 1.0 : 0.0"},
				},
			},
			want: []eachValue{{Value: 1.0, Labels: map[string]string{}}},
		},
		{
			name: "CELResult with labels from expression",
			metric: Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"spec", "replicas"},
					},
					ValueFrom: ValueFrom{CelExpr: "CELResult(double(value), {'scaled': value > 1 ? 'yes' : 'no'})"},
				},
			},
			want: []eachValue{{Value: 1.0, Labels: map[string]string{"scaled": "no"}}},
		},
		{
			name: "CELResult combined with LabelsFromPath",
			metric: Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"metadata"},
						LabelsFromPath: map[string][]string{
							"name": {"name"},
						},
					},
					ValueFrom: ValueFrom{CelExpr: "CELResult(1.0, {'source': 'cel'})"},
				},
			},
			want: []eachValue{{Value: 1.0, Labels: map[string]string{"name": "foo", "source": "cel"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled := mustCompileMetric(t, tt.metric)
			results, errs := scrapeValuesFor(compiled, cr)

			if tt.wantErr {
				assert.NotEmpty(t, errs)
			} else {
				assert.Empty(t, errs)
				assert.Equal(t, tt.want, results)
			}
		})
	}
}

// Test_CEL_Label_Precedence tests that CELResult's AdditionalLabels
// take precedence over labelsFromPath when there are conflicts.
func Test_CEL_Label_Precedence(t *testing.T) {
	tests := []struct {
		name   string
		metric Metric
		want   []eachValue
	}{
		{
			name: "CELResult labels override labelsFromPath",
			metric: Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"metadata"},
						LabelsFromPath: map[string][]string{
							"name":   {"name"},
							"status": {"labels", "status"}, // This would be "bar" from CR
						},
					},
					ValueFrom: ValueFrom{CelExpr: "CELResult(1.0, {'status': 'overridden', 'extra': 'label'})"},
				},
			},
			want: []eachValue{{
				Value: 1.0,
				Labels: map[string]string{
					"name":   "foo",        // From labelsFromPath
					"status": "overridden", // From CELResult (overrides labelsFromPath)
					"extra":  "label",      // From CELResult
				},
			}},
		},
		{
			name: "CELResult labels merge with labelsFromPath when no conflicts",
			metric: Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"spec"},
						LabelsFromPath: map[string][]string{
							"version": {"version"},
						},
					},
					ValueFrom: ValueFrom{CelExpr: "CELResult(double(value.replicas), {'dynamic': 'value'})"},
				},
			},
			want: []eachValue{{
				Value: 1.0,
				Labels: map[string]string{
					"version": "v0.0.0", // From labelsFromPath
					"dynamic": "value",  // From CELResult
				},
			}},
		},
		{
			name: "CELResult with empty labels doesn't affect labelsFromPath",
			metric: Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"metadata"},
						LabelsFromPath: map[string][]string{
							"name": {"name"},
						},
					},
					ValueFrom: ValueFrom{CelExpr: "CELResult(1.0, {})"},
				},
			},
			want: []eachValue{{
				Value: 1.0,
				Labels: map[string]string{
					"name": "foo", // From labelsFromPath
				},
			}},
		},
		{
			name: "direct value (no CELResult) uses only labelsFromPath",
			metric: Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"metadata"},
						LabelsFromPath: map[string][]string{
							"name": {"name"},
						},
					},
					ValueFrom: ValueFrom{CelExpr: "1.0"},
				},
			},
			want: []eachValue{{
				Value: 1.0,
				Labels: map[string]string{
					"name": "foo", // From labelsFromPath
				},
			}},
		},
		{
			name: "CELResult can override multiple labelsFromPath labels",
			metric: Metric{
				Type: metric.Gauge,
				Gauge: &MetricGauge{
					MetricMeta: MetricMeta{
						Path: []string{"spec"},
						LabelsFromPath: map[string][]string{
							"version":  {"version"},
							"replicas": {"replicas"},
						},
					},
					ValueFrom: ValueFrom{CelExpr: "CELResult(100.0, {'version': 'cel-override', 'replicas': 'cel-override'})"},
				},
			},
			want: []eachValue{{
				Value: 100.0,
				Labels: map[string]string{
					"version":  "cel-override", // Overridden by CELResult
					"replicas": "cel-override", // Overridden by CELResult
				},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled := mustCompileMetric(t, tt.metric)
			results, errs := scrapeValuesFor(compiled, cr)
			assert.Empty(t, errs)
			assert.Equal(t, tt.want, results)
		})
	}
}

// Test_CEL_Compilation_Errors tests that invalid CEL expressions produce appropriate errors.
func Test_CEL_Compilation_Errors(t *testing.T) {
	tests := []struct {
		name         string
		celExpr      string
		errSubstring string
	}{
		{
			name:         "empty expression",
			celExpr:      "",
			errSubstring: "cannot be empty",
		},
		{
			name:         "syntax error",
			celExpr:      "value +",
			errSubstring: "failed to compile",
		},
		{
			name:         "undefined variable",
			celExpr:      "unknownVar * 2",
			errSubstring: "undeclared reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newCELValueExtractor(tt.celExpr, nil, nil, false)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errSubstring)
		})
	}
}
