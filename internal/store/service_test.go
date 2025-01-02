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

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestServiceStore(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_service_annotations Kubernetes annotations converted to Prometheus labels.
		# TYPE kube_service_annotations gauge
		# HELP kube_service_info [STABLE] Information about service.
		# TYPE kube_service_info gauge
		# HELP kube_service_created [STABLE] Unix creation timestamp
		# TYPE kube_service_created gauge
		# HELP kube_service_labels [STABLE] Kubernetes labels converted to Prometheus labels.
		# TYPE kube_service_labels gauge
		# HELP kube_service_spec_type [STABLE] Type about service.
		# TYPE kube_service_spec_type gauge
		# HELP kube_service_spec_external_ip [STABLE] Service external ips. One series for each ip
		# TYPE kube_service_spec_external_ip gauge
		# HELP kube_service_status_load_balancer_ingress [STABLE] Service load balancer ingress status
		# TYPE kube_service_status_load_balancer_ingress gauge
	`
	cases := []generateMetricsTestCase{
		{
			Obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-service1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					UID:               "uid1",
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
				# HELP kube_service_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_service_created [STABLE] Unix creation timestamp
				# HELP kube_service_info [STABLE] Information about service.
				# HELP kube_service_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_service_spec_type [STABLE] Type about service.
				# TYPE kube_service_annotations gauge
				# TYPE kube_service_created gauge
				# TYPE kube_service_info gauge
				# TYPE kube_service_labels gauge
				# TYPE kube_service_spec_type gauge
				kube_service_created{namespace="default",service="test-service1",uid="uid1"} 1.5e+09
				kube_service_info{cluster_ip="1.2.3.4",external_name="",external_traffic_policy="",load_balancer_ip="",namespace="default",service="test-service1",uid="uid1"} 1
				kube_service_spec_type{namespace="default",service="test-service1",type="ClusterIP",uid="uid1"} 1
`,
			MetricNames: []string{
				"kube_service_annotations",
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
					UID:               "uid2",
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
				kube_service_created{namespace="default",service="test-service2",uid="uid2"} 1.5e+09
				kube_service_info{cluster_ip="1.2.3.5",external_name="",external_traffic_policy="",load_balancer_ip="",namespace="default",service="test-service2",uid="uid2"} 1
				kube_service_spec_type{namespace="default",service="test-service2",uid="uid2",type="NodePort"} 1
`,
		},
		{
			Obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-service3",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					UID:               "uid3",
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
				kube_service_created{namespace="default",service="test-service3",uid="uid3"} 1.5e+09
				kube_service_info{cluster_ip="1.2.3.6",external_name="",external_traffic_policy="",load_balancer_ip="1.2.3.7",namespace="default",service="test-service3",uid="uid3"} 1
				kube_service_spec_type{namespace="default",service="test-service3",type="LoadBalancer",uid="uid3"} 1
`,
		},
		{
			Obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-service4",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					UID:               "uid4",
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
				kube_service_created{namespace="default",service="test-service4",uid="uid4"} 1.5e+09
				kube_service_info{cluster_ip="",external_name="www.example.com",external_traffic_policy="",load_balancer_ip="",namespace="default",service="test-service4",uid="uid4"} 1
				kube_service_spec_type{namespace="default",service="test-service4",uid="uid4",type="ExternalName"} 1
			`,
		},
		{
			Obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-service5",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					UID:               "uid5",
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
				kube_service_created{namespace="default",service="test-service5",uid="uid5"} 1.5e+09
				kube_service_info{cluster_ip="",external_name="",external_traffic_policy="",load_balancer_ip="",namespace="default",service="test-service5",uid="uid5"} 1
				kube_service_spec_type{namespace="default",service="test-service5",type="LoadBalancer",uid="uid5"} 1
				kube_service_status_load_balancer_ingress{hostname="www.example.com",ip="1.2.3.8",namespace="default",service="test-service5",uid="uid5"} 1
			`,
		},
		{
			Obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-service6",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					UID:               "uid6",
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
				kube_service_created{namespace="default",service="test-service6",uid="uid6"} 1.5e+09
				kube_service_info{cluster_ip="",external_name="",external_traffic_policy="",load_balancer_ip="",namespace="default",service="test-service6",uid="uid6"} 1
				kube_service_spec_type{namespace="default",service="test-service6",uid="uid6",type="ClusterIP"} 1
				kube_service_spec_external_ip{external_ip="1.2.3.9",namespace="default",service="test-service6",uid="uid6"} 1
				kube_service_spec_external_ip{external_ip="1.2.3.10",namespace="default",service="test-service6",uid="uid6"} 1
			`,
		},
		{
			Obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-service7",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					UID:               "uid7",
					Labels: map[string]string{
						"app": "example7",
					},
				},
				Spec: v1.ServiceSpec{
					ClusterIP:             "1.2.3.11",
					Type:                  v1.ServiceTypeClusterIP,
					ExternalTrafficPolicy: "Cluster",
				},
			},
			Want: metadata + `
				kube_service_created{namespace="default",service="test-service7",uid="uid7"} 1.5e+09
				kube_service_info{cluster_ip="1.2.3.11",external_name="",external_traffic_policy="Cluster",load_balancer_ip="",namespace="default",service="test-service7",uid="uid7"} 1
				kube_service_spec_type{namespace="default",service="test-service7",uid="uid7",type="ClusterIP"} 1
			`,
		},
		{
			Obj: &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-service8",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					UID:               "uid8",
					Labels: map[string]string{
						"app": "example8",
					},
				},
				Spec: v1.ServiceSpec{
					ClusterIP:             "1.2.3.12",
					LoadBalancerIP:        "1.2.3.13",
					Type:                  v1.ServiceTypeLoadBalancer,
					ExternalTrafficPolicy: "Local",
				},
			},
			Want: metadata + `
				kube_service_created{namespace="default",service="test-service8",uid="uid8"} 1.5e+09
				kube_service_info{cluster_ip="1.2.3.12",external_name="",external_traffic_policy="Local",load_balancer_ip="1.2.3.13",namespace="default",service="test-service8",uid="uid8"} 1
				kube_service_spec_type{namespace="default",service="test-service8",uid="uid8",type="LoadBalancer"} 1
			`,
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(serviceMetricFamilies(nil, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(serviceMetricFamilies(nil, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
