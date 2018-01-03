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

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockIngressStore struct {
	list func() (v1beta1.IngressList, error)
}

func (ig mockIngressStore) List() (ingresses v1beta1.IngressList, err error) {
	return ig.list()
}

func TestIngressCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_ingress_info The info of ingress.
		# TYPE kube_ingress_info gauge
		# HELP kube_ingress_metadata_generation Sequence number representing a specific generation of the desired state.
		# TYPE kube_ingress_metadata_generation gauge
		# HELP kube_ingress_loadbalancer kube ingress loadbalancer.
		# TYPE kube_ingress_loadbalancer gauge
	`
	cases := []struct {
		igs  []v1beta1.Ingress
		want string
	}{
		{
			igs: []v1beta1.Ingress{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "igs1",
						Namespace:  "igs1",
						Generation: 21,
					},
					Status: v1beta1.IngressStatus{
						LoadBalancer: v1.LoadBalancerStatus{
							Ingress: []v1.LoadBalancerIngress{
								v1.LoadBalancerIngress{
									IP: "10.233.0.4",
								},
							},
						},
					},
				},
			},
			want: metadata + `
				kube_ingress_info{ingress="igs1",namespace="igs1"} 1
				kube_ingress_metadata_generation{ingress="igs1",namespace="igs1"} 21
				kube_ingress_loadbalancer{hostname="",ingress="igs1",ip="10.233.0.4",namespace="igs1"} 1
			`,
		},
	}
	for _, c := range cases {
		dc := &ingressCollector{
			store: mockIngressStore{
				list: func() (v1beta1.IngressList, error) { return v1beta1.IngressList{Items: c.igs}, nil },
			},
		}
		if err := gatherAndCompare(dc, c.want, nil); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
