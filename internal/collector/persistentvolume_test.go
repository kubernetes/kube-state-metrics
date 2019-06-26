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

package collector

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/metric"
)

func TestPersistentVolumeCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
			# HELP kube_persistentvolume_status_phase The phase indicates if a volume is available, bound to a claim, or released by a claim.
			# TYPE kube_persistentvolume_status_phase gauge
			# HELP kube_persistentvolume_labels Kubernetes labels converted to Prometheus labels.
			# TYPE kube_persistentvolume_labels gauge
			# HELP kube_persistentvolume_info Information about persistentvolume.
			# TYPE kube_persistentvolume_info gauge
			# HELP kube_persistentvolume_capacity_bytes The size of the Persistentvolume in bytes.
			# TYPE kube_persistentvolume_capacity_bytes gauge
	`
	cases := []generateMetricsTestCase{
		// Verify phase enumerations.
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-pending",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumePending,
				},
			},
			Want: `
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Available"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Bound"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Failed"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Pending"} 1
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Released"} 0
`,
			MetricNames: []string{
				"kube_persistentvolume_status_phase",
			},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-available",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					kube_persistentvolume_status_phase{persistentvolume="test-pv-available",phase="Available"} 1
					kube_persistentvolume_status_phase{persistentvolume="test-pv-available",phase="Bound"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-available",phase="Failed"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-available",phase="Pending"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-available",phase="Released"} 0
`,
			MetricNames: []string{"kube_persistentvolume_status_phase"},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-bound",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeBound,
				},
			},
			Want: `
					kube_persistentvolume_status_phase{persistentvolume="test-pv-bound",phase="Available"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-bound",phase="Bound"} 1
					kube_persistentvolume_status_phase{persistentvolume="test-pv-bound",phase="Failed"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-bound",phase="Pending"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-bound",phase="Released"} 0
`,
			MetricNames: []string{"kube_persistentvolume_status_phase"},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-released",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeReleased,
				},
			},
			Want: `
					kube_persistentvolume_status_phase{persistentvolume="test-pv-released",phase="Available"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-released",phase="Bound"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-released",phase="Failed"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-released",phase="Pending"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-released",phase="Released"} 1
`,
			MetricNames: []string{"kube_persistentvolume_status_phase"},
		},
		{

			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-failed",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeFailed,
				},
			},
			Want: `
					kube_persistentvolume_status_phase{persistentvolume="test-pv-failed",phase="Available"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-failed",phase="Bound"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-failed",phase="Failed"} 1
					kube_persistentvolume_status_phase{persistentvolume="test-pv-failed",phase="Pending"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-failed",phase="Released"} 0
`,
			MetricNames: []string{"kube_persistentvolume_status_phase"},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-pending",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumePending,
				},
				Spec: v1.PersistentVolumeSpec{
					StorageClassName: "test",
				},
			},
			Want: `
				kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Available"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Bound"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Failed"} 0
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Pending"} 1
					kube_persistentvolume_status_phase{persistentvolume="test-pv-pending",phase="Released"} 0
`,
			MetricNames: []string{
				"kube_persistentvolume_status_phase",
			},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-available",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					kube_persistentvolume_info{persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-labeled-pv",
					Labels: map[string]string{
						"app": "mysql-server",
					},
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumePending,
				},
				Spec: v1.PersistentVolumeSpec{
					StorageClassName: "test",
				},
			},
			Want: `
					kube_persistentvolume_labels{label_app="mysql-server",persistentvolume="test-labeled-pv"} 1
				`,
			MetricNames: []string{"kube_persistentvolume_labels"},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-unlabeled-pv",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					kube_persistentvolume_labels{persistentvolume="test-unlabeled-pv"} 1
				`,
			MetricNames: []string{"kube_persistentvolume_labels"},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv",
				},
				Spec: v1.PersistentVolumeSpec{
					Capacity: v1.ResourceList{
						v1.ResourceStorage: resource.MustParse("5Gi"),
					},
				},
			},
			Want: `
					kube_persistentvolume_capacity_bytes{persistentvolume="test-pv"} 5.36870912e+09
				`,
			MetricNames: []string{"kube_persistentvolume_capacity_bytes"},
		},
	}
	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(persistentVolumeMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
