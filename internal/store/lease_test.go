/*
Copyright 2020 The Kubernetes Authors All rights reserved.

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

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/v2/pkg/allow"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestLeaseStore(t *testing.T) {
	const metadata = `
		# HELP kube_lease_annotations Kubernetes annotations converted to Prometheus labels.
		# TYPE kube_lease_annotations gauge
        # HELP kube_lease_owner Information about the Lease's owner.
        # TYPE kube_lease_owner gauge
        # HELP kube_lease_renew_time Kube lease renew time.
        # TYPE kube_lease_renew_time gauge
	`

	var (
		cases = []generateMetricsTestCase{
			{
				Obj: &coordinationv1.Lease{
					ObjectMeta: metav1.ObjectMeta{
						Generation:        2,
						Name:              "kube-master",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						OwnerReferences: []metav1.OwnerReference{
							{
								Kind: "Node",
								Name: "kube-master",
							},
						},
						Annotations: map[string]string{
							"whitelisted":     "true",
							"not-whitelisted": "false",
						},
					},
					Spec: coordinationv1.LeaseSpec{
						RenewTime: &metav1.MicroTime{Time: time.Unix(1500000000, 0)},
					},
				},
				Want: metadata + `
					kube_lease_annotations{annotation_whitelisted="true",lease="kube-master"} 1
                    kube_lease_owner{lease="kube-master",owner_kind="Node",owner_name="kube-master"} 1
                    kube_lease_renew_time{lease="kube-master"} 1.5e+09
			`,
				MetricNames: []string{
					"kube_lease_owner",
					"kube_lease_renew_time",
					"kube_lease_annotations",
				},
				allowLabels: allow.Labels{"kube_lease_annotations": append([]string{"annotation_whitelisted"}, descLeaseLabelsDefaultLabels...)},
			},
		}
	)
	for i, c := range cases {
		filteredWhitelistedAnnotationMetricFamilies := generator.FilterMetricFamiliesLabels(c.allowLabels, leaseMetricFamilies)
		c.Func = generator.ComposeMetricGenFuncs(filteredWhitelistedAnnotationMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(filteredWhitelistedAnnotationMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %dth run:\n%v", i, err)
		}
	}
}
