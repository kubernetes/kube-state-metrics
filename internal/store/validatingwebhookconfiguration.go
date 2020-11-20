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
	"context"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descValidatingWebhookConfigurationDefaultLabels = []string{"namespace", "validatingwebhookconfiguration"}

	validatingWebhookConfigurationMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			"kube_validatingwebhookconfiguration_info",
			"Information about the ValidatingWebhookConfiguration.",
			metric.Gauge,
			"",
			wrapValidatingWebhookConfigurationFunc(func(vwc *admissionregistrationv1.ValidatingWebhookConfiguration) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: 1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_validatingwebhookconfiguration_created",
			"Unix creation timestamp.",
			metric.Gauge,
			"",
			wrapValidatingWebhookConfigurationFunc(func(vwc *admissionregistrationv1.ValidatingWebhookConfiguration) *metric.Family {
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
		),
		*generator.NewFamilyGenerator(
			"kube_validatingwebhookconfiguration_metadata_resource_version",
			"Resource version representing a specific version of the ValidatingWebhookConfiguration.",
			metric.Gauge,
			"",
			wrapValidatingWebhookConfigurationFunc(func(vwc *admissionregistrationv1.ValidatingWebhookConfiguration) *metric.Family {
				return &metric.Family{
					Metrics: resourceVersionMetric(vwc.ObjectMeta.ResourceVersion),
				}
			}),
		),
	}
)

func createValidatingWebhookConfigurationListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Watch(context.TODO(), opts)
		},
	}
}

func wrapValidatingWebhookConfigurationFunc(f func(*admissionregistrationv1.ValidatingWebhookConfiguration) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		mutatingWebhookConfiguration := obj.(*admissionregistrationv1.ValidatingWebhookConfiguration)

		metricFamily := f(mutatingWebhookConfiguration)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descValidatingWebhookConfigurationDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{mutatingWebhookConfiguration.Namespace, mutatingWebhookConfiguration.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}
