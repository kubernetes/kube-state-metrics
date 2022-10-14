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

package store

import (
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestPersistentVolumeClaimStore(t *testing.T) {
	storageClassName := "rbd"
	cases := []generateMetricsTestCase{
		// Verify phase enumerations.
		{
			AllowAnnotationsList: []string{
				"app.k8s.io/owner",
			},
			Obj: &v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "mysql-data",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
					Annotations: map[string]string{
						"app":              "mysql-server",
						"app.k8s.io/owner": "@foo",
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
					Conditions: []v1.PersistentVolumeClaimCondition{
						{Type: v1.PersistentVolumeClaimResizing, Status: v1.ConditionTrue},
						{Type: v1.PersistentVolumeClaimFileSystemResizePending, Status: v1.ConditionFalse},
						{Type: v1.PersistentVolumeClaimConditionType("CustomizedType"), Status: v1.ConditionTrue},
					},
				},
			},
			Want: `
				# HELP kube_persistentvolumeclaim_created Unix creation timestamp
				# HELP kube_persistentvolumeclaim_access_mode [STABLE] The access mode(s) specified by the persistent volume claim.
				# HELP kube_persistentvolumeclaim_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_persistentvolumeclaim_info [STABLE] Information about persistent volume claim.
				# HELP kube_persistentvolumeclaim_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_persistentvolumeclaim_resource_requests_storage_bytes [STABLE] The capacity of storage requested by the persistent volume claim.
				# HELP kube_persistentvolumeclaim_status_phase [STABLE] The phase the persistent volume claim is currently in.
				# HELP kube_persistentvolumeclaim_status_condition Information about status of different conditions of persistent volume claim.
				# TYPE kube_persistentvolumeclaim_created gauge
				# TYPE kube_persistentvolumeclaim_access_mode gauge
				# TYPE kube_persistentvolumeclaim_annotations gauge
				# TYPE kube_persistentvolumeclaim_info gauge
				# TYPE kube_persistentvolumeclaim_labels gauge
				# TYPE kube_persistentvolumeclaim_resource_requests_storage_bytes gauge
				# TYPE kube_persistentvolumeclaim_status_phase gauge
				# TYPE kube_persistentvolumeclaim_status_condition gauge
				kube_persistentvolumeclaim_created{namespace="default",persistentvolumeclaim="mysql-data"} 1.5e+09
				kube_persistentvolumeclaim_info{namespace="default",persistentvolumeclaim="mysql-data",storageclass="rbd",volumename="pvc-mysql-data"} 1
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Bound"} 1
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Lost"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Pending"} 0
				kube_persistentvolumeclaim_resource_requests_storage_bytes{namespace="default",persistentvolumeclaim="mysql-data"} 1.073741824e+09
				kube_persistentvolumeclaim_annotations{annotation_app_k8s_io_owner="@foo",namespace="default",persistentvolumeclaim="mysql-data"} 1
				kube_persistentvolumeclaim_labels{namespace="default",persistentvolumeclaim="mysql-data"} 1
				kube_persistentvolumeclaim_access_mode{namespace="default",persistentvolumeclaim="mysql-data",access_mode="ReadWriteOnce"} 1
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="false",condition="CustomizedType"} 0
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="false",condition="FileSystemResizePending"} 1
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="false",condition="Resizing"} 0
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="true",condition="CustomizedType"} 1
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="true",condition="FileSystemResizePending"} 0
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="true",condition="Resizing"} 1
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="unknown",condition="CustomizedType"} 0
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="unknown",condition="FileSystemResizePending"} 0
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="unknown",condition="Resizing"} 0
`,
			MetricNames: []string{"kube_persistentvolumeclaim_info", "kube_persistentvolumeclaim_status_phase", "kube_persistentvolumeclaim_resource_requests_storage_bytes", "kube_persistentvolumeclaim_annotations", "kube_persistentvolumeclaim_labels", "kube_persistentvolumeclaim_access_mode", "kube_persistentvolumeclaim_status_condition", "kube_persistentvolumeclaim_created"},
		},
		{
			AllowLabelsList: []string{
				"app",
			},
			Obj: &v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "mysql-data",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
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
					Conditions: []v1.PersistentVolumeClaimCondition{
						{Type: v1.PersistentVolumeClaimResizing, Status: v1.ConditionTrue},
						{Type: v1.PersistentVolumeClaimFileSystemResizePending, Status: v1.ConditionFalse},
						{Type: v1.PersistentVolumeClaimConditionType("CustomizedType"), Status: v1.ConditionTrue},
					},
				},
			},
			Want: `
				# HELP kube_persistentvolumeclaim_created Unix creation timestamp
				# HELP kube_persistentvolumeclaim_access_mode [STABLE] The access mode(s) specified by the persistent volume claim.
				# HELP kube_persistentvolumeclaim_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_persistentvolumeclaim_info [STABLE] Information about persistent volume claim.
				# HELP kube_persistentvolumeclaim_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_persistentvolumeclaim_resource_requests_storage_bytes [STABLE] The capacity of storage requested by the persistent volume claim.
				# HELP kube_persistentvolumeclaim_status_phase [STABLE] The phase the persistent volume claim is currently in.
				# HELP kube_persistentvolumeclaim_status_condition Information about status of different conditions of persistent volume claim.
				# TYPE kube_persistentvolumeclaim_created gauge
				# TYPE kube_persistentvolumeclaim_access_mode gauge
				# TYPE kube_persistentvolumeclaim_annotations gauge
				# TYPE kube_persistentvolumeclaim_info gauge
				# TYPE kube_persistentvolumeclaim_labels gauge
				# TYPE kube_persistentvolumeclaim_resource_requests_storage_bytes gauge
				# TYPE kube_persistentvolumeclaim_status_phase gauge
				# TYPE kube_persistentvolumeclaim_status_condition gauge
				kube_persistentvolumeclaim_created{namespace="default",persistentvolumeclaim="mysql-data"} 1.5e+09
				kube_persistentvolumeclaim_info{namespace="default",persistentvolumeclaim="mysql-data",storageclass="rbd",volumename="pvc-mysql-data"} 1
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Bound"} 1
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Lost"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="mysql-data",phase="Pending"} 0
				kube_persistentvolumeclaim_resource_requests_storage_bytes{namespace="default",persistentvolumeclaim="mysql-data"} 1.073741824e+09
				kube_persistentvolumeclaim_annotations{namespace="default",persistentvolumeclaim="mysql-data"} 1
				kube_persistentvolumeclaim_labels{label_app="mysql-server",namespace="default",persistentvolumeclaim="mysql-data"} 1
				kube_persistentvolumeclaim_access_mode{namespace="default",persistentvolumeclaim="mysql-data",access_mode="ReadWriteOnce"} 1
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="false",condition="CustomizedType"} 0
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="false",condition="FileSystemResizePending"} 1
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="false",condition="Resizing"} 0
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="true",condition="CustomizedType"} 1
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="true",condition="FileSystemResizePending"} 0
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="true",condition="Resizing"} 1
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="unknown",condition="CustomizedType"} 0
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="unknown",condition="FileSystemResizePending"} 0
				kube_persistentvolumeclaim_status_condition{namespace="default",persistentvolumeclaim="mysql-data",status="unknown",condition="Resizing"} 0
`,
			MetricNames: []string{"kube_persistentvolumeclaim_info", "kube_persistentvolumeclaim_status_phase", "kube_persistentvolumeclaim_resource_requests_storage_bytes", "kube_persistentvolumeclaim_annotations", "kube_persistentvolumeclaim_labels", "kube_persistentvolumeclaim_access_mode", "kube_persistentvolumeclaim_status_condition", "kube_persistentvolumeclaim_created"},
		},
		{
			Obj: &v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "prometheus-data",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "default",
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
				# HELP kube_persistentvolumeclaim_created Unix creation timestamp
				# HELP kube_persistentvolumeclaim_access_mode [STABLE] The access mode(s) specified by the persistent volume claim.
				# HELP kube_persistentvolumeclaim_info [STABLE] Information about persistent volume claim.
				# HELP kube_persistentvolumeclaim_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_persistentvolumeclaim_resource_requests_storage_bytes [STABLE] The capacity of storage requested by the persistent volume claim.
				# HELP kube_persistentvolumeclaim_status_phase [STABLE] The phase the persistent volume claim is currently in.
				# HELP kube_persistentvolumeclaim_status_condition Information about status of different conditions of persistent volume claim.
				# TYPE kube_persistentvolumeclaim_created gauge
				# TYPE kube_persistentvolumeclaim_access_mode gauge
				# TYPE kube_persistentvolumeclaim_info gauge
				# TYPE kube_persistentvolumeclaim_labels gauge
				# TYPE kube_persistentvolumeclaim_resource_requests_storage_bytes gauge
				# TYPE kube_persistentvolumeclaim_status_phase gauge
				# TYPE kube_persistentvolumeclaim_status_condition gauge
				kube_persistentvolumeclaim_created{namespace="default",persistentvolumeclaim="prometheus-data"} 1.5e+09
				kube_persistentvolumeclaim_info{namespace="default",persistentvolumeclaim="prometheus-data",storageclass="rbd",volumename="pvc-prometheus-data"} 1
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="prometheus-data",phase="Bound"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="prometheus-data",phase="Lost"} 0
				kube_persistentvolumeclaim_status_phase{namespace="default",persistentvolumeclaim="prometheus-data",phase="Pending"} 1
				kube_persistentvolumeclaim_labels{namespace="default",persistentvolumeclaim="prometheus-data"} 1
				kube_persistentvolumeclaim_access_mode{namespace="default",persistentvolumeclaim="prometheus-data",access_mode="ReadWriteOnce"} 1
			`,
			MetricNames: []string{"kube_persistentvolumeclaim_info", "kube_persistentvolumeclaim_status_phase", "kube_persistentvolumeclaim_resource_requests_storage_bytes", "kube_persistentvolumeclaim_labels", "kube_persistentvolumeclaim_access_mode", "kube_persistentvolumeclaim_status_condition", "kube_persistentvolumeclaim_created"},
		},
		{
			Obj: &v1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "mongo-data",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Spec: v1.PersistentVolumeClaimSpec{
					AccessModes: []v1.PersistentVolumeAccessMode{
						v1.ReadWriteOnce,
					},
				},
				Status: v1.PersistentVolumeClaimStatus{
					Phase: v1.ClaimLost,
					Conditions: []v1.PersistentVolumeClaimCondition{
						{Type: v1.PersistentVolumeClaimResizing, Status: v1.ConditionFalse},
						{Type: v1.PersistentVolumeClaimFileSystemResizePending, Status: v1.ConditionTrue},
						{Type: v1.PersistentVolumeClaimConditionType("CustomizedType"), Status: v1.ConditionFalse},
					},
				},
			},
			Want: `
				# HELP kube_persistentvolumeclaim_created Unix creation timestamp
				# HELP kube_persistentvolumeclaim_access_mode [STABLE] The access mode(s) specified by the persistent volume claim.
				# HELP kube_persistentvolumeclaim_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_persistentvolumeclaim_info [STABLE] Information about persistent volume claim.
				# HELP kube_persistentvolumeclaim_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_persistentvolumeclaim_resource_requests_storage_bytes [STABLE] The capacity of storage requested by the persistent volume claim.
				# HELP kube_persistentvolumeclaim_status_phase [STABLE] The phase the persistent volume claim is currently in.
				# HELP kube_persistentvolumeclaim_status_condition Information about status of different conditions of persistent volume claim.
				# TYPE kube_persistentvolumeclaim_created gauge
				# TYPE kube_persistentvolumeclaim_access_mode gauge
				# TYPE kube_persistentvolumeclaim_annotations gauge
				# TYPE kube_persistentvolumeclaim_info gauge
				# TYPE kube_persistentvolumeclaim_labels gauge
				# TYPE kube_persistentvolumeclaim_resource_requests_storage_bytes gauge
				# TYPE kube_persistentvolumeclaim_status_phase gauge
				# TYPE kube_persistentvolumeclaim_status_condition gauge
				kube_persistentvolumeclaim_created{namespace="",persistentvolumeclaim="mongo-data"} 1.5e+09
				kube_persistentvolumeclaim_info{namespace="",persistentvolumeclaim="mongo-data",storageclass="<none>",volumename=""} 1
				kube_persistentvolumeclaim_status_phase{namespace="",persistentvolumeclaim="mongo-data",phase="Bound"} 0
				kube_persistentvolumeclaim_status_phase{namespace="",persistentvolumeclaim="mongo-data",phase="Lost"} 1
				kube_persistentvolumeclaim_status_phase{namespace="",persistentvolumeclaim="mongo-data",phase="Pending"} 0
				kube_persistentvolumeclaim_labels{namespace="",persistentvolumeclaim="mongo-data"} 1
				kube_persistentvolumeclaim_annotations{namespace="",persistentvolumeclaim="mongo-data"} 1
				kube_persistentvolumeclaim_access_mode{namespace="",persistentvolumeclaim="mongo-data",access_mode="ReadWriteOnce"} 1
				kube_persistentvolumeclaim_status_condition{namespace="",persistentvolumeclaim="mongo-data",status="false",condition="CustomizedType"} 1
				kube_persistentvolumeclaim_status_condition{namespace="",persistentvolumeclaim="mongo-data",status="false",condition="FileSystemResizePending"} 0
				kube_persistentvolumeclaim_status_condition{namespace="",persistentvolumeclaim="mongo-data",status="false",condition="Resizing"} 1
				kube_persistentvolumeclaim_status_condition{namespace="",persistentvolumeclaim="mongo-data",status="true",condition="CustomizedType"} 0
				kube_persistentvolumeclaim_status_condition{namespace="",persistentvolumeclaim="mongo-data",status="true",condition="FileSystemResizePending"} 1
				kube_persistentvolumeclaim_status_condition{namespace="",persistentvolumeclaim="mongo-data",status="true",condition="Resizing"} 0
				kube_persistentvolumeclaim_status_condition{namespace="",persistentvolumeclaim="mongo-data",status="unknown",condition="CustomizedType"} 0
				kube_persistentvolumeclaim_status_condition{namespace="",persistentvolumeclaim="mongo-data",status="unknown",condition="FileSystemResizePending"} 0
				kube_persistentvolumeclaim_status_condition{namespace="",persistentvolumeclaim="mongo-data",status="unknown",condition="Resizing"} 0
`,
			MetricNames: []string{"kube_persistentvolumeclaim_created", "kube_persistentvolumeclaim_info", "kube_persistentvolumeclaim_status_phase", "kube_persistentvolumeclaim_resource_requests_storage_bytes", "kube_persistentvolumeclaim_annotations", "kube_persistentvolumeclaim_labels", "kube_persistentvolumeclaim_access_mode", "kube_persistentvolumeclaim_status_condition"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(persistentVolumeClaimMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(persistentVolumeClaimMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
