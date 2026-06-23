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

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestResourceSliceStore(t *testing.T) {
	const metadata = `
		# HELP kube_resourceslice_created Unix creation timestamp
		# TYPE kube_resourceslice_created gauge
		# HELP kube_resourceslice_info Information about resource slice.
		# TYPE kube_resourceslice_info gauge
		# HELP kube_resourceslice_devices_total The total count of devices published by this resource slice.
		# TYPE kube_resourceslice_devices_total gauge
		# HELP kube_resourceslice_device_info Details of individual devices inside the resource slice.
		# TYPE kube_resourceslice_device_info gauge
	`

	cases := []generateMetricsTestCase{
		{
			Obj: &resourcev1beta1.ResourceSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "slice-1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Spec: resourcev1beta1.ResourceSliceSpec{
					Driver:   "driver-1",
					NodeName: "node-1",
					AllNodes: false,
					Pool: resourcev1beta1.ResourcePool{
						Name: "pool-1",
					},
					Devices: []resourcev1beta1.Device{
						{
							Name: "device-1",
						},
						{
							Name: "device-2",
						},
					},
				},
			},
			Want: metadata + `
				kube_resourceslice_created{resourceslice="slice-1"} 1.5e+09
				kube_resourceslice_info{all_nodes="false",driver="driver-1",node_name="node-1",pool_name="pool-1",resourceslice="slice-1"} 1
				kube_resourceslice_devices_total{driver="driver-1",node_name="node-1",pool_name="pool-1",resourceslice="slice-1"} 2
				kube_resourceslice_device_info{device_name="device-1",driver="driver-1",node_name="node-1",pool_name="pool-1",resourceslice="slice-1"} 1
				kube_resourceslice_device_info{device_name="device-2",driver="driver-1",node_name="node-1",pool_name="pool-1",resourceslice="slice-1"} 1
			`,
			MetricNames: []string{
				"kube_resourceslice_created",
				"kube_resourceslice_info",
				"kube_resourceslice_devices_total",
				"kube_resourceslice_device_info",
			},
		},
	}

	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(resourceSliceMetricFamilies)
		c.Headers = generator.ExtractMetricFamilyHeaders(resourceSliceMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %dth run:\n%v", i, err)
		}
	}
}
