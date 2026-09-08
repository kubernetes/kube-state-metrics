/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestResourceClaimStore(t *testing.T) {
	const metadata = `
		# HELP kube_resourceclaim_created Unix creation timestamp
		# TYPE kube_resourceclaim_created gauge
		# HELP kube_resourceclaim_info Information about resource claim.
		# TYPE kube_resourceclaim_info gauge
		# HELP kube_resourceclaim_status_allocated Indicates whether the resource claim has been allocated.
		# TYPE kube_resourceclaim_status_allocated gauge
		# HELP kube_resourceclaim_status_reserved_for Indicates which consumers have currently reserved the resource claim.
		# TYPE kube_resourceclaim_status_reserved_for gauge
		# HELP kube_resourceclaim_allocation_device_info Allocation information about the devices allocated to the resource claim.
		# TYPE kube_resourceclaim_allocation_device_info gauge
	`

	cases := []generateMetricsTestCase{
		{
			Obj: &resourcev1beta1.ResourceClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "claim-1",
					Namespace:         "ns-1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Spec: resourcev1beta1.ResourceClaimSpec{},
				Status: resourcev1beta1.ResourceClaimStatus{
					Allocation: &resourcev1beta1.AllocationResult{
						Devices: resourcev1beta1.DeviceAllocationResult{
							Results: []resourcev1beta1.DeviceRequestAllocationResult{
								{
									Request: "req-1",
									Driver:  "driver-1",
									Pool:    "pool-1",
									Device:  "device-1",
								},
							},
						},
					},
					ReservedFor: []resourcev1beta1.ResourceClaimConsumerReference{
						{
							APIGroup: "apps",
							Resource: "deployments",
							Name:     "dep-1",
							UID:      types.UID("uid-1"),
						},
					},
				},
			},
			Want: metadata + `
				kube_resourceclaim_created{namespace="ns-1",resourceclaim="claim-1"} 1.5e+09
				kube_resourceclaim_info{namespace="ns-1",resourceclaim="claim-1"} 1
				kube_resourceclaim_status_allocated{namespace="ns-1",resourceclaim="claim-1"} 1
				kube_resourceclaim_status_reserved_for{consumer_apigroup="apps",consumer_name="dep-1",consumer_resource="deployments",consumer_uid="uid-1",namespace="ns-1",resourceclaim="claim-1"} 1
				kube_resourceclaim_allocation_device_info{driver="driver-1",device="device-1",namespace="ns-1",pool="pool-1",request="req-1",resourceclaim="claim-1"} 1
			`,
			MetricNames: []string{
				"kube_resourceclaim_created",
				"kube_resourceclaim_info",
				"kube_resourceclaim_status_allocated",
				"kube_resourceclaim_status_reserved_for",
				"kube_resourceclaim_allocation_device_info",
			},
		},
	}

	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(resourceClaimMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(resourceClaimMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %dth run:\n%v", i, err)
		}
	}
}
