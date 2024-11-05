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
	descServiceAccountLabelsDefaultLabels = []string{"namespace", "serviceaccount", "uid"}
)

func serviceAccountMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		createServiceAccountInfoFamilyGenerator(),
		createServiceAccountCreatedFamilyGenerator(),
		createServiceAccountDeletedFamilyGenerator(),
		createServiceAccountSecretFamilyGenerator(),
		createServiceAccountImagePullSecretFamilyGenerator(),
		createServiceAccountAnnotationsGenerator(allowAnnotationsList),
		createServiceAccountLabelsGenerator(allowLabelsList),
	}
}

func createServiceAccountInfoFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_serviceaccount_info",
		"Information about a service account",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapServiceAccountFunc(func(sa *v1.ServiceAccount) *metric.Family {
			var automountToken string

			if sa.AutomountServiceAccountToken != nil {
				automountToken = strconv.FormatBool(*sa.AutomountServiceAccountToken)
			}

			return &metric.Family{
				Metrics: []*metric.Metric{{
					LabelKeys:   []string{"automount_token"},
					LabelValues: []string{automountToken},
					Value:       1,
				}},
			}
		}),
	)
}

func createServiceAccountCreatedFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_serviceaccount_created",
		"Unix creation timestamp",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapServiceAccountFunc(func(sa *v1.ServiceAccount) *metric.Family {
			var ms []*metric.Metric

			if !sa.CreationTimestamp.IsZero() {
				ms = append(ms, &metric.Metric{
					LabelKeys:   []string{},
					LabelValues: []string{},
					Value:       float64(sa.CreationTimestamp.Unix()),
				})
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createServiceAccountDeletedFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_serviceaccount_deleted",
		"Unix deletion timestamp",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapServiceAccountFunc(func(sa *v1.ServiceAccount) *metric.Family {
			var ms []*metric.Metric

			if sa.DeletionTimestamp != nil && !sa.DeletionTimestamp.IsZero() {
				ms = append(ms, &metric.Metric{
					LabelKeys:   []string{},
					LabelValues: []string{},
					Value:       float64(sa.DeletionTimestamp.Unix()),
				})
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createServiceAccountSecretFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_serviceaccount_secret",
		"Secret being referenced by a service account",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapServiceAccountFunc(func(sa *v1.ServiceAccount) *metric.Family {
			var ms []*metric.Metric

			for _, s := range sa.Secrets {
				ms = append(ms, &metric.Metric{
					LabelKeys:   []string{"name"},
					LabelValues: []string{s.Name},
					Value:       1,
				})
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createServiceAccountImagePullSecretFamilyGenerator() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_serviceaccount_image_pull_secret",
		"Secret being referenced by a service account for the purpose of pulling images",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapServiceAccountFunc(func(sa *v1.ServiceAccount) *metric.Family {
			var ms []*metric.Metric

			for _, s := range sa.ImagePullSecrets {
				ms = append(ms, &metric.Metric{
					LabelKeys:   []string{"name"},
					LabelValues: []string{s.Name},
					Value:       1,
				})
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createServiceAccountAnnotationsGenerator(allowAnnotations []string) generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_serviceaccount_annotations",
		"Kubernetes annotations converted to Prometheus labels.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapServiceAccountFunc(func(sa *v1.ServiceAccount) *metric.Family {
			if len(allowAnnotations) == 0 {
				return &metric.Family{}
			}
			annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", sa.Annotations, allowAnnotations)
			m := metric.Metric{
				LabelKeys:   annotationKeys,
				LabelValues: annotationValues,
				Value:       1,
			}
			return &metric.Family{
				Metrics: []*metric.Metric{&m},
			}
		}),
	)
}

func createServiceAccountLabelsGenerator(allowLabelsList []string) generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_serviceaccount_labels",
		"Kubernetes labels converted to Prometheus labels.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapServiceAccountFunc(func(sa *v1.ServiceAccount) *metric.Family {
			if len(allowLabelsList) == 0 {
				return &metric.Family{}
			}
			labelKeys, labelValues := createPrometheusLabelKeysValues("label", sa.Labels, allowLabelsList)
			m := metric.Metric{
				LabelKeys:   labelKeys,
				LabelValues: labelValues,
				Value:       1,
			}
			return &metric.Family{
				Metrics: []*metric.Metric{&m},
			}
		}),
	)
}

func wrapServiceAccountFunc(f func(*v1.ServiceAccount) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		serviceAccount := obj.(*v1.ServiceAccount)

		metricFamily := f(serviceAccount)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descServiceAccountLabelsDefaultLabels, []string{serviceAccount.Namespace, serviceAccount.Name, string(serviceAccount.UID)}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createServiceAccountListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().ServiceAccounts(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CoreV1().ServiceAccounts(ns).Watch(context.TODO(), opts)
		},
	}
}
