/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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
	descRoleAnnotationsName     = "kube_role_annotations"
	descRoleAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descRoleLabelsName          = "kube_role_labels"
	descRoleLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descRoleLabelsDefaultLabels = []string{"namespace", "role"}
)

func roleMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			descRoleAnnotationsName,
			descRoleAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapRoleFunc(func(r *rbacv1.Role) *metric.Family {
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
			descRoleLabelsName,
			descRoleLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapRoleFunc(func(r *rbacv1.Role) *metric.Family {
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
			"kube_role_info",
			"Information about role.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapRoleFunc(func(_ *rbacv1.Role) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       1,
					}},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_role_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapRoleFunc(func(r *rbacv1.Role) *metric.Family {
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
			"kube_role_metadata_resource_version",
			"Resource version representing a specific version of the role.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapRoleFunc(func(r *rbacv1.Role) *metric.Family {
				return &metric.Family{
					Metrics: resourceVersionMetric(r.ResourceVersion),
				}
			}),
		),
	}
}

func createRoleListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.RbacV1().Roles(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.RbacV1().Roles(ns).Watch(context.TODO(), opts)
		},
	}
}

func wrapRoleFunc(f func(*rbacv1.Role) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		role := obj.(*rbacv1.Role)

		metricFamily := f(role)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descRoleLabelsDefaultLabels, []string{role.Namespace, role.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}
