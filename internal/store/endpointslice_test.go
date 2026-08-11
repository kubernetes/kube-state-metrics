/*
Copyright 2022 The Kubernetes Authors All rights reserved.
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
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestEndpointSliceStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)
	portname := "http"
	portnumber := int32(80)
	portprotocol := corev1.Protocol("TCP")
	nodename := "node"
	hostname := "host"
	zone := "west"
	ready := true
	serving := true
	terminating := false
	addresses := []string{"10.0.0.1", "192.168.1.10"}

	cases := []generateMetricsTestCase{
		{
			Obj: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_endpointslice-info",
					Namespace: "test",
				},
				AddressType: "IPv4",
			},
			Want: `
					# HELP kube_endpointslice_info Information about endpointslice.
					# TYPE kube_endpointslice_info gauge
					kube_endpointslice_info{endpointslice="test_endpointslice-info",addresstype="IPv4",namespace="test"} 1
				`,
			MetricNames: []string{
				"kube_endpointslice_info",
			},
		},
		{
			Obj: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test_kube_endpointslice-created",
					Namespace:         "test",
					CreationTimestamp: metav1StartTime,
				},
				AddressType: "IPv4",
			},
			Want: `
					# HELP kube_endpointslice_created Unix creation timestamp
					# TYPE kube_endpointslice_created gauge
					kube_endpointslice_created{endpointslice="test_kube_endpointslice-created",namespace="test"} 1.501569018e+09
				`,
			MetricNames: []string{
				"kube_endpointslice_created",
			},
		},
		{
			Obj: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_endpointslice-ports",
					Namespace: "test",
				},
				AddressType: "IPv4",
				Ports: []discoveryv1.EndpointPort{
					{Name: &portname,
						Port:     &portnumber,
						Protocol: &portprotocol,
					},
				},
			},
			Want: `
					# HELP kube_endpointslice_ports Ports attached to the endpointslice.
					# TYPE kube_endpointslice_ports gauge
					kube_endpointslice_ports{endpointslice="test_endpointslice-ports",port_name="http",port_protocol="TCP",port_number="80",namespace="test"} 1
				`,
			MetricNames: []string{
				"kube_endpointslice_ports",
			},
		},
		{
			// Port is optional and, unlike Name and Protocol, is not defaulted by
			// the API server: an EndpointSlice that is not used for routing
			// traffic may omit it.
			Obj: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_endpointslice-ports-without-number",
					Namespace: "test",
				},
				AddressType: "IPv4",
				Ports: []discoveryv1.EndpointPort{
					{Name: &portname,
						Protocol: &portprotocol,
					},
				},
			},
			Want: `
					# HELP kube_endpointslice_ports Ports attached to the endpointslice.
					# TYPE kube_endpointslice_ports gauge
					kube_endpointslice_ports{endpointslice="test_endpointslice-ports-without-number",port_name="http",port_protocol="TCP",port_number="",namespace="test"} 1
				`,
			MetricNames: []string{
				"kube_endpointslice_ports",
			},
		},
		{
			Obj: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_endpointslice-endpoints",
					Namespace: "test",
				},
				AddressType: "IPv4",
				Endpoints: []discoveryv1.Endpoint{
					{
						NodeName: &nodename,
						Conditions: discoveryv1.EndpointConditions{
							Ready:       &ready,
							Terminating: &terminating,
							Serving:     &serving,
						},
						Hostname:  &hostname,
						Zone:      &zone,
						Addresses: addresses,
					},
				},
			},
			Want: `
					# HELP kube_endpointslice_endpoints Endpoints attached to the endpointslice.
					# HELP kube_endpointslice_endpoints_hints Topology routing hints attached to endpoints
					# TYPE kube_endpointslice_endpoints gauge
					# TYPE kube_endpointslice_endpoints_hints gauge
					kube_endpointslice_endpoints{address="10.0.0.1",endpoint_nodename="node",endpoint_zone="west",endpointslice="test_endpointslice-endpoints",hostname="host",namespace="test",ready="true",serving="true",targetref_kind="",targetref_name="",targetref_namespace="",terminating="false"} 1
					kube_endpointslice_endpoints{address="192.168.1.10",endpoint_nodename="node",endpoint_zone="west",endpointslice="test_endpointslice-endpoints",hostname="host",namespace="test",ready="true",serving="true",targetref_kind="",targetref_name="",targetref_namespace="",terminating="false"} 1
				  `,

			MetricNames: []string{
				"kube_endpointslice_endpoints",
			},
		},
		{
			Obj: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_endpointslice-endpoints",
					Namespace: "test",
				},
				AddressType: "IPv4",
				Endpoints: []discoveryv1.Endpoint{
					{
						NodeName: &nodename,
						Conditions: discoveryv1.EndpointConditions{
							Ready:       &ready,
							Terminating: &terminating,
							Serving:     &serving,
						},
						Hostname:  &hostname,
						Zone:      &zone,
						Addresses: addresses,
						Hints: &discoveryv1.EndpointHints{
							ForZones: []discoveryv1.ForZone{
								{Name: "zone1"},
							},
						},
					},
				},
			},
			Want: `
					# HELP kube_endpointslice_endpoints Endpoints attached to the endpointslice.
					# HELP kube_endpointslice_endpoints_hints Topology routing hints attached to endpoints
					# TYPE kube_endpointslice_endpoints gauge
        			# TYPE kube_endpointslice_endpoints_hints gauge
         			kube_endpointslice_endpoints_hints{address="10.0.0.1",endpointslice="test_endpointslice-endpoints",for_zone="zone1",namespace="test"} 1
         			kube_endpointslice_endpoints{address="10.0.0.1",endpoint_nodename="node",endpoint_zone="west",endpointslice="test_endpointslice-endpoints",hostname="host",namespace="test",ready="true",serving="true",targetref_kind="",targetref_name="",targetref_namespace="",terminating="false"} 1
					kube_endpointslice_endpoints{address="192.168.1.10",endpoint_nodename="node",endpoint_zone="west",endpointslice="test_endpointslice-endpoints",hostname="host",namespace="test",ready="true",serving="true",targetref_kind="",targetref_name="",targetref_namespace="",terminating="false"} 1
				`,

			MetricNames: []string{
				"kube_endpointslice_endpoints",
			},
		},
		{
			AllowAnnotationsList: []string{
				"foo",
			},
			Obj: &discoveryv1.EndpointSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test_endpointslice-labels",
					Namespace: "test",
					Annotations: map[string]string{
						"foo": "baz",
					},
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				AddressType: "IPv4",
			},
			Want: `
					# HELP kube_endpointslice_annotations Kubernetes annotations converted to Prometheus labels.
					# HELP kube_endpointslice_labels Kubernetes labels converted to Prometheus labels.
					# TYPE kube_endpointslice_annotations gauge
					# TYPE kube_endpointslice_labels gauge
					kube_endpointslice_annotations{endpointslice="test_endpointslice-labels",annotation_foo="baz",namespace="test"} 1
				`,
			MetricNames: []string{
				"kube_endpointslice_annotations", "kube_endpointslice_labels",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(endpointSliceMetricFamilies(c.AllowAnnotationsList, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(endpointSliceMetricFamilies(c.AllowAnnotationsList, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}

// Each hint gets its own label slices. Building them by appending onto a slice
// shared across the loop works only while that slice has no spare capacity; with
// any slack every zone writes into the same backing array and they all report
// the last one.
func TestEndpointSliceHintsPerZone(t *testing.T) {
	es := &discoveryv1.EndpointSlice{
		ObjectMeta:  metav1.ObjectMeta{Name: "es", Namespace: "ns"},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{{
			Addresses: []string{"10.0.0.1"},
			Hints: &discoveryv1.EndpointHints{
				ForZones: []discoveryv1.ForZone{{Name: "zone-a"}, {Name: "zone-b"}, {Name: "zone-c"}},
			},
		}},
	}

	g := createEndpointsSliceHints()
	family := g.Generate(es)

	got := []string{}
	for _, m := range family.Metrics {
		if len(m.LabelKeys) != len(m.LabelValues) {
			t.Fatalf("label keys and values differ in length: %v vs %v", m.LabelKeys, m.LabelValues)
		}
		for i, k := range m.LabelKeys {
			if k == "for_zone" {
				got = append(got, m.LabelValues[i])
			}
		}
	}

	want := []string{"zone-a", "zone-b", "zone-c"}
	if !slices.Equal(got, want) {
		t.Errorf("for_zone values: got %v, want %v", got, want)
	}
}

// Validation requires at least one address today, so this is defence against a
// future relaxation rather than a reachable panic.
func TestEndpointSliceHintsWithoutAddresses(t *testing.T) {
	es := &discoveryv1.EndpointSlice{
		ObjectMeta:  metav1.ObjectMeta{Name: "es", Namespace: "ns"},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{{
			Hints: &discoveryv1.EndpointHints{ForZones: []discoveryv1.ForZone{{Name: "zone-a"}}},
		}},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on an endpoint with no addresses: %v", r)
		}
	}()

	g := createEndpointsSliceHints()
	if family := g.Generate(es); len(family.Metrics) != 0 {
		t.Errorf("expected the endpoint to be skipped, got %d metrics", len(family.Metrics))
	}
}
