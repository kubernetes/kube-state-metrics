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

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	certv1 "k8s.io/api/certificates/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descPodCertificateRequestAnnotationsName     = "kube_podcertificaterequest_annotations"
	descPodCertificateRequestAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descPodCertificateRequestLabelsName          = "kube_podcertificaterequest_labels"
	descPodCertificateRequestLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descPodCertificateRequestLabelsDefaultLabels = []string{"namespace", "podcertificaterequest"}
)

func podCertificateRequestMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_podcertificaterequest_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapPodCertificateRequestFunc(func(r *certv1.PodCertificateRequest) *metric.Family {
				ms := []*metric.Metric{}
				if !r.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(r.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_podcertificaterequest_info",
			"Information about podcertificaterequest.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapPodCertificateRequestFunc(func(r *certv1.PodCertificateRequest) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"signer_name", "pod", "node", "service_account"},
							LabelValues: []string{r.Spec.SignerName, r.Spec.PodName, string(r.Spec.NodeName), r.Spec.ServiceAccountName},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_podcertificaterequest_status_condition",
			"The condition of a podcertificaterequest.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapPodCertificateRequestFunc(func(r *certv1.PodCertificateRequest) *metric.Family {
				ms := make([]*metric.Metric, 0, len(r.Status.Conditions)*len(conditionStatuses))
				for _, c := range r.Status.Conditions {
					for _, m := range addConditionMetrics(v1.ConditionStatus(c.Status)) {
						m.LabelKeys = []string{"condition", "status"}
						m.LabelValues = append([]string{c.Type}, m.LabelValues...)
						ms = append(ms, m)
					}
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_podcertificaterequest_status_not_before_timestamp_seconds",
			"Unix timestamp at which the issued certificate becomes valid.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapPodCertificateRequestFunc(func(r *certv1.PodCertificateRequest) *metric.Family {
				return timestampFamily(r.Status.NotBefore)
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_podcertificaterequest_status_not_after_timestamp_seconds",
			"Unix timestamp at which the issued certificate expires.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapPodCertificateRequestFunc(func(r *certv1.PodCertificateRequest) *metric.Family {
				return timestampFamily(r.Status.NotAfter)
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_podcertificaterequest_status_begin_refresh_at_timestamp_seconds",
			"Unix timestamp at which the kubelet should begin trying to refresh the certificate.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapPodCertificateRequestFunc(func(r *certv1.PodCertificateRequest) *metric.Family {
				return timestampFamily(r.Status.BeginRefreshAt)
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_podcertificaterequest_spec_max_expiration_seconds",
			"Maximum lifetime in seconds permitted for the requested certificate.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapPodCertificateRequestFunc(func(r *certv1.PodCertificateRequest) *metric.Family {
				if r.Spec.MaxExpirationSeconds == nil {
					return &metric.Family{}
				}
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(*r.Spec.MaxExpirationSeconds),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descPodCertificateRequestAnnotationsName,
			descPodCertificateRequestAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapPodCertificateRequestFunc(func(r *certv1.PodCertificateRequest) *metric.Family {
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
			descPodCertificateRequestLabelsName,
			descPodCertificateRequestLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapPodCertificateRequestFunc(func(r *certv1.PodCertificateRequest) *metric.Family {
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
	}
}

// timestampFamily returns a family with a single Unix timestamp metric, or an
// empty family when t is unset.
func timestampFamily(t *metav1.Time) *metric.Family {
	if t == nil || t.IsZero() {
		return &metric.Family{}
	}
	return &metric.Family{
		Metrics: []*metric.Metric{
			{
				Value: float64(t.Unix()),
			},
		},
	}
}

func wrapPodCertificateRequestFunc(f func(*certv1.PodCertificateRequest) *metric.Family) func(any) *metric.Family {
	return func(obj any) *metric.Family {
		podCertificateRequest := obj.(*certv1.PodCertificateRequest)

		metricFamily := f(podCertificateRequest)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descPodCertificateRequestLabelsDefaultLabels, []string{podCertificateRequest.Namespace, podCertificateRequest.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createPodCertificateRequestListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CertificatesV1().PodCertificateRequests(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.CertificatesV1().PodCertificateRequests(ns).Watch(context.TODO(), opts)
		},
	}
}
