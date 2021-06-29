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

package store

import (
	"context"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descCrdLabelsName          = "kube_crd_labels"
	descCrdLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descCrdLabelsDefaultLabels = []string{"crd", "scope"}
)

func crdMetricFamilies(allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			descCrdLabelsName,
			descCrdLabelsHelp,
			metric.Gauge,
			"",
			wrapCrdFunc(func(j *apiextensionsv1.CustomResourceDefinition) *metric.Family {
				labelKeys, labelValues := createLabelKeysValues(j.Labels, allowLabelsList)
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
		*generator.NewFamilyGenerator(
			"kube_crd_info",
			"Info about crd.",
			metric.Gauge,
			"",
			wrapCrdFunc(func(j *apiextensionsv1.CustomResourceDefinition) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"kind", "scope"},
							LabelValues: []string{j.Spec.Names.Kind, string(j.Spec.Scope)},
							Value:       1,
						},
					},
				}
			}),
		),
	}
}

func wrapCrdFunc(f func(*apiextensionsv1.CustomResourceDefinition) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		crd := obj.(*apiextensionsv1.CustomResourceDefinition)

		metricFamily := f(crd)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descCronJobLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{crd.Namespace, crd.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createCrdListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.ApiextensionsV1().CustomResourceDefinitions().Watch(context.TODO(), opts)
		},
	}
}
