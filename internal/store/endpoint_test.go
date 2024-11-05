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

func TestEndpointStore(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_endpoint_annotations Kubernetes annotations converted to Prometheus labels.
		# TYPE kube_endpoint_annotations gauge
		# HELP kube_endpoint_created [STABLE] Unix creation timestamp
		# TYPE kube_endpoint_created gauge
		# HELP kube_endpoint_info [STABLE] Information about endpoint.
		# TYPE kube_endpoint_info gauge
		# HELP kube_endpoint_labels [STABLE] Kubernetes labels converted to Prometheus labels.
		# TYPE kube_endpoint_labels gauge
		# HELP kube_endpoint_ports [STABLE] (Deprecated since v2.14.0) Information about the Endpoint ports.
		# TYPE kube_endpoint_ports gauge
		# HELP kube_endpoint_address [STABLE] Information about Endpoint available and non available addresses.
		# TYPE kube_endpoint_address gauge
	`
	cases := []generateMetricsTestCase{
		{
			Obj: &v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-endpoint",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					Labels: map[string]string{
						"app": "foobar",
					},
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{IP: "127.0.0.1"}, {IP: "10.0.0.1"},
						},
						NotReadyAddresses: []v1.EndpointAddress{
							{IP: "10.0.0.10"},
						},
						Ports: []v1.EndpointPort{
							{Port: 8080, Name: "http", Protocol: v1.ProtocolTCP}, {Port: 8081, Name: "app", Protocol: v1.ProtocolTCP},
						},
					},
					{
						Addresses: []v1.EndpointAddress{
							{IP: "172.22.23.202"},
						},
						Ports: []v1.EndpointPort{
							{Port: 8443, Name: "https", Protocol: v1.ProtocolTCP}, {Port: 9090, Name: "prometheus", Protocol: v1.ProtocolTCP},
						},
					},
					{
						NotReadyAddresses: []v1.EndpointAddress{
							{IP: "192.168.1.3"}, {IP: "192.168.2.2"},
						},
						Ports: []v1.EndpointPort{
							{Port: 1234, Name: "syslog", Protocol: v1.ProtocolUDP}, {Port: 5678, Name: "syslog-tcp", Protocol: v1.ProtocolTCP},
						},
					},
				},
			},
			Want: metadata + `
				kube_endpoint_created{endpoint="test-endpoint",namespace="default"} 1.5e+09
				kube_endpoint_info{endpoint="test-endpoint",namespace="default"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="10.0.0.1",namespace="default",port_name="app",port_number="8081",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="10.0.0.1",namespace="default",port_name="http",port_number="8080",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="10.0.0.10",namespace="default",port_name="app",port_number="8081",port_protocol="TCP",ready="false"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="10.0.0.10",namespace="default",port_name="http",port_number="8080",port_protocol="TCP",ready="false"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="127.0.0.1",namespace="default",port_name="app",port_number="8081",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="127.0.0.1",namespace="default",port_name="http",port_number="8080",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="172.22.23.202",namespace="default",port_name="https",port_number="8443",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="172.22.23.202",namespace="default",port_name="prometheus",port_number="9090",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="192.168.1.3",namespace="default",port_name="syslog",port_number="1234",port_protocol="UDP",ready="false"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="192.168.1.3",namespace="default",port_name="syslog-tcp",port_number="5678",port_protocol="TCP",ready="false"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="192.168.2.2",namespace="default",port_name="syslog",port_number="1234",port_protocol="UDP",ready="false"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="192.168.2.2",namespace="default",port_name="syslog-tcp",port_number="5678",port_protocol="TCP",ready="false"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="http",port_protocol="TCP",port_number="8080"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="app",port_protocol="TCP",port_number="8081"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="https",port_protocol="TCP",port_number="8443"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="prometheus",port_protocol="TCP",port_number="9090"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="syslog",port_protocol="UDP",port_number="1234"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="syslog-tcp",port_protocol="TCP",port_number="5678"} 1
			`,
		},
		{
			Obj: &v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "single-port-endpoint",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					Labels: map[string]string{
						"app": "single-foobar",
					},
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{IP: "127.0.0.1"}, {IP: "10.0.0.1"},
						},
						NotReadyAddresses: []v1.EndpointAddress{
							{IP: "10.0.0.10"},
						},
						Ports: []v1.EndpointPort{
							{Port: 8080, Protocol: v1.ProtocolTCP},
						},
					},
				},
			},
			Want: metadata + `
				kube_endpoint_created{endpoint="single-port-endpoint",namespace="default"} 1.5e+09
				kube_endpoint_info{endpoint="single-port-endpoint",namespace="default"} 1
				kube_endpoint_ports{endpoint="single-port-endpoint",namespace="default",port_name="",port_number="8080",port_protocol="TCP"} 1
                                kube_endpoint_address{endpoint="single-port-endpoint",ip="10.0.0.1",namespace="default",port_name="",port_number="8080",port_protocol="TCP",ready="true"} 1
                                kube_endpoint_address{endpoint="single-port-endpoint",ip="10.0.0.10",namespace="default",port_name="",port_number="8080",port_protocol="TCP",ready="false"} 1
                                kube_endpoint_address{endpoint="single-port-endpoint",ip="127.0.0.1",namespace="default",port_name="",port_number="8080",port_protocol="TCP",ready="true"} 1
			`,
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(endpointMetricFamilies(nil, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(endpointMetricFamilies(nil, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}

func TestEndpointStoreWithLabels(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_endpoint_annotations Kubernetes annotations converted to Prometheus labels.
		# TYPE kube_endpoint_annotations gauge
		# HELP kube_endpoint_created [STABLE] Unix creation timestamp
		# TYPE kube_endpoint_created gauge
		# HELP kube_endpoint_info [STABLE] Information about endpoint.
		# TYPE kube_endpoint_info gauge
		# HELP kube_endpoint_labels [STABLE] Kubernetes labels converted to Prometheus labels.
		# TYPE kube_endpoint_labels gauge
		# HELP kube_endpoint_ports [STABLE] (Deprecated since v2.14.0) Information about the Endpoint ports.
		# TYPE kube_endpoint_ports gauge
		# HELP kube_endpoint_address [STABLE] Information about Endpoint available and non available addresses.
		# TYPE kube_endpoint_address gauge
	`
	cases := []generateMetricsTestCase{
		{
			Obj: &v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-endpoint",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					Annotations: map[string]string{
						"app": "foobar",
					},
					Labels: map[string]string{
						"app": "foobar",
					},
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{IP: "127.0.0.1"}, {IP: "10.0.0.1"},
						},
						NotReadyAddresses: []v1.EndpointAddress{
							{IP: "10.0.0.10"},
						},
						Ports: []v1.EndpointPort{
							{Port: 8080, Name: "http", Protocol: v1.ProtocolTCP}, {Port: 8081, Name: "app", Protocol: v1.ProtocolTCP},
						},
					},
					{
						Addresses: []v1.EndpointAddress{
							{IP: "172.22.23.202"},
						},
						Ports: []v1.EndpointPort{
							{Port: 8443, Name: "https", Protocol: v1.ProtocolTCP}, {Port: 9090, Name: "prometheus", Protocol: v1.ProtocolTCP},
						},
					},
					{
						NotReadyAddresses: []v1.EndpointAddress{
							{IP: "192.168.1.3"}, {IP: "192.168.2.2"},
						},
						Ports: []v1.EndpointPort{
							{Port: 1234, Name: "syslog", Protocol: v1.ProtocolUDP}, {Port: 5678, Name: "syslog-tcp", Protocol: v1.ProtocolTCP},
						},
					},
				},
			},
			Want: metadata + `
				kube_endpoint_annotations{endpoint="test-endpoint",annotation_app="foobar",namespace="default"} 1
				kube_endpoint_created{endpoint="test-endpoint",namespace="default"} 1.5e+09
				kube_endpoint_info{endpoint="test-endpoint",namespace="default"} 1
				kube_endpoint_labels{endpoint="test-endpoint",label_app="foobar",namespace="default"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="http",port_protocol="TCP",port_number="8080"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="app",port_protocol="TCP",port_number="8081"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="https",port_protocol="TCP",port_number="8443"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="prometheus",port_protocol="TCP",port_number="9090"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="syslog",port_protocol="UDP",port_number="1234"} 1
				kube_endpoint_ports{endpoint="test-endpoint",namespace="default",port_name="syslog-tcp",port_protocol="TCP",port_number="5678"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="10.0.0.1",namespace="default",port_name="app",port_number="8081",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="10.0.0.1",namespace="default",port_name="http",port_number="8080",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="10.0.0.10",namespace="default",port_name="app",port_number="8081",port_protocol="TCP",ready="false"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="10.0.0.10",namespace="default",port_name="http",port_number="8080",port_protocol="TCP",ready="false"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="127.0.0.1",namespace="default",port_name="app",port_number="8081",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="127.0.0.1",namespace="default",port_name="http",port_number="8080",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="172.22.23.202",namespace="default",port_name="https",port_number="8443",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="172.22.23.202",namespace="default",port_name="prometheus",port_number="9090",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="192.168.1.3",namespace="default",port_name="syslog",port_number="1234",port_protocol="UDP",ready="false"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="192.168.1.3",namespace="default",port_name="syslog-tcp",port_number="5678",port_protocol="TCP",ready="false"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="192.168.2.2",namespace="default",port_name="syslog",port_number="1234",port_protocol="UDP",ready="false"} 1
				kube_endpoint_address{endpoint="test-endpoint",ip="192.168.2.2",namespace="default",port_name="syslog-tcp",port_number="5678",port_protocol="TCP",ready="false"} 1
			`,
		},
		{
			Obj: &v1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "single-port-endpoint",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					Annotations: map[string]string{
						"app": "single-foobar",
					},
					Labels: map[string]string{
						"app": "single-foobar",
					},
				},
				Subsets: []v1.EndpointSubset{
					{
						Addresses: []v1.EndpointAddress{
							{IP: "127.0.0.1"}, {IP: "10.0.0.1"},
						},
						NotReadyAddresses: []v1.EndpointAddress{
							{IP: "10.0.0.10"},
						},
						Ports: []v1.EndpointPort{
							{Port: 8080, Protocol: v1.ProtocolTCP},
						},
					},
				},
			},
			Want: metadata + `
				kube_endpoint_annotations{endpoint="single-port-endpoint",annotation_app="single-foobar",namespace="default"} 1
				kube_endpoint_created{endpoint="single-port-endpoint",namespace="default"} 1.5e+09
				kube_endpoint_info{endpoint="single-port-endpoint",namespace="default"} 1
				kube_endpoint_labels{endpoint="single-port-endpoint",label_app="single-foobar",namespace="default"} 1
				kube_endpoint_ports{endpoint="single-port-endpoint",namespace="default",port_name="",port_number="8080",port_protocol="TCP"} 1
				kube_endpoint_address{endpoint="single-port-endpoint",ip="10.0.0.1",namespace="default",port_name="",port_number="8080",port_protocol="TCP",ready="true"} 1
				kube_endpoint_address{endpoint="single-port-endpoint",ip="10.0.0.10",namespace="default",port_name="",port_number="8080",port_protocol="TCP",ready="false"} 1
				kube_endpoint_address{endpoint="single-port-endpoint",ip="127.0.0.1",namespace="default",port_name="",port_number="8080",port_protocol="TCP",ready="true"} 1
			`,
		},
	}
	for i, c := range cases {
		allowAnnotations := []string{
			"app",
		}
		allowLabels := []string{
			"app",
		}
		c.Func = generator.ComposeMetricGenFuncs(endpointMetricFamilies(allowAnnotations, allowLabels))
		c.Headers = generator.ExtractMetricFamilyHeaders(endpointMetricFamilies(allowAnnotations, allowLabels))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
