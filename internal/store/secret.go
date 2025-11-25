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
	"strconv"

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
	descSecretAnnotationsName     = "kube_secret_annotations"
	descSecretAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels." //nolint:gosec
	descSecretLabelsName          = "kube_secret_labels"
	descSecretLabelsHelp          = "Kubernetes labels converted to Prometheus labels." //nolint:gosec
	descSecretLabelsDefaultLabels = []string{"namespace", "secret"}
)

func secretMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_secret_info",
			"Information about secret.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapSecretFunc(func(_ *v1.Secret) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: 1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_secret_type",
			"Type about secret.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapSecretFunc(func(s *v1.Secret) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"type"},
							LabelValues: []string{string(s.Type)},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descSecretAnnotationsName,
			descSecretAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapSecretFunc(func(s *v1.Secret) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", s.Annotations, allowAnnotationsList)
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
			descSecretLabelsName,
			descSecretLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapSecretFunc(func(s *v1.Secret) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", s.Labels, allowLabelsList)
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
			"kube_secret_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapSecretFunc(func(s *v1.Secret) *metric.Family {
				ms := []*metric.Metric{}

				if !s.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(s.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_secret_metadata_resource_version",
			"Resource version representing a specific version of secret.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapSecretFunc(func(s *v1.Secret) *metric.Family {
				return &metric.Family{
					Metrics: resourceVersionMetric(s.ResourceVersion),
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_secret_owner",
			"Information about the Secret's owner.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapSecretFunc(func(j *v1.Secret) *metric.Family {
				labelKeys := []string{"owner_kind", "owner_name", "owner_is_controller"}

				owners := j.GetOwnerReferences()

				if len(owners) == 0 {
					return &metric.Family{
						Metrics: []*metric.Metric{
							{
								LabelKeys:   labelKeys,
								LabelValues: []string{"", "", ""},
								Value:       1,
							},
						},
					}
				}

				ms := make([]*metric.Metric, len(owners))

				for i, owner := range owners {
					if owner.Controller != nil {
						ms[i] = &metric.Metric{
							LabelKeys:   labelKeys,
							LabelValues: []string{owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller)},
							Value:       1,
						}
					} else {
						ms[i] = &metric.Metric{
							LabelKeys:   labelKeys,
							LabelValues: []string{owner.Kind, owner.Name, "false"},
							Value:       1,
						}
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}

}

func wrapSecretFunc(f func(*v1.Secret) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		secret := obj.(*v1.Secret)

		metricFamily := f(secret)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descSecretLabelsDefaultLabels, []string{secret.Namespace, secret.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createSecretListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().Secrets(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().Secrets(ns).Watch(context.TODO(), opts)
		},
	}
}
