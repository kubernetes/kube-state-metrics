/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

// ValueType specifies how to parse the value from the resource field.
type ValueType string

const (
	// ValueTypeDefault uses automatic type detection (current behavior).
	// This is the default when ValueType is not specified.
	ValueTypeDefault ValueType = ""
	// ValueTypeDuration parses duration strings (e.g., "1h", "30m", "1h30m45s") using time.ParseDuration.
	// The parsed duration is converted to seconds as a float64 for Prometheus compatibility.
	ValueTypeDuration ValueType = "duration"
	// ValueTypeQuantity parses Kubernetes resource quantities (e.g., "250m" for millicores, "1Gi" for memory).
	ValueTypeQuantity ValueType = "quantity"
)

// MetricMeta are variables which may used for any metric type.
type MetricMeta struct {
	// LabelsFromPath adds additional labels where the value of the label is taken from a field under Path.
	LabelsFromPath map[string][]string `yaml:"labelsFromPath" json:"labelsFromPath"`
	// Path is the path to to generate metric(s) for.
	Path []string `yaml:"path" json:"path"`
}

// MetricGauge targets a Path that may be a single value, array, or object. Arrays and objects will generate a metric per element.
// Ref: https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md#gauge
type MetricGauge struct {
	// LabelFromKey adds a label with the given name if Path is an object. The label value will be the object key.
	LabelFromKey string `yaml:"labelFromKey" json:"labelFromKey"`
	MetricMeta   `yaml:",inline" json:",inline"`

	// ValueFrom is the path to a numeric field under Path that will be the metric value.
	ValueFrom []string  `yaml:"valueFrom" json:"valueFrom"`
	ValueType ValueType `yaml:"valueType,omitempty" json:"valueType,omitempty"`

	// NilIsZero indicates that if a value is nil it will be treated as zero value.
	NilIsZero bool `yaml:"nilIsZero" json:"nilIsZero"`
}

// MetricInfo is a metric which is used to expose textual information.
// Ref: https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md#info
type MetricInfo struct {
	// LabelFromKey adds a label with the given name if Path is an object. The label value will be the object key.
	LabelFromKey string `yaml:"labelFromKey" json:"labelFromKey"`
	MetricMeta   `yaml:",inline" json:",inline"`
}

// MetricStateSet is a metric which represent a series of related boolean values, also called a bitset.
// Ref: https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md#stateset
type MetricStateSet struct {
	MetricMeta `yaml:",inline" json:",inline"`

	// List is the list of values to expose a value for.
	List []string `yaml:"list" json:"list"`
	// LabelName is the key of the label which is used for each entry in List to expose the value.
	LabelName string `yaml:"labelName" json:"labelName"`
	// ValueFrom is the subpath to compare the list to.
	ValueFrom []string `yaml:"valueFrom" json:"valueFrom"`
}
