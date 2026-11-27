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
	"crypto/x509"
	"encoding/pem"
	"time"

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

const pemTypeCertificate = "CERTIFICATE"

var (
	descClusterTrustBundleAnnotationsName     = "kube_clustertrustbundle_annotations"
	descClusterTrustBundleAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descClusterTrustBundleLabelsName          = "kube_clustertrustbundle_labels"
	descClusterTrustBundleLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descClusterTrustBundleLabelsDefaultLabels = []string{"clustertrustbundle"}
)

func clusterTrustBundleMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_clustertrustbundle_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterTrustBundleFunc(func(b *certv1.ClusterTrustBundle) *metric.Family {
				ms := []*metric.Metric{}
				if !b.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(b.CreationTimestamp.Unix()),
					})
				}
				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_clustertrustbundle_info",
			"Information about clustertrustbundle.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterTrustBundleFunc(func(b *certv1.ClusterTrustBundle) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"signer_name"},
							LabelValues: []string{b.Spec.SignerName},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_clustertrustbundle_certificates",
			"Number of CERTIFICATE PEM blocks in the clustertrustbundle.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterTrustBundleFunc(func(b *certv1.ClusterTrustBundle) *metric.Family {
				count := 0
				rest := []byte(b.Spec.TrustBundle)
				for {
					var block *pem.Block
					block, rest = pem.Decode(rest)
					if block == nil {
						break
					}
					if block.Type == pemTypeCertificate {
						count++
					}
				}
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(count),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_clustertrustbundle_earliest_expiry_timestamp_seconds",
			"Unix timestamp of the earliest NotAfter across the parsable certificates in the clustertrustbundle.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterTrustBundleFunc(func(b *certv1.ClusterTrustBundle) *metric.Family {
				earliest, ok := earliestCertificateExpiry(b.Spec.TrustBundle)
				if !ok {
					return &metric.Family{}
				}
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(earliest.Unix()),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descClusterTrustBundleAnnotationsName,
			descClusterTrustBundleAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterTrustBundleFunc(func(b *certv1.ClusterTrustBundle) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", b.Annotations, allowAnnotationsList)
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
			descClusterTrustBundleLabelsName,
			descClusterTrustBundleLabelsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapClusterTrustBundleFunc(func(b *certv1.ClusterTrustBundle) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", b.Labels, allowLabelsList)
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

// earliestCertificateExpiry returns the minimum NotAfter across all
// CERTIFICATE PEM blocks in bundle that parse as X.509 certificates. Blocks
// that are not certificates or fail to parse are skipped. ok is false when no
// certificate could be parsed.
func earliestCertificateExpiry(bundle string) (earliest time.Time, ok bool) {
	rest := []byte(bundle)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			return earliest, ok
		}
		if block.Type != pemTypeCertificate {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		if !ok || cert.NotAfter.Before(earliest) {
			earliest = cert.NotAfter
			ok = true
		}
	}
}

func wrapClusterTrustBundleFunc(f func(*certv1.ClusterTrustBundle) *metric.Family) func(any) *metric.Family {
	return func(obj any) *metric.Family {
		clusterTrustBundle := obj.(*certv1.ClusterTrustBundle)

		metricFamily := f(clusterTrustBundle)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descClusterTrustBundleLabelsDefaultLabels, []string{clusterTrustBundle.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createClusterTrustBundleListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CertificatesV1().ClusterTrustBundles().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CertificatesV1().ClusterTrustBundles().Watch(context.TODO(), opts)
		},
	}
}
