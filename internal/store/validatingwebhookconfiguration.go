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
	admissionregistration "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/pkg/metric"
)

var (
	descValidatingWebhookConfigurationHelp          = "Kubernetes labels converted to Prometheus labels."
	descValidatingWebhookConfigurationDefaultLabels = []string{"namespace", "validatingwebhookconfiguration"}

	validatingWebhookConfigurationMetricFamilies = []metric.FamilyGenerator{
		{
			Name: "kube_validatingwebhookconfiguration_info",
			Type: metric.Gauge,
			Help: "Information about the ValidatingWebhookConfiguration.",
			GenerateFunc: wrapValidatingWebhookConfigurationFunc(func(vwc *admissionregistration.ValidatingWebhookConfiguration) *metric.Family {
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
			Name: "kube_validatingwebhookconfiguration_created",
			Type: metric.Gauge,
			Help: "Unix creation timestamp.",
			GenerateFunc: wrapValidatingWebhookConfigurationFunc(func(vwc *admissionregistration.ValidatingWebhookConfiguration) *metric.Family {
				ms := []*metric.Metric{}

				if !vwc.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(vwc.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_validatingwebhookconfiguration_metadata_resource_version",
			Type: metric.Gauge,
			Help: "Resource version representing a specific version of the ValidatingWebhookConfiguration.",
			GenerateFunc: wrapValidatingWebhookConfigurationFunc(func(vwc *admissionregistration.ValidatingWebhookConfiguration) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"resource_version"},
							LabelValues: []string{vwc.ObjectMeta.ResourceVersion},
							Value:       1,
						},
					},
				}
			}),
		},
	}
)

func createValidatingWebhookConfigurationListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Watch(opts)
		},
	}
}

func wrapValidatingWebhookConfigurationFunc(f func(*admissionregistration.ValidatingWebhookConfiguration) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		mutatingWebhookConfiguration := obj.(*admissionregistration.ValidatingWebhookConfiguration)

		metricFamily := f(mutatingWebhookConfiguration)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descValidatingWebhookConfigurationDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{mutatingWebhookConfiguration.Namespace, mutatingWebhookConfiguration.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}
