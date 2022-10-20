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
	"fmt"
	"strings"

	"github.com/gobuffalo/flect"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/pkg/customresource"
)

// Metrics is the top level configuration object.
type Metrics struct {
	Spec MetricsSpec `yaml:"spec" json:"spec"`
}

// MetricsSpec is the configuration describing the custom resource state metrics to generate.
type MetricsSpec struct {
	// Resources is the list of custom resources to be monitored. A resource with the same GroupVersionKind may appear
	// multiple times (e.g., to customize the namespace or subsystem,) but will incur additional overhead.
	Resources []Resource `yaml:"resources" json:"resources"`
}

// Resource configures a custom resource for metric generation.
type Resource struct {
	// MetricNamePrefix defines a prefix for all metrics of the resource.
	// If set to "", no prefix will be added.
	// Example: If set to "foo", MetricNamePrefix will be "foo_<metric>".
	MetricNamePrefix *string `yaml:"metricNamePrefix" json:"metricNamePrefix"`

	// GroupVersionKind of the custom resource to be monitored.
	GroupVersionKind GroupVersionKind `yaml:"groupVersionKind" json:"groupVersionKind"`

	// Labels are added to all metrics. If the same key is used in a metric, the value from the metric will overwrite the value here.
	Labels `yaml:",inline" json:",inline"`

	// Metrics are the custom resource fields to be collected.
	Metrics []Generator `yaml:"metrics" json:"metrics"`
	// ErrorLogV defines the verbosity threshold for errors logged for this resource.
	ErrorLogV klog.Level `yaml:"errorLogV" json:"errorLogV"`

	// ResourcePlural sets the plural name of the resource. Defaults to the plural version of the Kind according to flect.Pluralize.
	ResourcePlural string `yaml:"resourcePlural" json:"resourcePlural"`
}

// GetMetricNamePrefix returns the prefix to use for metrics.
func (r Resource) GetMetricNamePrefix() string {
	p := r.MetricNamePrefix
	if p == nil {
		return "kube_crd"
	}
	return *p
}

// GetResourceName returns the lowercase, plural form of the resource Kind. This is ResourcePlural if it is set.
func (r Resource) GetResourceName() string {
	if r.ResourcePlural != "" {
		return r.ResourcePlural
	}
	// kubebuilder default:
	return strings.ToLower(flect.Pluralize(r.GroupVersionKind.Kind))
}

// GroupVersionKind is the Kubernetes group, version, and kind of a resource.
type GroupVersionKind struct {
	Group   string `yaml:"group" json:"group"`
	Version string `yaml:"version" json:"version"`
	Kind    string `yaml:"kind" json:"kind"`
}

// Labels is common configuration of labels to add to metrics.
type Labels struct {
	// CommonLabels are added to all metrics.
	CommonLabels map[string]string `yaml:"commonLabels" json:"commonLabels"`
	// LabelsFromPath adds additional labels where the value is taken from a field in the resource.
	LabelsFromPath map[string][]string `yaml:"labelsFromPath" json:"labelsFromPath"`
}

// Merge combines the labels from two configs, returning a new config. The other Labels will overwrite keys in this Labels.
func (l Labels) Merge(other Labels) Labels {
	common := make(map[string]string)
	paths := make(map[string][]string)

	for k, v := range l.CommonLabels {
		common[k] = v
	}
	for k, v := range l.LabelsFromPath {
		paths[k] = v
	}
	for k, v := range other.CommonLabels {
		common[k] = v
	}
	for k, v := range other.LabelsFromPath {
		paths[k] = v
	}
	return Labels{
		CommonLabels:   common,
		LabelsFromPath: paths,
	}
}

// Generator describes a unique metric name.
type Generator struct {
	// Name of the metric. Subject to prefixing based on the configuration of the Resource.
	Name string `yaml:"name" json:"name"`
	// Help text for the metric.
	Help string `yaml:"help" json:"help"`
	// Each targets a value or values from the resource.
	Each Metric `yaml:"each" json:"each"`

	// Labels are added to all metrics. Labels from Each will overwrite these if using the same key.
	Labels `yaml:",inline" json:",inline"` // json will inline because it is already tagged
	// ErrorLogV defines the verbosity threshold for errors logged for this metric. Must be non-zero to override the resource setting.
	ErrorLogV klog.Level `yaml:"errorLogV" json:"errorLogV"`
}

// Metric defines a metric to expose.
// +union
type Metric struct {
	// Type defines the type of the metric.
	// +unionDiscriminator
	Type MetricType `yaml:"type" json:"type"`

	// Gauge defines a gauge metric.
	// +optional
	Gauge *MetricGauge `yaml:"gauge" json:"gauge"`
	// StateSet defines a state set metric.
	// +optional
	StateSet *MetricStateSet `yaml:"stateSet" json:"stateSet"`
	// Info defines an info metric.
	// +optional
	Info *MetricInfo `yaml:"info" json:"info"`
}

// ConfigDecoder is for use with FromConfig.
type ConfigDecoder interface {
	Decode(v interface{}) (err error)
}

// FromConfig decodes a configuration source into a slice of customresource.RegistryFactory that are ready to use.
func FromConfig(decoder ConfigDecoder) ([]customresource.RegistryFactory, error) {
	var crconfig Metrics
	var factories []customresource.RegistryFactory
	factoriesIndex := map[string]bool{}
	if err := decoder.Decode(&crconfig); err != nil {
		return nil, fmt.Errorf("failed to parse Custom Resource State metrics: %w", err)
	}
	for _, resource := range crconfig.Spec.Resources {
		factory, err := NewCustomResourceMetrics(resource)
		if err != nil {
			return nil, fmt.Errorf("failed to create metrics factory for %s: %w", resource.GroupVersionKind, err)
		}
		if _, ok := factoriesIndex[factory.Name()]; ok {
			return nil, fmt.Errorf("found multiple custom resource configurations for the same resource %s", factory.Name())
		}
		factoriesIndex[factory.Name()] = true
		factories = append(factories, factory)
	}
	return factories, nil
}
