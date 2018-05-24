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

package collectors

import (
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/collectors/testutils"
	"k8s.io/kube-state-metrics/pkg/options"
)

type mockSecretStore struct {
	f func() ([]v1.Secret, error)
}

func (ss mockSecretStore) List() (secrets []v1.Secret, err error) {
	return ss.f()
}

func TestSecretCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.

	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	const metadata = `
        # HELP kube_secret_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_secret_labels gauge
        # HELP kube_secret_info Information about secret.
		# TYPE kube_secret_info gauge
		# HELP kube_secret_type Type about secret.
		# TYPE kube_secret_type gauge
		# HELP kube_secret_created Unix creation timestamp
		# TYPE kube_secret_created gauge
		# HELP kube_secret_metadata_resource_version Resource version representing a specific version of secret.
		# TYPE kube_secret_metadata_resource_version gauge
	`
	cases := []struct {
		secrets []v1.Secret
		metrics []string
		want    string
	}{
		{
			secrets: []v1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "secret1",
						Namespace:       "ns1",
						ResourceVersion: "000000",
					},
					Type: v1.SecretTypeOpaque,
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "secret2",
						Namespace:         "ns2",
						CreationTimestamp: metav1StartTime,
						ResourceVersion:   "123456",
					},
					Type: v1.SecretTypeServiceAccountToken,
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "secret3",
						Namespace:         "ns3",
						CreationTimestamp: metav1StartTime,
						Labels:            map[string]string{"test-3": "test-3"},
						ResourceVersion:   "abcdef",
					},
					Type: v1.SecretTypeDockercfg,
				},
			},
			want: metadata + `
				kube_secret_info{secret="secret1",namespace="ns1"} 1
				kube_secret_info{secret="secret2",namespace="ns2"} 1
				kube_secret_info{secret="secret3",namespace="ns3"} 1
				kube_secret_type{secret="secret1",namespace="ns1",type="Opaque"} 1
				kube_secret_type{secret="secret2",namespace="ns2",type="kubernetes.io/service-account-token"} 1
				kube_secret_type{secret="secret3",namespace="ns3",type="kubernetes.io/dockercfg"} 1
				kube_secret_created{secret="secret2",namespace="ns2"} 1.501569018e+09
				kube_secret_created{secret="secret3",namespace="ns3"} 1.501569018e+09
				kube_secret_metadata_resource_version{secret="secret1",namespace="ns1",resource_version="000000"} 1
				kube_secret_metadata_resource_version{secret="secret2",namespace="ns2",resource_version="123456"} 1
				kube_secret_metadata_resource_version{secret="secret3",namespace="ns3",resource_version="abcdef"} 1
				kube_secret_labels{secret="secret3",namespace="ns3",label_test_3="test-3"} 1
				kube_secret_labels{secret="secret2",namespace="ns2"} 1
				kube_secret_labels{secret="secret1",namespace="ns1"} 1
				`,
			metrics: []string{"kube_secret_info", "kube_secret_metadata_resource_version", "kube_secret_created", "kube_secret_labels", "kube_secret_type"},
		},
	}
	for _, c := range cases {
		sc := &secretCollector{
			store: mockSecretStore{
				f: func() ([]v1.Secret, error) { return c.secrets, nil },
			},
			opts: &options.Options{},
		}
		if err := testutils.GatherAndCompare(sc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
