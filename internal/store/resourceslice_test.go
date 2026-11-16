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

	v1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestResourceSliceStore(t *testing.T) {
	const metadata = `
		# HELP kube_resourceslice_created Unix creation timestamp
		# TYPE kube_resourceslice_created gauge
		# HELP kube_resourceslice_info Information about resource slice.
		# TYPE kube_resourceslice_info gauge
		# HELP kube_resourceslice_devices The total count of devices published by this resource slice.
		# TYPE kube_resourceslice_devices gauge
	`

	cases := []generateMetricsTestCase{
		{
			Obj: &resourcev1.ResourceSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "slice-1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Spec: resourcev1.ResourceSliceSpec{
					Driver:   "driver-1",
					NodeName: ptr.To("node-1"),
					Pool: resourcev1.ResourcePool{
						Name: "pool-1",
					},
					Devices: []resourcev1.Device{
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
				kube_resourceslice_info{driver="driver-1",node="node-1",node_selection="node_name",pool="pool-1",resourceslice="slice-1"} 1
				kube_resourceslice_devices{driver="driver-1",node="node-1",pool="pool-1",resourceslice="slice-1"} 2
			`,
			MetricNames: []string{
				"kube_resourceslice_created",
				"kube_resourceslice_info",
				"kube_resourceslice_devices",
			},
		},
		{
			Obj: &resourcev1.ResourceSlice{
				ObjectMeta: metav1.ObjectMeta{Name: "slice-selector"},
				Spec: resourcev1.ResourceSliceSpec{
					Driver:       "driver-1",
					NodeSelector: &v1.NodeSelector{},
					Pool:         resourcev1.ResourcePool{Name: "pool-1"},
				},
			},
			Want: `
				# HELP kube_resourceslice_info Information about resource slice.
				# TYPE kube_resourceslice_info gauge
				kube_resourceslice_info{driver="driver-1",node="",node_selection="node_selector",pool="pool-1",resourceslice="slice-selector"} 1
			`,
			MetricNames: []string{"kube_resourceslice_info"},
		},
		{
			Obj: &resourcev1.ResourceSlice{
				ObjectMeta: metav1.ObjectMeta{Name: "slice-all"},
				Spec: resourcev1.ResourceSliceSpec{
					Driver:   "driver-1",
					AllNodes: ptr.To(true),
					Pool:     resourcev1.ResourcePool{Name: "pool-1"},
				},
			},
			Want: `
				# HELP kube_resourceslice_info Information about resource slice.
				# TYPE kube_resourceslice_info gauge
				kube_resourceslice_info{driver="driver-1",node="",node_selection="all_nodes",pool="pool-1",resourceslice="slice-all"} 1
			`,
			MetricNames: []string{"kube_resourceslice_info"},
		},
		{
			Obj: &resourcev1.ResourceSlice{
				ObjectMeta: metav1.ObjectMeta{Name: "slice-per-device"},
				Spec: resourcev1.ResourceSliceSpec{
					Driver:                 "driver-1",
					PerDeviceNodeSelection: ptr.To(true),
					Pool:                   resourcev1.ResourcePool{Name: "pool-1"},
				},
			},
			Want: `
				# HELP kube_resourceslice_info Information about resource slice.
				# TYPE kube_resourceslice_info gauge
				kube_resourceslice_info{driver="driver-1",node="",node_selection="per_device",pool="pool-1",resourceslice="slice-per-device"} 1
			`,
			MetricNames: []string{"kube_resourceslice_info"},
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
