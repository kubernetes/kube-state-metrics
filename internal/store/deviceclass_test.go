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

	resourcev1 "k8s.io/api/resource/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestDeviceClassStore(t *testing.T) {
	const metadata = `
		# HELP kube_deviceclass_created Unix creation timestamp
		# TYPE kube_deviceclass_created gauge
		# HELP kube_deviceclass_info Information about device class.
		# TYPE kube_deviceclass_info gauge
	`

	cases := []generateMetricsTestCase{
		{
			Obj: &resourcev1.DeviceClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "nvidia-gpu",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Spec: resourcev1.DeviceClassSpec{
					ExtendedResourceName: ptr.To("nvidia.com/gpu"),
				},
			},
			Want: metadata + `
				kube_deviceclass_created{deviceclass="nvidia-gpu"} 1.5e+09
				kube_deviceclass_info{deviceclass="nvidia-gpu",extended_resource_name="nvidia.com/gpu"} 1
			`,
			MetricNames: []string{
				"kube_deviceclass_created",
				"kube_deviceclass_info",
			},
		},
		{
			Obj: &resourcev1.DeviceClass{
				ObjectMeta: metav1.ObjectMeta{Name: "no-extended-resource"},
			},
			Want: `
				# HELP kube_deviceclass_info Information about device class.
				# TYPE kube_deviceclass_info gauge
				kube_deviceclass_info{deviceclass="no-extended-resource",extended_resource_name=""} 1
			`,
			MetricNames: []string{"kube_deviceclass_info"},
		},
		{
			AllowAnnotationsList: []string{"team"},
			AllowLabelsList:      []string{"app"},
			Obj: &resourcev1.DeviceClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "deviceclass-labels",
					Annotations: map[string]string{"team": "gpu", "ignored": "x"},
					Labels:      map[string]string{"app": "trainer", "ignored": "x"},
				},
			},
			Want: `
				# HELP kube_deviceclass_annotations Kubernetes annotations converted to Prometheus labels.
				# TYPE kube_deviceclass_annotations gauge
				# HELP kube_deviceclass_labels Kubernetes labels converted to Prometheus labels.
				# TYPE kube_deviceclass_labels gauge
				kube_deviceclass_annotations{annotation_team="gpu",deviceclass="deviceclass-labels"} 1
				kube_deviceclass_labels{label_app="trainer",deviceclass="deviceclass-labels"} 1
			`,
			MetricNames: []string{"kube_deviceclass_annotations", "kube_deviceclass_labels"},
		},
	}

	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(deviceClassMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(deviceClassMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %dth run:\n%v", i, err)
		}
	}
}
