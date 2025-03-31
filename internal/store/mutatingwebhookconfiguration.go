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
	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descMutatingWebhookConfigurationDefaultLabels = []string{"namespace", "mutatingwebhookconfiguration"}

	mutatingWebhookConfigurationMetricFamilies = []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_mutatingwebhookconfiguration_info",
			"Information about the MutatingWebhookConfiguration.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapMutatingWebhookConfigurationFunc(func(_ *admissionregistrationv1.MutatingWebhookConfiguration) *metric.Family {
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
			"kube_mutatingwebhookconfiguration_created",
			"Unix creation timestamp.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapMutatingWebhookConfigurationFunc(func(mwc *admissionregistrationv1.MutatingWebhookConfiguration) *metric.Family {
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
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_mutatingwebhookconfiguration_metadata_resource_version",
			"Resource version representing a specific version of the MutatingWebhookConfiguration.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapMutatingWebhookConfigurationFunc(func(mwc *admissionregistrationv1.MutatingWebhookConfiguration) *metric.Family {
				return &metric.Family{
					Metrics: resourceVersionMetric(mwc.ResourceVersion),
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_mutatingwebhookconfiguration_webhook_clientconfig_service",
			"Service used by the apiserver to connect to a mutating webhook.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapMutatingWebhookConfigurationFunc(func(mwc *admissionregistrationv1.MutatingWebhookConfiguration) *metric.Family {
				ms := []*metric.Metric{}
				for _, webhook := range mwc.Webhooks {
					var serviceName, serviceNamespace string
					if webhook.ClientConfig.Service != nil {
						serviceName = webhook.ClientConfig.Service.Name
						serviceNamespace = webhook.ClientConfig.Service.Namespace
					}

					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"webhook_name", "service_name", "service_namespace"},
						LabelValues: []string{webhook.Name, serviceName, serviceNamespace},
						Value:       1,
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
)

func createMutatingWebhookConfigurationListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Watch(context.TODO(), opts)
		},
	}
}

func wrapMutatingWebhookConfigurationFunc(f func(*admissionregistrationv1.MutatingWebhookConfiguration) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		mutatingWebhookConfiguration := obj.(*admissionregistrationv1.MutatingWebhookConfiguration)

		metricFamily := f(mutatingWebhookConfiguration)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descMutatingWebhookConfigurationDefaultLabels, []string{mutatingWebhookConfiguration.Namespace, mutatingWebhookConfiguration.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}
