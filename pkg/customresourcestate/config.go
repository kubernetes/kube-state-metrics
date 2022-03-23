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
)

// GroupVersionKind is the Kubernetes group, version, and kind of a resource.
type GroupVersionKind struct {
	Group   string `yaml:"group" json:"group"`
	Version string `yaml:"version" json:"version"`
	Kind    string `yaml:"kind" json:"kind"`
}

// MetricPer targets a Path that may be a single value, array, or object. Arrays and objects will generate a metric per element.
type MetricPer struct {
	// Path is the path to the value to generate metric(s) for.
	Path []string `yaml:"path" json:"path"`
	// ValueFrom is the path to a numeric field under Path that will be the metric value.
	ValueFrom []string `yaml:"valueFrom" json:"valueFrom"`
	// LabelFromKey adds a label with the given name if Path is an object. The label value will be the object key.
	LabelFromKey string `yaml:"labelFromKey" json:"labelFromKey"`
	// LabelsFromPath adds additional labels where the value of the label is taken from a field under Path.
	LabelsFromPath map[string][]string `yaml:"labelsFromPath" json:"labelsFromPath"`
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
	Each MetricPer `yaml:"each" json:"each"`

	// Labels are added to all metrics. Labels from Each will overwrite these if using the same key.
	Labels `yaml:",inline"` // json will inline because it is already tagged
	// ErrorLogV defines the verbosity threshold for errors logged for this metric. Must be non-zero to override the resource setting.
	ErrorLogV klog.Level `yaml:"errorLogV" json:"errorLogV"`
}

// Resource configures a custom resource for metric generation.
type Resource struct {
	// Namespace is an optional prefix for all metrics. Defaults to "kube" if not set. If set to "_", no namespace will be added.
	// The combination of Namespace and Subsystem will be prefixed to all metrics generated for this resource.
	// e.g., if Namespace is "kube" and Subsystem is "myteam_io_v1_MyResource", all metrics will be prefixed with "kube_myteam_io_v1_MyResource_".
	Namespace string `yaml:"namespace" json:"namespace"`
	// Subsystem defaults to the GroupVersionKind string, with invalid character replaced with _. If set to "_", no subsystem will be added.
	// e.g., if GroupVersionKind is "myteam.io/v1/MyResource", Subsystem will be "myteam_io_v1_MyResource".
	Subsystem string `yaml:"subsystem" json:"subsystem"`

	// GroupVersionKind of the custom resource to be monitored.
	GroupVersionKind GroupVersionKind `yaml:"groupVersionKind" json:"groupVersionKind"`

	// Labels are added to all metrics. If the same key is used in a metric, the value from the metric will overwrite the value here.
	Labels `yaml:",inline"`

	// Metrics are the custom resource fields to be collected.
	Metrics []Generator `yaml:"metrics" json:"metrics"`
	// ErrorLogV defines the verbosity threshold for errors logged for this resource.
	ErrorLogV klog.Level `yaml:"errorLogV" json:"errorLogV"`

	// ResourcePlural sets the plural name of the resource. Defaults to the plural version of the Kind according to flect.Pluralize.
	ResourcePlural string `yaml:"resourcePlural" json:"resourcePlural"`
}

// GetNamespace returns the namespace prefix to use for metrics.
func (r Resource) GetNamespace() string {
	if r.Namespace == "" {
		return "kube"
	}
	if r.Namespace == "_" {
		return ""
	}
	return r.Namespace
}

// GetSubsystem returns the subsystem prefix to use for metrics (will be joined between namespace and the metric name).
func (r Resource) GetSubsystem() string {
	if r.Subsystem == "" {
		return strings.NewReplacer(
			"/", "_",
			".", "_",
		).Replace(fmt.Sprintf("%s_%s_%s", r.GroupVersionKind.Group, r.GroupVersionKind.Version, r.GroupVersionKind.Kind))
	}
	if r.Subsystem == "_" {
		return ""
	}
	return r.Subsystem
}

// GetResourceName returns the lowercase, plural form of the resource Kind. This is ResourcePlural if it is set.
func (r Resource) GetResourceName() string {
	if r.ResourcePlural != "" {
		return r.ResourcePlural
	}
	// kubebuilder default:
	return strings.ToLower(flect.Pluralize(r.GroupVersionKind.Kind))
}

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
