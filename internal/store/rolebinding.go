/*
Copyright 2022 The Kubernetes Authors All rights reserved.

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

	rbacv1 "k8s.io/api/rbac/v1"
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
	descRoleBindingAnnotationsName     = "kube_rolebinding_annotations"
	descRoleBindingAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descRoleBindingLabelsName          = "kube_rolebinding_labels"
	descRoleBindingLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descRoleBindingLabelsDefaultLabels = []string{"namespace", "rolebinding"}
)

func roleBindingMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			descRoleBindingAnnotationsName,
			descRoleBindingAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapRoleBindingFunc(func(r *rbacv1.RoleBinding) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", r.Annotations, allowAnnotationsList)
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
			descRoleBindingLabelsName,
			descRoleBindingLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapRoleBindingFunc(func(r *rbacv1.RoleBinding) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", r.Labels, allowLabelsList)
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
		*generator.NewFamilyGeneratorWithStability(
			"kube_rolebinding_info",
			"Information about rolebinding.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapRoleBindingFunc(func(r *rbacv1.RoleBinding) *metric.Family {
				labelKeys := []string{"roleref_kind", "roleref_name"}
				labelValues := []string{r.RoleRef.Kind, r.RoleRef.Name}
				return &metric.Family{
					Metrics: []*metric.Metric{{
						LabelKeys:   labelKeys,
						LabelValues: labelValues,
						Value:       1,
					}},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_rolebinding_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapRoleBindingFunc(func(r *rbacv1.RoleBinding) *metric.Family {
				ms := []*metric.Metric{}

				if !r.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(r.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_rolebinding_metadata_resource_version",
			"Resource version representing a specific version of the rolebinding.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapRoleBindingFunc(func(r *rbacv1.RoleBinding) *metric.Family {
				return &metric.Family{
					Metrics: resourceVersionMetric(r.ResourceVersion),
				}
			}),
		),
	}
}

func createRoleBindingListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.RbacV1().RoleBindings(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.RbacV1().RoleBindings(ns).Watch(context.TODO(), opts)
		},
	}
}

func wrapRoleBindingFunc(f func(*rbacv1.RoleBinding) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		rolebinding := obj.(*rbacv1.RoleBinding)

		metricFamily := f(rolebinding)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descRoleBindingLabelsDefaultLabels, []string{rolebinding.Namespace, rolebinding.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}
