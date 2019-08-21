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

package store

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/pkg/metric"
)

func TestServiceStore(t *testing.T) {
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
		# HELP kube_service_spec_external_ip Service external ips. One series for each ip
		# TYPE kube_service_spec_external_ip gauge
		# HELP kube_service_status_load_balancer_ingress Service load balancer ingress status
		# TYPE kube_service_status_load_balancer_ingress gauge
	`
	cases := []generateMetricsTestCase{
		{
			Obj: &v1.Service{
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
			Want: `
				# HELP kube_service_created Unix creation timestamp
				# HELP kube_service_info Information about service.
				# HELP kube_service_labels Kubernetes labels converted to Prometheus labels.
				# HELP kube_service_spec_type Type about service.
				# TYPE kube_service_created gauge
				# TYPE kube_service_info gauge
				# TYPE kube_service_labels gauge
				# TYPE kube_service_spec_type gauge
				kube_service_created{namespace="default",service="test-service1"} 1.5e+09
				kube_service_info{cluster_ip="1.2.3.4",external_name="",load_balancer_ip="",namespace="default",service="test-service1"} 1
				kube_service_labels{label_app="example1",namespace="default",service="test-service1"} 1
				kube_service_spec_type{namespace="default",service="test-service1",type="ClusterIP"} 1
`,
			MetricNames: []string{
				"kube_service_created",
				"kube_service_info",
				"kube_service_labels",
				"kube_service_spec_type",
			},
		},
		{

			Obj: &v1.Service{
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
			Want: metadata + `
				kube_service_created{namespace="default",service="test-service2"} 1.5e+09
				kube_service_info{cluster_ip="1.2.3.5",external_name="",load_balancer_ip="",namespace="default",service="test-service2"} 1
				kube_service_labels{label_app="example2",namespace="default",service="test-service2"} 1
				kube_service_spec_type{namespace="default",service="test-service2",type="NodePort"} 1
`,
		},
		{
			Obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-service3",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					Labels: map[string]string{
						"app": "example3",
					},
				},
				Spec: v1.ServiceSpec{
					ClusterIP:      "1.2.3.6",
					LoadBalancerIP: "1.2.3.7",
					Type:           v1.ServiceTypeLoadBalancer,
				},
			},
			Want: metadata + `
				kube_service_created{namespace="default",service="test-service3"} 1.5e+09
				kube_service_info{cluster_ip="1.2.3.6",external_name="",load_balancer_ip="1.2.3.7",namespace="default",service="test-service3"} 1
				kube_service_labels{label_app="example3",namespace="default",service="test-service3"} 1
				kube_service_spec_type{namespace="default",service="test-service3",type="LoadBalancer"} 1
`,
		},
		{
			Obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-service4",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					Labels: map[string]string{
						"app": "example4",
					},
				},
				Spec: v1.ServiceSpec{
					ExternalName: "www.example.com",
					Type:         v1.ServiceTypeExternalName,
				},
			},
			Want: metadata + `
				kube_service_created{namespace="default",service="test-service4"} 1.5e+09
				kube_service_info{cluster_ip="",external_name="www.example.com",load_balancer_ip="",namespace="default",service="test-service4"} 1
				kube_service_labels{label_app="example4",namespace="default",service="test-service4"} 1
				kube_service_spec_type{namespace="default",service="test-service4",type="ExternalName"} 1
			`,
		},
		{
			Obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-service5",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					Labels: map[string]string{
						"app": "example5",
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeLoadBalancer,
				},
				Status: v1.ServiceStatus{
					LoadBalancer: v1.LoadBalancerStatus{
						Ingress: []v1.LoadBalancerIngress{
							{
								IP:       "1.2.3.8",
								Hostname: "www.example.com",
							},
						},
					},
				},
			},
			Want: metadata + `
				kube_service_created{namespace="default",service="test-service5"} 1.5e+09
				kube_service_info{cluster_ip="",external_name="",load_balancer_ip="",namespace="default",service="test-service5"} 1
				kube_service_labels{label_app="example5",namespace="default",service="test-service5"} 1
				kube_service_spec_type{namespace="default",service="test-service5",type="LoadBalancer"} 1
				kube_service_status_load_balancer_ingress{hostname="www.example.com",ip="1.2.3.8",namespace="default",service="test-service5"} 1
			`,
		},
		{
			Obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-service6",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					Labels: map[string]string{
						"app": "example6",
					},
				},
				Spec: v1.ServiceSpec{
					Type: v1.ServiceTypeClusterIP,
					ExternalIPs: []string{
						"1.2.3.9",
						"1.2.3.10",
					},
				},
			},
			Want: metadata + `
				kube_service_created{namespace="default",service="test-service6"} 1.5e+09
				kube_service_info{cluster_ip="",external_name="",load_balancer_ip="",namespace="default",service="test-service6"} 1
				kube_service_labels{label_app="example6",namespace="default",service="test-service6"} 1
				kube_service_spec_type{namespace="default",service="test-service6",type="ClusterIP"} 1
				kube_service_spec_external_ip{external_ip="1.2.3.9",namespace="default",service="test-service6"} 1
				kube_service_spec_external_ip{external_ip="1.2.3.10",namespace="default",service="test-service6"} 1
			`,
		},
	}
	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(serviceMetricFamilies)
		c.Headers = metric.ExtractMetricFamilyHeaders(serviceMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
