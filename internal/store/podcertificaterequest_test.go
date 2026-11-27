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
	"testing"

	certv1 "k8s.io/api/certificates/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestPodCertificateRequestStore(t *testing.T) {
	const metadata = `
		# HELP kube_podcertificaterequest_created Unix creation timestamp
		# TYPE kube_podcertificaterequest_created gauge
		# HELP kube_podcertificaterequest_info Information about podcertificaterequest.
		# TYPE kube_podcertificaterequest_info gauge
		# HELP kube_podcertificaterequest_status_condition The condition of a podcertificaterequest.
		# TYPE kube_podcertificaterequest_status_condition gauge
		# HELP kube_podcertificaterequest_status_not_before_timestamp_seconds Unix timestamp at which the issued certificate becomes valid.
		# TYPE kube_podcertificaterequest_status_not_before_timestamp_seconds gauge
		# HELP kube_podcertificaterequest_status_not_after_timestamp_seconds Unix timestamp at which the issued certificate expires.
		# TYPE kube_podcertificaterequest_status_not_after_timestamp_seconds gauge
		# HELP kube_podcertificaterequest_status_begin_refresh_at_timestamp_seconds Unix timestamp at which the kubelet should begin trying to refresh the certificate.
		# TYPE kube_podcertificaterequest_status_begin_refresh_at_timestamp_seconds gauge
		# HELP kube_podcertificaterequest_spec_max_expiration_seconds Maximum lifetime in seconds permitted for the requested certificate.
		# TYPE kube_podcertificaterequest_spec_max_expiration_seconds gauge
		# HELP kube_podcertificaterequest_annotations Kubernetes annotations converted to Prometheus labels.
		# TYPE kube_podcertificaterequest_annotations gauge
		# HELP kube_podcertificaterequest_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_podcertificaterequest_labels gauge
	`
	allMetricNames := []string{
		"kube_podcertificaterequest_created",
		"kube_podcertificaterequest_info",
		"kube_podcertificaterequest_status_condition",
		"kube_podcertificaterequest_status_not_before_timestamp_seconds",
		"kube_podcertificaterequest_status_not_after_timestamp_seconds",
		"kube_podcertificaterequest_status_begin_refresh_at_timestamp_seconds",
		"kube_podcertificaterequest_spec_max_expiration_seconds",
		"kube_podcertificaterequest_annotations",
		"kube_podcertificaterequest_labels",
	}

	cases := []generateMetricsTestCase{
		{
			// Full object: issued certificate with all timestamps set.
			Obj: &certv1.PodCertificateRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "pcr-issued",
					Namespace:         "default",
					CreationTimestamp: metav1.Unix(1500000000, 0),
					Labels: map[string]string{
						"app": "web",
					},
				},
				Spec: certv1.PodCertificateRequestSpec{
					SignerName:           "example.com/signer",
					PodName:              "web-0",
					PodUID:               "pod-uid",
					ServiceAccountName:   "web-sa",
					ServiceAccountUID:    "sa-uid",
					NodeName:             "node-1",
					NodeUID:              "node-uid",
					MaxExpirationSeconds: ptr.To[int32](86400),
				},
				Status: certv1.PodCertificateRequestStatus{
					Conditions: []metav1.Condition{
						{
							Type:    certv1.PodCertificateRequestConditionTypeIssued,
							Status:  metav1.ConditionTrue,
							Reason:  "Issued",
							Message: "free text that must not become a label",
						},
					},
					CertificateChain: "-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n",
					NotBefore:        ptr.To(metav1.Unix(1500000100, 0)),
					BeginRefreshAt:   ptr.To(metav1.Unix(1500050000, 0)),
					NotAfter:         ptr.To(metav1.Unix(1500086500, 0)),
				},
			},
			Want: metadata + `
				kube_podcertificaterequest_created{namespace="default",podcertificaterequest="pcr-issued"} 1.5e+09
				kube_podcertificaterequest_info{namespace="default",podcertificaterequest="pcr-issued",signer_name="example.com/signer",pod="web-0",node="node-1",service_account="web-sa"} 1
				kube_podcertificaterequest_status_condition{namespace="default",podcertificaterequest="pcr-issued",condition="Issued",status="true"} 1
				kube_podcertificaterequest_status_condition{namespace="default",podcertificaterequest="pcr-issued",condition="Issued",status="false"} 0
				kube_podcertificaterequest_status_condition{namespace="default",podcertificaterequest="pcr-issued",condition="Issued",status="unknown"} 0
				kube_podcertificaterequest_status_not_before_timestamp_seconds{namespace="default",podcertificaterequest="pcr-issued"} 1.5000001e+09
				kube_podcertificaterequest_status_not_after_timestamp_seconds{namespace="default",podcertificaterequest="pcr-issued"} 1.5000865e+09
				kube_podcertificaterequest_status_begin_refresh_at_timestamp_seconds{namespace="default",podcertificaterequest="pcr-issued"} 1.50005e+09
				kube_podcertificaterequest_spec_max_expiration_seconds{namespace="default",podcertificaterequest="pcr-issued"} 86400
			`,
			MetricNames: allMetricNames,
		},
		{
			// Denied request: condition present, no certificate timestamps.
			Obj: &certv1.PodCertificateRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pcr-denied",
					Namespace: "ns1",
				},
				Spec: certv1.PodCertificateRequestSpec{
					SignerName:         "example.com/signer",
					PodName:            "web-1",
					ServiceAccountName: "web-sa",
					NodeName:           "node-2",
				},
				Status: certv1.PodCertificateRequestStatus{
					Conditions: []metav1.Condition{
						{
							Type:   certv1.PodCertificateRequestConditionTypeDenied,
							Status: metav1.ConditionTrue,
							Reason: certv1.PodCertificateRequestConditionUnsupportedKeyType,
						},
					},
				},
			},
			Want: metadata + `
				kube_podcertificaterequest_info{namespace="ns1",podcertificaterequest="pcr-denied",signer_name="example.com/signer",pod="web-1",node="node-2",service_account="web-sa"} 1
				kube_podcertificaterequest_status_condition{namespace="ns1",podcertificaterequest="pcr-denied",condition="Denied",status="true"} 1
				kube_podcertificaterequest_status_condition{namespace="ns1",podcertificaterequest="pcr-denied",condition="Denied",status="false"} 0
				kube_podcertificaterequest_status_condition{namespace="ns1",podcertificaterequest="pcr-denied",condition="Denied",status="unknown"} 0
			`,
			MetricNames: allMetricNames,
		},
		{
			// Minimal object: only metadata, everything optional unset.
			Obj: &certv1.PodCertificateRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pcr-minimal",
					Namespace: "ns2",
				},
			},
			Want: metadata + `
				kube_podcertificaterequest_info{namespace="ns2",podcertificaterequest="pcr-minimal",signer_name="",pod="",node="",service_account=""} 1
			`,
			MetricNames: allMetricNames,
		},
		{
			AllowAnnotationsList: []string{"team"},
			AllowLabelsList:      []string{"app"},
			Obj: &certv1.PodCertificateRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pcr-metadata",
					Namespace: "ns3",
					Labels: map[string]string{
						"app":   "web",
						"other": "dropped",
					},
					Annotations: map[string]string{
						"team":  "security",
						"other": "dropped",
					},
				},
			},
			Want: `
				# HELP kube_podcertificaterequest_annotations Kubernetes annotations converted to Prometheus labels.
				# TYPE kube_podcertificaterequest_annotations gauge
				# HELP kube_podcertificaterequest_labels Kubernetes labels converted to Prometheus labels.
				# TYPE kube_podcertificaterequest_labels gauge
				kube_podcertificaterequest_annotations{namespace="ns3",podcertificaterequest="pcr-metadata",annotation_team="security"} 1
				kube_podcertificaterequest_labels{namespace="ns3",podcertificaterequest="pcr-metadata",label_app="web"} 1
			`,
			MetricNames: []string{
				"kube_podcertificaterequest_annotations",
				"kube_podcertificaterequest_labels",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(podCertificateRequestMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(podCertificateRequestMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
