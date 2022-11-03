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

// MetricType is the type of a metric.
type MetricType string

// Supported metric types.
const (
	MetricTypeGauge    MetricType = "Gauge"
	MetricTypeStateSet MetricType = "StateSet"
	MetricTypeInfo     MetricType = "Info"
)

// MetricMeta are variables which may used for any metric type.
type MetricMeta struct {
	// LabelsFromPath adds additional labels where the value of the label is taken from a field under Path.
	LabelsFromPath map[string][]string `yaml:"labelsFromPath" json:"labelsFromPath"`
	// Path is the path to to generate metric(s) for.
	Path []string `yaml:"path" json:"path"`
}

// MetricGauge targets a Path that may be a single value, array, or object. Arrays and objects will generate a metric per element.
// Ref: https://github.com/OpenObservability/OpenMetrics/blob/main/specification/OpenMetrics.md#gauge
type MetricGauge struct {
	MetricMeta `yaml:",inline" json:",inline"`

	// ValueFrom is the path to a numeric field under Path that will be the metric value.
	ValueFrom []string `yaml:"valueFrom" json:"valueFrom"`
	// LabelFromKey adds a label with the given name if Path is an object. The label value will be the object key.
	LabelFromKey string `yaml:"labelFromKey" json:"labelFromKey"`
	// NilIsZero indicates that if a value is nil it will be treated as zero value.
	NilIsZero bool `yaml:"nilIsZero" json:"nilIsZero"`
}

// MetricInfo is a metric which is used to expose textual information.
// Ref: https://github.com/OpenObservability/OpenMetrics/blob/main/specification/OpenMetrics.md#info
type MetricInfo struct {
	MetricMeta `yaml:",inline" json:",inline"`
	// LabelFromKey adds a label with the given name if Path is an object. The label value will be the object key.
	LabelFromKey string `yaml:"labelFromKey" json:"labelFromKey"`
}

// MetricStateSet is a metric which represent a series of related boolean values, also called a bitset.
// Ref: https://github.com/OpenObservability/OpenMetrics/blob/main/specification/OpenMetrics.md#stateset
type MetricStateSet struct {
	MetricMeta `yaml:",inline" json:",inline"`

	// List is the list of values to expose a value for.
	List []string `yaml:"list" json:"list"`
	// LabelName is the key of the label which is used for each entry in List to expose the value.
	LabelName string `yaml:"labelName" json:"labelName"`
	// ValueFrom is the subpath to compare the list to.
	ValueFrom []string `yaml:"valueFrom" json:"valueFrom"`
}
