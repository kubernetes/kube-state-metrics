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
	"testing"
	"time"

	certv1 "k8s.io/api/certificates/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestCsrStore(t *testing.T) {
	const metadata = `
		# HELP kube_certificatesigningrequest_labels [STABLE] Kubernetes labels converted to Prometheus labels.
		# TYPE kube_certificatesigningrequest_labels gauge
		# HELP kube_certificatesigningrequest_created [STABLE] Unix creation timestamp
		# TYPE kube_certificatesigningrequest_created gauge
		# HELP kube_certificatesigningrequest_condition [STABLE] The number of each certificatesigningrequest condition
		# TYPE kube_certificatesigningrequest_condition gauge
		# HELP kube_certificatesigningrequest_cert_length [STABLE] Length of the issued cert
		# TYPE kube_certificatesigningrequest_cert_length gauge
	`
	cases := []generateMetricsTestCase{
		{
			Obj: &certv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "certificate-test",
					Generation: 1,
					Labels: map[string]string{
						"cert": "test",
					},
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Status: certv1.CertificateSigningRequestStatus{},
				Spec: certv1.CertificateSigningRequestSpec{
					SignerName: "signer",
				},
			},
			Want: metadata + `
				kube_certificatesigningrequest_created{certificatesigningrequest="certificate-test",signer_name="signer"} 1.5e+09
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="approved"} 0
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="denied"} 0
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="failed"} 0
				kube_certificatesigningrequest_cert_length{certificatesigningrequest="certificate-test",signer_name="signer"} 0
`,
			MetricNames: []string{"kube_certificatesigningrequest_created", "kube_certificatesigningrequest_condition", "kube_certificatesigningrequest_labels", "kube_certificatesigningrequest_cert_length"},
		},
		{
			Obj: &certv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "certificate-test",
					Generation: 1,
					Labels: map[string]string{
						"cert": "test",
					},
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Status: certv1.CertificateSigningRequestStatus{
					Conditions: []certv1.CertificateSigningRequestCondition{
						{
							Type: certv1.CertificateDenied,
						},
					},
				},
				Spec: certv1.CertificateSigningRequestSpec{
					SignerName: "signer",
				},
			},
			Want: metadata + `
				kube_certificatesigningrequest_created{certificatesigningrequest="certificate-test",signer_name="signer"} 1.5e+09
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="approved"} 0
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="denied"} 1
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="failed"} 0
				kube_certificatesigningrequest_cert_length{certificatesigningrequest="certificate-test",signer_name="signer"} 0
`,
			MetricNames: []string{"kube_certificatesigningrequest_created", "kube_certificatesigningrequest_condition", "kube_certificatesigningrequest_labels", "kube_certificatesigningrequest_cert_length"},
		},
		{
			Obj: &certv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "certificate-test",
					Generation: 1,
					Labels: map[string]string{
						"cert": "test",
					},
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Status: certv1.CertificateSigningRequestStatus{
					Conditions: []certv1.CertificateSigningRequestCondition{
						{
							Type: certv1.CertificateApproved,
						},
					},
				},
				Spec: certv1.CertificateSigningRequestSpec{
					SignerName: "signer",
				},
			},
			Want: metadata + `
				kube_certificatesigningrequest_created{certificatesigningrequest="certificate-test",signer_name="signer"} 1.5e+09
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="approved"} 1
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="denied"} 0
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="failed"} 0
				kube_certificatesigningrequest_cert_length{certificatesigningrequest="certificate-test",signer_name="signer"} 0
`,
			MetricNames: []string{"kube_certificatesigningrequest_created", "kube_certificatesigningrequest_condition", "kube_certificatesigningrequest_labels", "kube_certificatesigningrequest_cert_length"},
		},
		{
			Obj: &certv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "certificate-test",
					Generation: 1,
					Labels: map[string]string{
						"cert": "test",
					},
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Status: certv1.CertificateSigningRequestStatus{
					Conditions: []certv1.CertificateSigningRequestCondition{
						{
							Type: certv1.CertificateFailed,
						},
					},
				},
				Spec: certv1.CertificateSigningRequestSpec{
					SignerName: "signer",
				},
			},
			Want: metadata + `
				kube_certificatesigningrequest_created{certificatesigningrequest="certificate-test",signer_name="signer"} 1.5e+09
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="approved"} 0
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="denied"} 0
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="failed"} 1
				kube_certificatesigningrequest_cert_length{certificatesigningrequest="certificate-test",signer_name="signer"} 0
`,
			MetricNames: []string{"kube_certificatesigningrequest_created", "kube_certificatesigningrequest_condition", "kube_certificatesigningrequest_labels", "kube_certificatesigningrequest_cert_length"},
		},
		{
			Obj: &certv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "certificate-test",
					Generation: 1,
					Labels: map[string]string{
						"cert": "test",
					},
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Spec: certv1.CertificateSigningRequestSpec{
					SignerName: "signer",
				},
				Status: certv1.CertificateSigningRequestStatus{
					Certificate: []byte("just for test"),
					Conditions: []certv1.CertificateSigningRequestCondition{
						{
							Type: certv1.CertificateApproved,
						},
					},
				},
			},
			Want: metadata + `
				kube_certificatesigningrequest_created{certificatesigningrequest="certificate-test",signer_name="signer"} 1.5e+09
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="approved"} 1
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="denied"} 0
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="failed"} 0
				kube_certificatesigningrequest_cert_length{certificatesigningrequest="certificate-test",signer_name="signer"} 13
`,
			MetricNames: []string{"kube_certificatesigningrequest_created", "kube_certificatesigningrequest_condition", "kube_certificatesigningrequest_labels", "kube_certificatesigningrequest_cert_length"},
		},
		{
			Obj: &certv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "certificate-test",
					Generation: 1,
					Labels: map[string]string{
						"cert": "test",
					},
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Spec: certv1.CertificateSigningRequestSpec{
					SignerName: "signer",
				},
				Status: certv1.CertificateSigningRequestStatus{
					Conditions: []certv1.CertificateSigningRequestCondition{
						{
							Type: certv1.CertificateApproved,
						},
						{
							Type: certv1.CertificateDenied,
						},
						{
							Type: certv1.CertificateFailed,
						},
					},
				},
			},
			Want: metadata + `
				kube_certificatesigningrequest_created{certificatesigningrequest="certificate-test",signer_name="signer"} 1.5e+09
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="approved"} 1
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="denied"} 1
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="failed"} 1
				kube_certificatesigningrequest_cert_length{certificatesigningrequest="certificate-test",signer_name="signer"} 0
`,
			MetricNames: []string{"kube_certificatesigningrequest_created", "kube_certificatesigningrequest_condition", "kube_certificatesigningrequest_labels", "kube_certificatesigningrequest_cert_length"},
		},
		{
			Obj: &certv1.CertificateSigningRequest{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "certificate-test",
					Generation: 1,
					Labels: map[string]string{
						"cert": "test",
					},
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Spec: certv1.CertificateSigningRequestSpec{
					SignerName: "signer",
				},
				Status: certv1.CertificateSigningRequestStatus{
					Conditions: []certv1.CertificateSigningRequestCondition{
						{
							Type: certv1.CertificateApproved,
						},
						{
							Type: certv1.CertificateDenied,
						},
						{
							Type: certv1.CertificateApproved,
						},
						{
							Type: certv1.CertificateDenied,
						},
						{
							Type: certv1.CertificateFailed,
						},
						{
							Type: certv1.CertificateFailed,
						},
					},
				},
			},
			Want: metadata + `
				kube_certificatesigningrequest_created{certificatesigningrequest="certificate-test",signer_name="signer"} 1.5e+09
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="approved"} 2
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="denied"} 2
				kube_certificatesigningrequest_condition{certificatesigningrequest="certificate-test",signer_name="signer",condition="failed"} 2
				kube_certificatesigningrequest_cert_length{certificatesigningrequest="certificate-test",signer_name="signer"} 0
`,
			MetricNames: []string{"kube_certificatesigningrequest_created", "kube_certificatesigningrequest_condition", "kube_certificatesigningrequest_labels", "kube_certificatesigningrequest_cert_length"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(csrMetricFamilies(nil, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(csrMetricFamilies(nil, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected error when collecting result in %vth run:\n%s", i, err)
		}
	}
}
