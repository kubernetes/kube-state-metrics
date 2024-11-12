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

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestNetworkPolicyStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	cases := []generateMetricsTestCase{
		{
			Obj: &networkingv1.NetworkPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "netpol1",
					Namespace:         "ns1",
					CreationTimestamp: metav1StartTime,
				},
				Spec: networkingv1.NetworkPolicySpec{
					Ingress: []networkingv1.NetworkPolicyIngressRule{
						{},
						{},
					},
					Egress: []networkingv1.NetworkPolicyEgressRule{
						{},
						{},
						{},
					},
				},
			},
			Want: `
			kube_networkpolicy_created{namespace="ns1",networkpolicy="netpol1"} 1.501569018e+09
			kube_networkpolicy_spec_egress_rules{namespace="ns1",networkpolicy="netpol1"} 3
			kube_networkpolicy_spec_ingress_rules{namespace="ns1",networkpolicy="netpol1"} 2
			`,
			MetricNames: []string{
				"kube_networkpolicy_created",
				"kube_networkpolicy_labels",
				"kube_networkpolicy_spec_egress_rules",
				"kube_networkpolicy_spec_ingress_rules",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(networkPolicyMetricFamilies(nil, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %dth run:\n%s", i, err)
		}
	}
}
