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

	"k8s.io/kube-state-metrics/v2/pkg/metric"

	"github.com/gobuffalo/flect"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/internal/discovery"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/util"
)

// customResourceState is used to prefix the auto-generated GVK labels as well as an appendix for the metric itself
// if no custom metric name is defined
const customResourceState string = "customresource"

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

	// Labels are added to all metrics. If the same key is used in a metric, the value from the metric will overwrite the value here.
	Labels `yaml:",inline" json:",inline"`

	// MetricNamePrefix defines a prefix for all metrics of the resource.
	// If set to "", no prefix will be added.
	// Example: If set to "foo", MetricNamePrefix will be "foo_<metric>".
	MetricNamePrefix *string `yaml:"metricNamePrefix" json:"metricNamePrefix"`

	// GroupVersionKind of the custom resource to be monitored.
	GroupVersionKind GroupVersionKind `yaml:"groupVersionKind" json:"groupVersionKind"`

	// ResourcePlural sets the plural name of the resource. Defaults to the plural version of the Kind according to flect.Pluralize.
	ResourcePlural string `yaml:"resourcePlural" json:"resourcePlural"`

	// Metrics are the custom resource fields to be collected.
	Metrics []Generator `yaml:"metrics" json:"metrics"`
	// ErrorLogV defines the verbosity threshold for errors logged for this resource.
	ErrorLogV klog.Level `yaml:"errorLogV" json:"errorLogV"`
}

// GetMetricNamePrefix returns the prefix to use for metrics.
func (r Resource) GetMetricNamePrefix() string {
	p := r.MetricNamePrefix
	if p == nil {
		return "kube_" + customResourceState
	}
	return *p
}

// GetResourceName returns the lowercase, plural form of the resource Kind. This is ResourcePlural if it is set.
func (r Resource) GetResourceName() string {
	if r.ResourcePlural != "" {
		klog.InfoS("Using custom resource plural", "resource", r.GroupVersionKind.String(), "plural", r.ResourcePlural)
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

func (gvk GroupVersionKind) String() string {
	return fmt.Sprintf("%s_%s_%s", gvk.Group, gvk.Version, gvk.Kind)
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
	// Each targets a value or values from the resource.
	Each Metric `yaml:"each" json:"each"`

	// Labels are added to all metrics. Labels from Each will overwrite these if using the same key.
	Labels `yaml:",inline" json:",inline"` // json will inline because it is already tagged
	// Name of the metric. Subject to prefixing based on the configuration of the Resource.
	Name string `yaml:"name" json:"name"`
	// Help text for the metric.
	Help string `yaml:"help" json:"help"`
	// ErrorLogV defines the verbosity threshold for errors logged for this metric. Must be non-zero to override the resource setting.
	ErrorLogV klog.Level `yaml:"errorLogV" json:"errorLogV"`
}

// Metric defines a metric to expose.
// +union
type Metric struct {

	// Gauge defines a gauge metric.
	// +optional
	Gauge *MetricGauge `yaml:"gauge" json:"gauge"`
	// StateSet defines a state set metric.
	// +optional
	StateSet *MetricStateSet `yaml:"stateSet" json:"stateSet"`
	// Info defines an info metric.
	// +optional
	Info *MetricInfo `yaml:"info" json:"info"`
	// Type defines the type of the metric.
	// +unionDiscriminator
	Type metric.Type `yaml:"type" json:"type"`
}

// ConfigDecoder is for use with FromConfig.
type ConfigDecoder interface {
	Decode(v interface{}) (err error)
}

// FromConfig decodes a configuration source into a slice of `customresource.RegistryFactory` that are ready to use.
func FromConfig(decoder ConfigDecoder, discovererInstance *discovery.CRDiscoverer) (func() ([]customresource.RegistryFactory, error), error) {
	var customResourceConfig Metrics
	factoriesIndex := map[string]bool{}

	// Decode the configuration.
	if err := decoder.Decode(&customResourceConfig); err != nil {
		return nil, fmt.Errorf("failed to parse Custom Resource State metrics: %w", err)
	}

	// Override the configuration with any custom overrides.
	configOverrides(&customResourceConfig)

	// Create a factory for each resource.
	fn := func() (factories []customresource.RegistryFactory, err error) {
		resources := customResourceConfig.Spec.Resources
		// resolvedGVKPs will have the final list of GVKs, in addition to the resolved G** resources.
		var resolvedGVKPs []Resource
		for _, resource := range resources /* G** */ {
			resolvedSet /* GVKPs */, err := discovererInstance.ResolveGVKToGVKPs(schema.GroupVersionKind(resource.GroupVersionKind))
			if err != nil {
				klog.ErrorS(err, "failed to resolve GVK", "gvk", resource.GroupVersionKind)
			}
			for _, resolved /* GVKP */ := range resolvedSet {
				// Set their G** attributes to various resolutions of the GVK.
				resource.GroupVersionKind = GroupVersionKind(resolved.GroupVersionKind)
				// Set the plural name of the resource based on the extracted value from the same field in the CRD schema.
				resource.ResourcePlural = resolved.Plural
				resolvedGVKPs = append(resolvedGVKPs, resource)
			}
		}
		for _, resource := range resolvedGVKPs {
			factory, err := NewCustomResourceMetrics(resource)
			if err != nil {
				return nil, fmt.Errorf("failed to create metrics factory for %s: %w", resource.GroupVersionKind, err)
			}
			gvr, err := util.GVRFromType(factory.Name(), factory.ExpectedType())
			if err != nil {
				return nil, fmt.Errorf("failed to create GVR for %s: %w", resource.GroupVersionKind, err)
			}
			var gvrString string
			if gvr != nil {
				gvrString = gvr.String()
			} else {
				gvrString = factory.Name()
			}
			if _, ok := factoriesIndex[gvrString]; ok {
				klog.InfoS("reloaded factory", "GVR", gvrString)
			}
			factoriesIndex[gvrString] = true
			factories = append(factories, factory)
		}
		return factories, nil
	}
	return fn, nil
}

// configOverrides applies overrides to the configuration.
func configOverrides(config *Metrics) {
	for i := range config.Spec.Resources {
		for j := range config.Spec.Resources[i].Metrics {

			// Override the metric type to lowercase, so the internals have a single source of truth for metric type definitions.
			// This is done as a convenience measure for users, so they don't have to remember the exact casing.
			config.Spec.Resources[i].Metrics[j].Each.Type = metric.Type(strings.ToLower(string(config.Spec.Resources[i].Metrics[j].Each.Type)))
		}
	}
}
