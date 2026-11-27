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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	certv1 "k8s.io/api/certificates/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

// newTestCACertPEM returns a PEM-encoded self-signed CA certificate that
// expires at notAfter.
func newTestCACertPEM(t *testing.T, serial int64, notAfter time.Time) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             notAfter.Add(-365 * 24 * time.Hour),
		NotAfter:              notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestClusterTrustBundleStore(t *testing.T) {
	const metadata = `
		# HELP kube_clustertrustbundle_created Unix creation timestamp
		# TYPE kube_clustertrustbundle_created gauge
		# HELP kube_clustertrustbundle_info Information about clustertrustbundle.
		# TYPE kube_clustertrustbundle_info gauge
		# HELP kube_clustertrustbundle_certificates Number of CERTIFICATE PEM blocks in the clustertrustbundle.
		# TYPE kube_clustertrustbundle_certificates gauge
		# HELP kube_clustertrustbundle_earliest_expiry_timestamp_seconds Unix timestamp of the earliest NotAfter across the parsable certificates in the clustertrustbundle.
		# TYPE kube_clustertrustbundle_earliest_expiry_timestamp_seconds gauge
		# HELP kube_clustertrustbundle_annotations Kubernetes annotations converted to Prometheus labels.
		# TYPE kube_clustertrustbundle_annotations gauge
		# HELP kube_clustertrustbundle_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_clustertrustbundle_labels gauge
	`
	allMetricNames := []string{
		"kube_clustertrustbundle_created",
		"kube_clustertrustbundle_info",
		"kube_clustertrustbundle_certificates",
		"kube_clustertrustbundle_earliest_expiry_timestamp_seconds",
		"kube_clustertrustbundle_annotations",
		"kube_clustertrustbundle_labels",
	}

	// The later certificate comes first so that the minimum is not simply
	// the first parsed block.
	laterCert := newTestCACertPEM(t, 1, time.Unix(1900000000, 0))
	earlierCert := newTestCACertPEM(t, 2, time.Unix(1800000000, 0))
	garbageCert := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("not a certificate")}))
	otherBlock := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("ignored")}))
	bundle := strings.Join([]string{laterCert, garbageCert, earlierCert, otherBlock}, "")

	cases := []generateMetricsTestCase{
		{
			// Full object: two parsable certificates, one unparsable
			// CERTIFICATE block and one non-certificate block.
			Obj: &certv1.ClusterTrustBundle{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "example.com:signer:abc",
					CreationTimestamp: metav1.Unix(1500000000, 0),
					Labels: map[string]string{
						"app": "trust",
					},
					Annotations: map[string]string{
						"team": "security",
					},
				},
				Spec: certv1.ClusterTrustBundleSpec{
					SignerName:  "example.com/signer",
					TrustBundle: bundle,
				},
			},
			Want: metadata + `
				kube_clustertrustbundle_created{clustertrustbundle="example.com:signer:abc"} 1.5e+09
				kube_clustertrustbundle_info{clustertrustbundle="example.com:signer:abc",signer_name="example.com/signer"} 1
				kube_clustertrustbundle_certificates{clustertrustbundle="example.com:signer:abc"} 3
				kube_clustertrustbundle_earliest_expiry_timestamp_seconds{clustertrustbundle="example.com:signer:abc"} 1.8e+09
			`,
			MetricNames: allMetricNames,
		},
		{
			// Minimal object: no signer, no timestamp, no certificates.
			Obj: &certv1.ClusterTrustBundle{
				ObjectMeta: metav1.ObjectMeta{
					Name: "minimal",
				},
			},
			Want: metadata + `
				kube_clustertrustbundle_info{clustertrustbundle="minimal",signer_name=""} 1
				kube_clustertrustbundle_certificates{clustertrustbundle="minimal"} 0
			`,
			MetricNames: allMetricNames,
		},
		{
			// Only unparsable certificates: the expiry metric is omitted.
			Obj: &certv1.ClusterTrustBundle{
				ObjectMeta: metav1.ObjectMeta{
					Name: "garbage",
				},
				Spec: certv1.ClusterTrustBundleSpec{
					TrustBundle: garbageCert + "trailing junk",
				},
			},
			Want: metadata + `
				kube_clustertrustbundle_info{clustertrustbundle="garbage",signer_name=""} 1
				kube_clustertrustbundle_certificates{clustertrustbundle="garbage"} 1
			`,
			MetricNames: allMetricNames,
		},
		{
			AllowAnnotationsList: []string{"team"},
			AllowLabelsList:      []string{"app"},
			Obj: &certv1.ClusterTrustBundle{
				ObjectMeta: metav1.ObjectMeta{
					Name: "with-metadata",
					Labels: map[string]string{
						"app":   "trust",
						"other": "dropped",
					},
					Annotations: map[string]string{
						"team":  "security",
						"other": "dropped",
					},
				},
			},
			Want: `
				# HELP kube_clustertrustbundle_annotations Kubernetes annotations converted to Prometheus labels.
				# TYPE kube_clustertrustbundle_annotations gauge
				# HELP kube_clustertrustbundle_labels Kubernetes labels converted to Prometheus labels.
				# TYPE kube_clustertrustbundle_labels gauge
				kube_clustertrustbundle_annotations{clustertrustbundle="with-metadata",annotation_team="security"} 1
				kube_clustertrustbundle_labels{clustertrustbundle="with-metadata",label_app="trust"} 1
			`,
			MetricNames: []string{
				"kube_clustertrustbundle_annotations",
				"kube_clustertrustbundle_labels",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(clusterTrustBundleMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(clusterTrustBundleMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
