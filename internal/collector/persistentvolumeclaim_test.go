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

func TestPersistentVolumeClaimCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_persistentvolumeclaim_info Information about persistent volume claim.
		# TYPE kube_persistentvolumeclaim_info gauge
		# HELP kube_persistentvolumeclaim_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_persistentvolumeclaim_labels gauge
		# HELP kube_persistentvolumeclaim_status_phase The phase the persistent volume claim is currently in.
		# TYPE kube_persistentvolumeclaim_status_phase gauge
		# HELP kube_persistentvolumeclaim_resource_requests_storage_bytes The capacity of storage requested by the persistent volume claim.
		# TYPE kube_persistentvolumeclaim_resource_requests_storage_bytes gauge
		# HELP kube_persistentvolumeclaim_access_mode The access mode of the persistent volume.
		# TYPE kube_persistentvolumeclaim_access_mode gauge
	`
	storageClassName := "rbd"
	cases := []generateMetricsTestCase{
		// Verify phase enumerations.
		{
			Obj: &v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mysql-data",
					Namespace: "default",
					Labels: map[string]string{
						"app": "mysql-server",
					},
				},
				Spec: v1.PersistentVolumeClaimSpec{
					AccessModes: []v1.PersistentVolumeAccessMode{
						v1.ReadWriteOnce,
					},
					StorageClassName: &storageClassName,
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
					VolumeName: "pvc-mysql-data",
				},
				Status: v1.PersistentVolumeClaimStatus{
					Phase: v1.ClaimBound,
				},
			},
			Want: `
				kube_persistentvolumeclaim_info{namespace="default",persistentvolumeclaim="mysql-data",storageclass="rbd",volumename="pvc-mysql-data"} 1
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Bound"} 1
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Lost"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Pending"} 0
				kube_persistentvolumeclaim_resource_requests_storage_bytes{namespace="default",persistentvolumeclaim="mysql-data"} 1.073741824e+09
				kube_persistentvolumeclaim_labels{label_app="mysql-server",namespace="default",persistentvolumeclaim="mysql-data"} 1
				kube_persistentvolumeclaim_access_mode{namespace="default",persistentvolumeclaim="mysql-data",access_mode="ReadWriteOnce"} 1
`,
			MetricNames: []string{"kube_persistentvolumeclaim_info", "kube_persistentvolumeclaim_status_phase", "kube_persistentvolumeclaim_resource_requests_storage_bytes", "kube_persistentvolumeclaim_labels", "kube_persistentvolumeclaim_access_mode"},
		},
		{
			Obj: &v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "prometheus-data",
					Namespace: "default",
				},
				Spec: v1.PersistentVolumeClaimSpec{
					AccessModes: []v1.PersistentVolumeAccessMode{
						v1.ReadWriteOnce,
					},
					StorageClassName: &storageClassName,
					VolumeName:       "pvc-prometheus-data",
				},
				Status: v1.PersistentVolumeClaimStatus{
					Phase: v1.ClaimPending,
				},
			},
			Want: `
				kube_persistentvolumeclaim_info{namespace="default",persistentvolumeclaim="prometheus-data",storageclass="rbd",volumename="pvc-prometheus-data"} 1
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="prometheus-data",phase="Bound"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="prometheus-data",phase="Lost"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="prometheus-data",phase="Pending"} 1
				kube_persistentvolumeclaim_labels{namespace="default",persistentvolumeclaim="prometheus-data"} 1
				kube_persistentvolumeclaim_access_mode{namespace="default",persistentvolumeclaim="prometheus-data",access_mode="ReadWriteOnce"} 1
			`,
			MetricNames: []string{"kube_persistentvolumeclaim_info", "kube_persistentvolumeclaim_status_phase", "kube_persistentvolumeclaim_resource_requests_storage_bytes", "kube_persistentvolumeclaim_labels", "kube_persistentvolumeclaim_access_mode"},
		},
		{
			Obj: &v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mongo-data",
				},
				Spec: v1.PersistentVolumeClaimSpec{
					AccessModes: []v1.PersistentVolumeAccessMode{
						v1.ReadWriteOnce,
					},
				},
				Status: v1.PersistentVolumeClaimStatus{
					Phase: v1.ClaimLost,
				},
			},
			Want: `
				kube_persistentvolumeclaim_info{namespace="",persistentvolumeclaim="mongo-data",storageclass="<none>",volumename=""} 1
				kube_persistentvolumeclaim_status_phase{namespace="",persistentvolumeclaim="mongo-data",phase="Bound"} 0
				kube_persistentvolumeclaim_status_phase{namespace="",persistentvolumeclaim="mongo-data",phase="Lost"} 1
				kube_persistentvolumeclaim_status_phase{namespace="",persistentvolumeclaim="mongo-data",phase="Pending"} 0
				kube_persistentvolumeclaim_labels{namespace="",persistentvolumeclaim="mongo-data"} 1
				kube_persistentvolumeclaim_access_mode{namespace="",persistentvolumeclaim="mongo-data",access_mode="ReadWriteOnce"} 1
`,
			MetricNames: []string{"kube_persistentvolumeclaim_info", "kube_persistentvolumeclaim_status_phase", "kube_persistentvolumeclaim_resource_requests_storage_bytes", "kube_persistentvolumeclaim_labels", "kube_persistentvolumeclaim_access_mode"},
		},
	}
	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(persistentVolumeClaimMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
