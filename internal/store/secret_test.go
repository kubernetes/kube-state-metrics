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
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestSecretStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)
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
				# HELP kube_secret_type [STABLE] Type about secret.
				# TYPE kube_secret_created gauge
				# TYPE kube_secret_info gauge
				# TYPE kube_secret_labels gauge
				# TYPE kube_secret_metadata_resource_version gauge
				# TYPE kube_secret_type gauge
				kube_secret_info{namespace="ns1",secret="secret1"} 1
				kube_secret_type{namespace="ns1",secret="secret1",type="Opaque"} 1
				kube_secret_metadata_resource_version{namespace="ns1",secret="secret1"} 0
				kube_secret_labels{namespace="ns1",secret="secret1"} 1
`,
			MetricNames: []string{"kube_secret_info", "kube_secret_metadata_resource_version", "kube_secret_created", "kube_secret_labels", "kube_secret_type"},
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
				# HELP kube_secret_type [STABLE] Type about secret.
				# TYPE kube_secret_created gauge
				# TYPE kube_secret_info gauge
				# TYPE kube_secret_labels gauge
				# TYPE kube_secret_metadata_resource_version gauge
				# TYPE kube_secret_type gauge
				kube_secret_info{namespace="ns2",secret="secret2"} 1
				kube_secret_type{namespace="ns2",secret="secret2",type="kubernetes.io/service-account-token"} 1
				kube_secret_created{namespace="ns2",secret="secret2"} 1.501569018e+09
				kube_secret_labels{namespace="ns2",secret="secret2"} 1
				`,
			MetricNames: []string{"kube_secret_info", "kube_secret_metadata_resource_version", "kube_secret_created", "kube_secret_labels", "kube_secret_type"},
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
				# HELP kube_secret_type [STABLE] Type about secret.
				# TYPE kube_secret_created gauge
				# TYPE kube_secret_info gauge
				# TYPE kube_secret_labels gauge
				# TYPE kube_secret_metadata_resource_version gauge
				# TYPE kube_secret_type gauge
				kube_secret_info{namespace="ns3",secret="secret3"} 1
				kube_secret_type{namespace="ns3",secret="secret3",type="kubernetes.io/dockercfg"} 1
				kube_secret_created{namespace="ns3",secret="secret3"} 1.501569018e+09
				kube_secret_metadata_resource_version{namespace="ns3",secret="secret3"} 0
				kube_secret_labels{namespace="ns3",secret="secret3"} 1
`,
			MetricNames: []string{"kube_secret_info", "kube_secret_metadata_resource_version", "kube_secret_created", "kube_secret_labels", "kube_secret_type"},
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
