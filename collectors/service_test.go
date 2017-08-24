/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

type mockServiceStore struct {
	list func() ([]v1.Service, error)
}

func (ss mockServiceStore) List() ([]v1.Service, error) {
	return ss.list()
}

func TestServiceCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_service_info Information about service.
		# TYPE kube_service_info gauge
		# HELP kube_service_created Unix creation timestamp
		# TYPE kube_service_created gauge
		# HELP kube_service_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_service_labels gauge
		# HELP kube_service_spec_type Type about service.
		# TYPE kube_service_spec_type gauge
	`
	cases := []struct {
		services []v1.Service
		metrics  []string // which metrics should be checked
		want     string
	}{
		{
			services: []v1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-service",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Namespace:         "default",
						Labels: map[string]string{
							"app": "example",
						},
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "10.233.0.2",
						Type:      v1.ServiceTypeClusterIP,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-service",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Namespace:         "default",
						Labels: map[string]string{
							"app": "example",
						},
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "10.233.0.3",
						Type:      v1.ServiceTypeNodePort,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-service",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Namespace:         "default",
						Labels: map[string]string{
							"app": "example",
						},
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "10.233.0.4",
						Type:      v1.ServiceTypeLoadBalancer,
					},
				},
			},
			want: metadata + `
				kube_service_info{namespace="default",service="test-service"} 1
				kube_service_created{namespace="default",service="test-service"} 1.5e+09
				kube_service_labels{label_app="example",namespace="default",service="test-service"} 1
				kube_service_spec_type{clusterIP="10.233.0.2",namespace="default",service="test-service",type="ClusterIP"} 1
				kube_service_info{namespace="default",service="test-service"} 1
				kube_service_created{namespace="default",service="test-service"} 1.5e+09
				kube_service_labels{label_app="example",namespace="default",service="test-service"} 1
				kube_service_spec_type{clusterIP="10.233.0.3",namespace="default",service="test-service",type="NodePort"} 1
				kube_service_info{namespace="default",service="test-service"} 1
				kube_service_created{namespace="default",service="test-service"} 1.5e+09
				kube_service_labels{label_app="example",namespace="default",service="test-service"} 1
				kube_service_spec_type{clusterIP="10.233.0.4",namespace="default",service="test-service",type="LoadBalancer"} 1
			`,
		},
	}
	for _, c := range cases {
		sc := &serviceCollector{
			store: &mockServiceStore{
				list: func() ([]v1.Service, error) {
					return c.services, nil
				},
			},
		}
		if err := gatherAndCompare(sc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
