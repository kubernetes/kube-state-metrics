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

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/kube-state-metrics/pkg/metric"
)

func TestIngressStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_ingress_created Unix creation timestamp
		# HELP kube_ingress_info Information about ingress.
		# HELP kube_ingress_labels Kubernetes labels converted to Prometheus labels.
		# HELP kube_ingress_metadata_resource_version Resource version representing a specific version of ingress.
		# HELP kube_ingress_path Ingress host, paths and backend service information.
		# HELP kube_ingress_tls Ingress TLS host and secret information.
		# TYPE kube_ingress_created gauge
		# TYPE kube_ingress_info gauge
		# TYPE kube_ingress_labels gauge
		# TYPE kube_ingress_metadata_resource_version gauge
		# TYPE kube_ingress_path gauge
		# TYPE kube_ingress_tls gauge
	`
	cases := []generateMetricsTestCase{
		{
			Obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "ingress1",
					Namespace:       "ns1",
					ResourceVersion: "000000",
				},
			},
			Want: metadata + `
				kube_ingress_info{namespace="ns1",ingress="ingress1"} 1
				kube_ingress_metadata_resource_version{namespace="ns1",resource_version="000000",ingress="ingress1"} 1
				kube_ingress_labels{namespace="ns1",ingress="ingress1"} 1
`,
			MetricNames: []string{"kube_ingress_info", "kube_ingress_metadata_resource_version", "kube_ingress_created", "kube_ingress_labels", "kube_ingress_path", "kube_ingress_tls"},
		},
		{
			Obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "ingress2",
					Namespace:         "ns2",
					CreationTimestamp: metav1StartTime,
					ResourceVersion:   "123456",
				},
			},
			Want: metadata + `
				kube_ingress_info{namespace="ns2",ingress="ingress2"} 1
				kube_ingress_created{namespace="ns2",ingress="ingress2"} 1.501569018e+09
				kube_ingress_metadata_resource_version{namespace="ns2",resource_version="123456",ingress="ingress2"} 1
				kube_ingress_labels{namespace="ns2",ingress="ingress2"} 1
				`,
			MetricNames: []string{"kube_ingress_info", "kube_ingress_metadata_resource_version", "kube_ingress_created", "kube_ingress_labels", "kube_ingress_path", "kube_ingress_tls"},
		},
		{
			Obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "ingress3",
					Namespace:         "ns3",
					CreationTimestamp: metav1StartTime,
					Labels:            map[string]string{"test-3": "test-3"},
					ResourceVersion:   "abcdef",
				},
			},
			Want: metadata + `
				kube_ingress_info{namespace="ns3",ingress="ingress3"} 1
				kube_ingress_created{namespace="ns3",ingress="ingress3"} 1.501569018e+09
				kube_ingress_metadata_resource_version{namespace="ns3",resource_version="abcdef",ingress="ingress3"} 1
				kube_ingress_labels{label_test_3="test-3",namespace="ns3",ingress="ingress3"} 1
`,
			MetricNames: []string{"kube_ingress_info", "kube_ingress_metadata_resource_version", "kube_ingress_created", "kube_ingress_labels", "kube_ingress_path", "kube_ingress_tls"},
		},
		{
			Obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "ingress4",
					Namespace:         "ns4",
					CreationTimestamp: metav1StartTime,
					Labels:            map[string]string{"test-4": "test-4"},
					ResourceVersion:   "abcdef",
				},
				Spec: v1beta1.IngressSpec{
					Rules: []v1beta1.IngressRule{
						{
							Host: "somehost",
							IngressRuleValue: v1beta1.IngressRuleValue{
								HTTP: &v1beta1.HTTPIngressRuleValue{
									Paths: []v1beta1.HTTPIngressPath{
										{
											Path: "/somepath",
											Backend: v1beta1.IngressBackend{
												ServiceName: "someservice",
												ServicePort: intstr.FromInt(1234),
											},
										},
									},
								},
							},
						},
						{
							Host: "somehost2",
						},
					},
				},
			},
			Want: metadata + `
				kube_ingress_info{namespace="ns4",ingress="ingress4"} 1
				kube_ingress_created{namespace="ns4",ingress="ingress4"} 1.501569018e+09
				kube_ingress_metadata_resource_version{namespace="ns4",resource_version="abcdef",ingress="ingress4"} 1
				kube_ingress_labels{label_test_4="test-4",namespace="ns4",ingress="ingress4"} 1
				kube_ingress_path{namespace="ns4",ingress="ingress4",host="somehost",path="/somepath",service_name="someservice",service_port="1234"} 1
`,
			MetricNames: []string{"kube_ingress_info", "kube_ingress_metadata_resource_version", "kube_ingress_created", "kube_ingress_labels", "kube_ingress_path", "kube_ingress_tls"},
		},
		{
			Obj: &v1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "ingress5",
					Namespace:         "ns5",
					CreationTimestamp: metav1StartTime,
					Labels:            map[string]string{"test-5": "test-5"},
					ResourceVersion:   "abcdef",
				},
				Spec: v1beta1.IngressSpec{
					TLS: []v1beta1.IngressTLS{
						{
							Hosts:      []string{"somehost1", "somehost2"},
							SecretName: "somesecret",
						},
					},
				},
			},
			Want: metadata + `
				kube_ingress_info{namespace="ns5",ingress="ingress5"} 1
				kube_ingress_created{namespace="ns5",ingress="ingress5"} 1.501569018e+09
				kube_ingress_metadata_resource_version{namespace="ns5",resource_version="abcdef",ingress="ingress5"} 1
				kube_ingress_labels{label_test_5="test-5",namespace="ns5",ingress="ingress5"} 1
				kube_ingress_tls{namespace="ns5",ingress="ingress5",tls_host="somehost1",secret="somesecret"} 1
				kube_ingress_tls{namespace="ns5",ingress="ingress5",tls_host="somehost2",secret="somesecret"} 1
`,
			MetricNames: []string{"kube_ingress_info", "kube_ingress_metadata_resource_version", "kube_ingress_created", "kube_ingress_labels", "kube_ingress_path", "kube_ingress_tls"},
		},
	}
	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(ingressMetricFamilies)
		c.Headers = metric.ExtractMetricFamilyHeaders(ingressMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}

	}
}
