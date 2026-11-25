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

package store

import (
	"context"

	v1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	basemetrics "k8s.io/component-base/metrics"
	"k8s.io/utils/ptr"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descDeviceTaintRuleLabelsDefaultLabels = []string{"devicetaintrule"}
	descDeviceTaintRuleAnnotationsName     = "kube_devicetaintrule_annotations"
	descDeviceTaintRuleAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descDeviceTaintRuleLabelsName          = "kube_devicetaintrule_labels" //nolint:gosec
	descDeviceTaintRuleLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
)

func deviceTaintRuleMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_devicetaintrule_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapDeviceTaintRuleFunc(func(r *resourcev1.DeviceTaintRule) *metric.Family {
				ms := []*metric.Metric{}

				if !r.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(r.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_devicetaintrule_info",
			"Information about device taint rule.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapDeviceTaintRuleFunc(func(r *resourcev1.DeviceTaintRule) *metric.Family {
				// A nil selector matches no devices, an empty one matches all of them.
				selector := "none"
				var driver, pool, device string
				if s := r.Spec.DeviceSelector; s != nil {
					driver = ptr.Deref(s.Driver, "")
					pool = ptr.Deref(s.Pool, "")
					device = ptr.Deref(s.Device, "")
					selector = "all"
					if s.Driver != nil || s.Pool != nil || s.Device != nil {
						selector = "filtered"
					}
				}

				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"taint_key", "taint_value", "taint_effect", "device_selector", "driver", "pool", "device"},
							LabelValues: []string{r.Spec.Taint.Key, r.Spec.Taint.Value, string(r.Spec.Taint.Effect), selector, driver, pool, device},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_devicetaintrule_taint_time_added",
			"Unix timestamp at which the taint was added or its effect was last modified.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapDeviceTaintRuleFunc(func(r *resourcev1.DeviceTaintRule) *metric.Family {
				ms := []*metric.Metric{}

				if t := r.Spec.Taint.TimeAdded; t != nil && !t.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(t.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_devicetaintrule_metadata_generation",
			"Sequence number representing a specific generation of the desired state.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapDeviceTaintRuleFunc(func(r *resourcev1.DeviceTaintRule) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(r.Generation),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_devicetaintrule_status_condition",
			"The condition of a device taint rule.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapDeviceTaintRuleFunc(func(r *resourcev1.DeviceTaintRule) *metric.Family {
				ms := make([]*metric.Metric, 0, len(r.Status.Conditions)*len(conditionStatuses))

				for _, c := range r.Status.Conditions {
					for _, m := range addConditionMetrics(v1.ConditionStatus(c.Status)) {
						m.LabelKeys = []string{"condition", "status"}
						m.LabelValues = append([]string{c.Type}, m.LabelValues...)
						ms = append(ms, m)
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descDeviceTaintRuleAnnotationsName,
			descDeviceTaintRuleAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapDeviceTaintRuleFunc(func(r *resourcev1.DeviceTaintRule) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				keys, values := createPrometheusLabelKeysValues("annotation", r.Annotations, allowAnnotationsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   keys,
							LabelValues: values,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descDeviceTaintRuleLabelsName,
			descDeviceTaintRuleLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapDeviceTaintRuleFunc(func(r *resourcev1.DeviceTaintRule) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				keys, values := createPrometheusLabelKeysValues("label", r.Labels, allowLabelsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   keys,
							LabelValues: values,
							Value:       1,
						},
					},
				}
			}),
		),
	}
}

func wrapDeviceTaintRuleFunc(f func(*resourcev1.DeviceTaintRule) *metric.Family) func(any) *metric.Family {
	return func(obj any) *metric.Family {
		r := obj.(*resourcev1.DeviceTaintRule)

		metricFamily := f(r)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descDeviceTaintRuleLabelsDefaultLabels, []string{r.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createDeviceTaintRuleListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.ResourceV1().DeviceTaintRules().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.ResourceV1().DeviceTaintRules().Watch(context.TODO(), opts)
		},
	}
}
