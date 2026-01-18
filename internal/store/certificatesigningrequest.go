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

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	certv1 "k8s.io/api/certificates/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descCSRAnnotationsName     = "kube_certificatesigningrequest_annotations"
	descCSRAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descCSRLabelsName          = "kube_certificatesigningrequest_labels"
	descCSRLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descCSRLabelsDefaultLabels = []string{"certificatesigningrequest", "signer_name"}
)

func csrMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			descCSRAnnotationsName,
			descCSRAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapCSRFunc(func(j *certv1.CertificateSigningRequest) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", j.Annotations, allowAnnotationsList)
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
			descCSRLabelsName,
			descCSRLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapCSRFunc(func(j *certv1.CertificateSigningRequest) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", j.Labels, allowLabelsList)
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
			"kube_certificatesigningrequest_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapCSRFunc(func(csr *certv1.CertificateSigningRequest) *metric.Family {
				ms := []*metric.Metric{}
				if !csr.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(csr.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_certificatesigningrequest_condition",
			"The number of each certificatesigningrequest condition",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapCSRFunc(func(csr *certv1.CertificateSigningRequest) *metric.Family {
				return &metric.Family{
					Metrics: addCSRConditionMetrics(csr.Status),
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_certificatesigningrequest_cert_length",
			"Length of the issued cert",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapCSRFunc(func(csr *certv1.CertificateSigningRequest) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(len(csr.Status.Certificate)),
						},
					},
				}
			}),
		),
	}
}

func wrapCSRFunc(f func(*certv1.CertificateSigningRequest) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		csr := obj.(*certv1.CertificateSigningRequest)
		metricFamily := f(csr)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descCSRLabelsDefaultLabels, []string{csr.Name, csr.Spec.SignerName}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createCSRListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CertificatesV1().CertificateSigningRequests().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CertificatesV1().CertificateSigningRequests().Watch(context.TODO(), opts)
		},
	}
}

// addCSRConditionMetrics generates one metric for each possible csr condition status
func addCSRConditionMetrics(cs certv1.CertificateSigningRequestStatus) []*metric.Metric {
	cApproved := 0
	cDenied := 0
	cFailed := 0
	for _, s := range cs.Conditions {
		if s.Type == certv1.CertificateApproved {
			cApproved++
		}
		if s.Type == certv1.CertificateDenied {
			cDenied++
		}
		if s.Type == certv1.CertificateFailed {
			cFailed++
		}
	}

	return []*metric.Metric{
		{
			LabelValues: []string{"approved"},
			Value:       float64(cApproved),
			LabelKeys:   []string{"condition"},
		},
		{
			LabelValues: []string{"denied"},
			Value:       float64(cDenied),
			LabelKeys:   []string{"condition"},
		},
		{
			LabelValues: []string{"failed"},
			Value:       float64(cFailed),
			LabelKeys:   []string{"condition"},
		},
	}
}
