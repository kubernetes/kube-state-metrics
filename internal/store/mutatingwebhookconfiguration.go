/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/pkg/metric"
	generator "k8s.io/kube-state-metrics/pkg/metric_generator"
)

var (
	descMutatingWebhookConfigurationHelp          = "Kubernetes labels converted to Prometheus labels."
	descMutatingWebhookConfigurationDefaultLabels = []string{"namespace", "mutatingwebhookconfiguration"}

	mutatingWebhookConfigurationMetricFamilies = []generator.FamilyGenerator{
		{
			Name: "kube_mutatingwebhookconfiguration_info",
			Type: metric.Gauge,
			Help: "Information about the MutatingWebhookConfiguration.",
			GenerateFunc: wrapMutatingWebhookConfigurationFunc(func(mwc *admissionregistrationv1.MutatingWebhookConfiguration) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: 1,
						},
					},
				}
			}),
		},
		{
			Name: "kube_mutatingwebhookconfiguration_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp.",
			GenerateFunc: wrapMutatingWebhookConfigurationFunc(func(mwc *admissionregistrationv1.MutatingWebhookConfiguration) *metric.Family {
				ms := []*metric.Metric{}

				if !mwc.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(mwc.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_mutatingwebhookconfiguration_metadata_resource_version",
			Type: metric.Gauge,
			Help: "Resource version representing a specific version of the MutatingWebhookConfiguration.",
			GenerateFunc: wrapMutatingWebhookConfigurationFunc(func(mwc *admissionregistrationv1.MutatingWebhookConfiguration) *metric.Family {
				return &metric.Family{
					Metrics: resourceVersionMetric(mwc.ObjectMeta.ResourceVersion),
				}
			}),
		},
	}
)

func createMutatingWebhookConfigurationListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Watch(opts)
		},
	}
}

func wrapMutatingWebhookConfigurationFunc(f func(*admissionregistrationv1.MutatingWebhookConfiguration) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		mutatingWebhookConfiguration := obj.(*admissionregistrationv1.MutatingWebhookConfiguration)

		metricFamily := f(mutatingWebhookConfiguration)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descMutatingWebhookConfigurationDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{mutatingWebhookConfiguration.Namespace, mutatingWebhookConfiguration.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}
