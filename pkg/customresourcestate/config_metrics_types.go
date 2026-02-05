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

import (
	"encoding/json"
	"fmt"
)

// MetricMeta are variables which may used for any metric type.
type MetricMeta struct {
	// LabelsFromPath adds additional labels where the value of the label is taken from a field under Path.
	LabelsFromPath map[string][]string `yaml:"labelsFromPath" json:"labelsFromPath"`
	// Path is the path to to generate metric(s) for.
	Path []string `yaml:"path" json:"path"`
}

// ValueFrom defines how to derive a value from a path.
// Either PathValueFrom or CelExpr can be set, but not both.
type ValueFrom struct {
	// PathValueFrom specifies the path-based value extraction.
	PathValueFrom []string `yaml:"pathValueFrom,omitempty" json:"pathValueFrom,omitempty"`
	// CelExpr is a CEL expression to extract/compute the value.
	CelExpr string `yaml:"celExpr,omitempty" json:"celExpr,omitempty"`
}

// unmarshallValueFrom unmarshalls ValueFrom from either a string slice or struct.
func (vf *ValueFrom) unmarshallValueFrom(unmarshal func(interface{}) error) error {
	var stringSlice []string
	if err := unmarshal(&stringSlice); err == nil {
		vf.PathValueFrom = stringSlice
		vf.CelExpr = ""
		return nil
	}

	type valueFromAlias ValueFrom
	var valueFromStruct valueFromAlias
	if err := unmarshal(&valueFromStruct); err != nil {
		return err
	}

	// Ensure fields are mutually exclusive
	if len(valueFromStruct.PathValueFrom) > 0 && valueFromStruct.CelExpr != "" {
		return fmt.Errorf("cannot specify both pathValueFrom and celExpr")
	}

	*vf = ValueFrom(valueFromStruct)
	return nil
}

// UnmarshalJSON unmarshalls ValueFrom either from a string slice or from a full struct.
func (vf *ValueFrom) UnmarshalJSON(data []byte) error {
	return vf.unmarshallValueFrom(func(v interface{}) error {
		return json.Unmarshal(data, v)
	})
}

// UnmarshalYAML unmarshalls ValueFrom either from a string slice or from a full struct.
func (vf *ValueFrom) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return vf.unmarshallValueFrom(unmarshal)
}

// MetricGauge targets a Path that may be a single value, array, or object. Arrays and objects will generate a metric per element.
// Ref: https://github.com/prometheus/OpenMetrics/blob/v1.0.0/specification/OpenMetrics.md#gauge
type MetricGauge struct {
	// LabelFromKey adds a label with the given name if Path is an object. The label value will be the object key.
	LabelFromKey string `yaml:"labelFromKey" json:"labelFromKey"`
	MetricMeta   `yaml:",inline" json:",inline"`

	// ValueFrom is the subpath or function to derive the value from.
	ValueFrom ValueFrom `yaml:"valueFrom" json:"valueFrom"`
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
	ValueFrom ValueFrom `yaml:"valueFrom" json:"valueFrom"`
}
