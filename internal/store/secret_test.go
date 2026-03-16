/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"math/rand"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestSecretStore(t *testing.T) {
	var test = true
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)
	tlsCertExpiration := time.Unix(1893456000, 0)
	tlsCertPEM, err := generateTestTLSCertPEM(tlsCertExpiration)
	if err != nil {
		t.Fatalf("failed to generate tls cert: %v", err)
	}
	cases := []generateMetricsTestCase{
		{
			Obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "secret1",
					Namespace:       "ns1",
					ResourceVersion: "000000",
				},
				Type: v1.SecretTypeOpaque,
			},
			Want: `
				# HELP kube_secret_created [STABLE] Unix creation timestamp
				# HELP kube_secret_info [STABLE] Information about secret.
				# HELP kube_secret_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_secret_metadata_resource_version Resource version representing a specific version of secret.
				# HELP kube_secret_owner Information about the Secret's owner.
				# HELP kube_secret_tls_cert_not_after_seconds Unix notAfter timestamp of the TLS certificate in the secret.
				# HELP kube_secret_type [STABLE] Type about secret.
				# TYPE kube_secret_created gauge
				# TYPE kube_secret_info gauge
				# TYPE kube_secret_labels gauge
				# TYPE kube_secret_metadata_resource_version gauge
				# TYPE kube_secret_owner gauge
				# TYPE kube_secret_tls_cert_not_after_seconds gauge
				# TYPE kube_secret_type gauge
				kube_secret_info{namespace="ns1",secret="secret1"} 1
				kube_secret_owner{namespace="ns1",owner_is_controller="",owner_kind="",owner_name="",secret="secret1"} 1
				kube_secret_type{namespace="ns1",secret="secret1",type="Opaque"} 1
				kube_secret_metadata_resource_version{namespace="ns1",secret="secret1"} 0
`,
			MetricNames: []string{"kube_secret_info", "kube_secret_metadata_resource_version", "kube_secret_created", "kube_secret_labels", "kube_secret_type", "kube_secret_owner", "kube_secret_tls_cert_not_after_seconds"},
		},
		{
			Obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "secret2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "123456B",
				},
				Type: v1.SecretTypeServiceAccountToken,
			},
			Want: `
				# HELP kube_secret_created [STABLE] Unix creation timestamp
				# HELP kube_secret_info [STABLE] Information about secret.
				# HELP kube_secret_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_secret_metadata_resource_version Resource version representing a specific version of secret.
				# HELP kube_secret_owner Information about the Secret's owner.
				# HELP kube_secret_tls_cert_not_after_seconds Unix notAfter timestamp of the TLS certificate in the secret.
				# HELP kube_secret_type [STABLE] Type about secret.
				# TYPE kube_secret_created gauge
				# TYPE kube_secret_info gauge
				# TYPE kube_secret_labels gauge
				# TYPE kube_secret_metadata_resource_version gauge
				# TYPE kube_secret_owner gauge
				# TYPE kube_secret_tls_cert_not_after_seconds gauge
				# TYPE kube_secret_type gauge
				kube_secret_info{namespace="ns2",secret="secret2"} 1
				kube_secret_owner{namespace="ns2",owner_is_controller="",owner_kind="",owner_name="",secret="secret2"} 1
				kube_secret_type{namespace="ns2",secret="secret2",type="kubernetes.io/service-account-token"} 1
				kube_secret_created{namespace="ns2",secret="secret2"} 1.501569018e+09
				`,
			MetricNames: []string{"kube_secret_info", "kube_secret_metadata_resource_version", "kube_secret_created", "kube_secret_labels", "kube_secret_type", "kube_secret_owner", "kube_secret_tls_cert_not_after_seconds"},
		},
		{
			Obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "secret3",
					Namespace:         "ns3",
					CreationTimestamp: metav1StartTime,
					Labels:            map[string]string{"test-3": "test-3"},
					ResourceVersion:   "0",
				},
				Type: v1.SecretTypeDockercfg,
			},
			Want: `
				# HELP kube_secret_created [STABLE] Unix creation timestamp
				# HELP kube_secret_info [STABLE] Information about secret.
				# HELP kube_secret_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_secret_metadata_resource_version Resource version representing a specific version of secret.
				# HELP kube_secret_owner Information about the Secret's owner.
				# HELP kube_secret_tls_cert_not_after_seconds Unix notAfter timestamp of the TLS certificate in the secret.
				# HELP kube_secret_type [STABLE] Type about secret.
				# TYPE kube_secret_created gauge
				# TYPE kube_secret_info gauge
				# TYPE kube_secret_labels gauge
				# TYPE kube_secret_metadata_resource_version gauge
				# TYPE kube_secret_owner gauge
				# TYPE kube_secret_tls_cert_not_after_seconds gauge
				# TYPE kube_secret_type gauge
				kube_secret_info{namespace="ns3",secret="secret3"} 1
				kube_secret_owner{namespace="ns3",owner_is_controller="",owner_kind="",owner_name="",secret="secret3"} 1
				kube_secret_type{namespace="ns3",secret="secret3",type="kubernetes.io/dockercfg"} 1
				kube_secret_created{namespace="ns3",secret="secret3"} 1.501569018e+09
				kube_secret_metadata_resource_version{namespace="ns3",secret="secret3"} 0
`,
			MetricNames: []string{"kube_secret_info", "kube_secret_metadata_resource_version", "kube_secret_created", "kube_secret_labels", "kube_secret_type", "kube_secret_owner", "kube_secret_tls_cert_not_after_seconds"},
		},
		{
			Obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "secret4",
					Namespace:         "ns4",
					CreationTimestamp: metav1StartTime,
					Labels:            map[string]string{"test-4": "test-4"},
					ResourceVersion:   "0",
					OwnerReferences: []metav1.OwnerReference{
						{
							Name:       "managed-secret4",
							Kind:       "ManagedSecret",
							Controller: &test,
						},
					},
				},
				Type: v1.SecretTypeOpaque,
			},
			Want: `
				# HELP kube_secret_created [STABLE] Unix creation timestamp
				# HELP kube_secret_info [STABLE] Information about secret.
				# HELP kube_secret_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_secret_metadata_resource_version Resource version representing a specific version of secret.
				# HELP kube_secret_owner Information about the Secret's owner.
				# HELP kube_secret_tls_cert_not_after_seconds Unix notAfter timestamp of the TLS certificate in the secret.
				# HELP kube_secret_type [STABLE] Type about secret.
				# TYPE kube_secret_created gauge
				# TYPE kube_secret_info gauge
				# TYPE kube_secret_labels gauge
				# TYPE kube_secret_metadata_resource_version gauge
				# TYPE kube_secret_owner gauge
				# TYPE kube_secret_tls_cert_not_after_seconds gauge
				# TYPE kube_secret_type gauge
				kube_secret_info{namespace="ns4",secret="secret4"} 1
				kube_secret_owner{namespace="ns4",owner_is_controller="true",owner_kind="ManagedSecret",owner_name="managed-secret4",secret="secret4"} 1
				kube_secret_type{namespace="ns4",secret="secret4",type="Opaque"} 1
				kube_secret_created{namespace="ns4",secret="secret4"} 1.501569018e+09
				kube_secret_metadata_resource_version{namespace="ns4",secret="secret4"} 0
`,
			MetricNames: []string{"kube_secret_info", "kube_secret_metadata_resource_version", "kube_secret_created", "kube_secret_labels", "kube_secret_type", "kube_secret_owner", "kube_secret_tls_cert_not_after_seconds"},
		},
		{
			Obj: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "secret5",
					Namespace:       "ns5",
					ResourceVersion: "1",
				},
				Type: v1.SecretTypeTLS,
				Data: map[string][]byte{
					"tls.crt": tlsCertPEM,
				},
			},
			Want: `
				# HELP kube_secret_created [STABLE] Unix creation timestamp
				# HELP kube_secret_info [STABLE] Information about secret.
				# HELP kube_secret_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_secret_metadata_resource_version Resource version representing a specific version of secret.
				# HELP kube_secret_owner Information about the Secret's owner.
				# HELP kube_secret_tls_cert_not_after_seconds Unix notAfter timestamp of the TLS certificate in the secret.
				# HELP kube_secret_type [STABLE] Type about secret.
				# TYPE kube_secret_created gauge
				# TYPE kube_secret_info gauge
				# TYPE kube_secret_labels gauge
				# TYPE kube_secret_metadata_resource_version gauge
				# TYPE kube_secret_owner gauge
				# TYPE kube_secret_tls_cert_not_after_seconds gauge
				# TYPE kube_secret_type gauge
				kube_secret_info{namespace="ns5",secret="secret5"} 1
				kube_secret_owner{namespace="ns5",owner_is_controller="",owner_kind="",owner_name="",secret="secret5"} 1
				kube_secret_type{namespace="ns5",secret="secret5",type="kubernetes.io/tls"} 1
				kube_secret_metadata_resource_version{namespace="ns5",secret="secret5"} 1
				kube_secret_tls_cert_not_after_seconds{cn="test",namespace="ns5",sans="test,test2",secret="secret5",serial="1"} 1.893456e+09
`,
			MetricNames: []string{"kube_secret_info", "kube_secret_metadata_resource_version", "kube_secret_created", "kube_secret_labels", "kube_secret_type", "kube_secret_owner", "kube_secret_tls_cert_not_after_seconds"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(secretMetricFamilies(nil, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(secretMetricFamilies(nil, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}

	}
}

type deterministicReader struct {
	random *rand.Rand
}

func (r *deterministicReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(r.random.Intn(256))
	}
	return len(p), nil
}

func generateTestTLSCertPEM(notAfter time.Time) ([]byte, error) {
	reader := &deterministicReader{random: rand.New(rand.NewSource(1))} //nolint:gosec
	key, err := rsa.GenerateKey(reader, 2048)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		Subject: pkix.Name{
			CommonName: "test",
		},
		DNSNames:     []string{"test", "test2"},
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Unix(0, 0),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	derBytes, err := x509.CreateCertificate(reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes}), nil
}
