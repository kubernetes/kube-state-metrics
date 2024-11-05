/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descResourceQuotaAnnotationsName     = "kube_resourcequota_annotations"
	descResourceQuotaAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descResourceQuotaLabelsName          = "kube_resourcequota_labels"
	descResourceQuotaLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descResourceQuotaLabelsDefaultLabels = []string{"namespace", "resourcequota"}
)

func resourceQuotaMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_resourcequota_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapResourceQuotaFunc(func(r *v1.ResourceQuota) *metric.Family {
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
			"kube_resourcequota",
			"Information about resource quota.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapResourceQuotaFunc(func(r *v1.ResourceQuota) *metric.Family {
				ms := []*metric.Metric{}

				for res, qty := range r.Status.Hard {
					ms = append(ms, &metric.Metric{
						LabelValues: []string{string(res), "hard"},
						Value:       convertValueToFloat64(&qty),
					})
				}
				for res, qty := range r.Status.Used {
					ms = append(ms, &metric.Metric{
						LabelValues: []string{string(res), "used"},
						Value:       convertValueToFloat64(&qty),
					})
				}

				for _, m := range ms {
					m.LabelKeys = []string{"resource", "type"}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descResourceQuotaAnnotationsName,
			descResourceQuotaAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceQuotaFunc(func(d *v1.ResourceQuota) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", d.Annotations, allowAnnotationsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   annotationKeys,
							LabelValues: annotationValues,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descResourceQuotaLabelsName,
			descResourceQuotaLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapResourceQuotaFunc(func(d *v1.ResourceQuota) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", d.Labels, allowLabelsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					},
				}
			}),
		),
	}
}

func wrapResourceQuotaFunc(f func(*v1.ResourceQuota) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		resourceQuota := obj.(*v1.ResourceQuota)

		metricFamily := f(resourceQuota)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descResourceQuotaLabelsDefaultLabels, []string{resourceQuota.Namespace, resourceQuota.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createResourceQuotaListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().ResourceQuotas(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().ResourceQuotas(ns).Watch(context.TODO(), opts)
		},
	}
}
