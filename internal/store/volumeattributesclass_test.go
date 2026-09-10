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

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestVolumeAttributesClassStore(t *testing.T) {
	startTime := 1501569018
	metav1StartTime := metav1.Unix(int64(startTime), 0)

	cases := []generateMetricsTestCase{
		{
			Obj: &storagev1.VolumeAttributesClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_volumeattributesclass-info",
				},
				DriverName: "ebs.csi.aws.com",
				Parameters: map[string]string{
					"iops": "6000",
				},
			},
			Want: `
					# HELP kube_volumeattributesclass_info Information about volumeattributesclass.
					# TYPE kube_volumeattributesclass_info gauge
					kube_volumeattributesclass_info{volumeattributesclass="test_volumeattributesclass-info",driver_name="ebs.csi.aws.com"} 1
				`,
			MetricNames: []string{
				"kube_volumeattributesclass_info",
			},
		},
		{
			Obj: &storagev1.VolumeAttributesClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test_kube_volumeattributesclass-created",
					CreationTimestamp: metav1StartTime,
				},
				DriverName: "ebs.csi.aws.com",
			},
			Want: `
					# HELP kube_volumeattributesclass_created Unix creation timestamp
					# TYPE kube_volumeattributesclass_created gauge
					kube_volumeattributesclass_created{volumeattributesclass="test_kube_volumeattributesclass-created"} 1.501569018e+09
				`,
			MetricNames: []string{
				"kube_volumeattributesclass_created",
			},
		},
		{
			Obj: &storagev1.VolumeAttributesClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test_volumeattributesclass-labels",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				DriverName: "ebs.csi.aws.com",
			},
			Want: `
					# HELP kube_volumeattributesclass_labels Kubernetes labels converted to Prometheus labels.
					# TYPE kube_volumeattributesclass_labels gauge
				`,
			MetricNames: []string{
				"kube_volumeattributesclass_labels",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(volumeAttributesClassMetricFamilies(nil, nil, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(volumeAttributesClassMetricFamilies(nil, nil, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}

	parametersCase := generateMetricsTestCase{
		Obj: &storagev1.VolumeAttributesClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test_volumeattributesclass-parameters",
			},
			DriverName: "ebs.csi.aws.com",
			Parameters: map[string]string{
				"iops":       "6000",
				"throughput": "250",
				"unallowed":  "value",
			},
		},
		Want: `
				# HELP kube_volumeattributesclass_parameters VolumeAttributesClass parameters converted to Prometheus labels.
				# TYPE kube_volumeattributesclass_parameters gauge
				kube_volumeattributesclass_parameters{volumeattributesclass="test_volumeattributesclass-parameters",parameter_iops="6000",parameter_throughput="250"} 1
			`,
		MetricNames: []string{
			"kube_volumeattributesclass_parameters",
		},
	}
	parametersCase.Func = generator.ComposeMetricGenFuncs(volumeAttributesClassMetricFamilies(nil, nil, []string{"iops", "throughput"}))
	parametersCase.Headers = generator.ExtractMetricFamilyHeaders(volumeAttributesClassMetricFamilies(nil, nil, []string{"iops", "throughput"}))
	if err := parametersCase.run(); err != nil {
		t.Errorf("unexpected collecting result in parameters allowlist run:\n%s", err)
	}
}
