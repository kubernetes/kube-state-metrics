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

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/collectors/testutils"
	"k8s.io/kube-state-metrics/pkg/options"
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
						Name:              "test-service1",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Namespace:         "default",
						Labels: map[string]string{
							"app": "example1",
						},
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "1.2.3.4",
						Type:      v1.ServiceTypeClusterIP,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-service2",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Namespace:         "default",
						Labels: map[string]string{
							"app": "example2",
						},
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "1.2.3.5",
						Type:      v1.ServiceTypeNodePort,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-service3",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Namespace:         "default",
						Labels: map[string]string{
							"app": "example3",
						},
					},
					Spec: v1.ServiceSpec{
						ClusterIP: "1.2.3.6",
						Type:      v1.ServiceTypeLoadBalancer,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-service4",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Namespace:         "default",
						Labels: map[string]string{
							"app": "example4",
						},
					},
					Spec: v1.ServiceSpec{
						Type: v1.ServiceTypeExternalName,
					},
				},
			},
			want: metadata + `	
				kube_service_created{namespace="default",service="test-service1"} 1.5e+09
				kube_service_created{namespace="default",service="test-service2"} 1.5e+09
				kube_service_created{namespace="default",service="test-service3"} 1.5e+09
				kube_service_created{namespace="default",service="test-service4"} 1.5e+09		
				kube_service_info{cluster_ip="",namespace="default",service="test-service4"} 1
				kube_service_info{cluster_ip="1.2.3.4",namespace="default",service="test-service1"} 1
				kube_service_info{cluster_ip="1.2.3.5",namespace="default",service="test-service2"} 1
				kube_service_info{cluster_ip="1.2.3.6",namespace="default",service="test-service3"} 1		
				kube_service_labels{label_app="example1",namespace="default",service="test-service1"} 1
				kube_service_labels{label_app="example2",namespace="default",service="test-service2"} 1
				kube_service_labels{label_app="example3",namespace="default",service="test-service3"} 1
				kube_service_labels{label_app="example4",namespace="default",service="test-service4"} 1
				kube_service_spec_type{namespace="default",service="test-service1",type="ClusterIP"} 1
				kube_service_spec_type{namespace="default",service="test-service2",type="NodePort"} 1
				kube_service_spec_type{namespace="default",service="test-service3",type="LoadBalancer"} 1
				kube_service_spec_type{namespace="default",service="test-service4",type="ExternalName"} 1
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
			opts: &options.Options{},
		}
		if err := testutils.GatherAndCompare(sc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
