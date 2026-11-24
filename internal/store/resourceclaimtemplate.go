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

	resourcev1 "k8s.io/api/resource/v1"
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
	descResourceClaimTemplateLabelsDefaultLabels = []string{"resourceclaimtemplate", "namespace"}
	descResourceClaimTemplateAnnotationsName     = "kube_resourceclaimtemplate_annotations"
	descResourceClaimTemplateAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descResourceClaimTemplateLabelsName          = "kube_resourceclaimtemplate_labels" //nolint:gosec
	descResourceClaimTemplateLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
)

func resourceClaimTemplateMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_resourceclaimtemplate_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceClaimTemplateFunc(func(rct *resourcev1.ResourceClaimTemplate) *metric.Family {
				ms := []*metric.Metric{}

				if !rct.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(rct.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descResourceClaimTemplateAnnotationsName,
			descResourceClaimTemplateAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceClaimTemplateFunc(func(rct *resourcev1.ResourceClaimTemplate) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				keys, values := createPrometheusLabelKeysValues("annotation", rct.Annotations, allowAnnotationsList)
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
			descResourceClaimTemplateLabelsName,
			descResourceClaimTemplateLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapResourceClaimTemplateFunc(func(rct *resourcev1.ResourceClaimTemplate) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				keys, values := createPrometheusLabelKeysValues("label", rct.Labels, allowLabelsList)
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

func wrapResourceClaimTemplateFunc(f func(*resourcev1.ResourceClaimTemplate) *metric.Family) func(any) *metric.Family {
	return func(obj any) *metric.Family {
		rct := obj.(*resourcev1.ResourceClaimTemplate)

		metricFamily := f(rct)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descResourceClaimTemplateLabelsDefaultLabels, []string{rct.Name, rct.Namespace}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createResourceClaimTemplateListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ResourceV1().ResourceClaimTemplates(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.ResourceV1().ResourceClaimTemplates(ns).Watch(context.TODO(), opts)
		},
	}
}
