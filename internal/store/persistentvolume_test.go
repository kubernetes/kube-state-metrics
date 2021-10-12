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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestPersistentVolumeStore(t *testing.T) {
	iscsiInitiatorName := "iqn.my.test.initiator:112233"
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
					# HELP kube_persistentvolume_status_phase The phase indicates if a volume is available, bound to a claim, or released by a claim.
					# TYPE kube_persistentvolume_status_phase gauge
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
					# HELP kube_persistentvolume_status_phase The phase indicates if a volume is available, bound to a claim, or released by a claim.
					# TYPE kube_persistentvolume_status_phase gauge
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
					# HELP kube_persistentvolume_status_phase The phase indicates if a volume is available, bound to a claim, or released by a claim.
					# TYPE kube_persistentvolume_status_phase gauge
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
					# HELP kube_persistentvolume_status_phase The phase indicates if a volume is available, bound to a claim, or released by a claim.
					# TYPE kube_persistentvolume_status_phase gauge
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
					# HELP kube_persistentvolume_status_phase The phase indicates if a volume is available, bound to a claim, or released by a claim.
					# TYPE kube_persistentvolume_status_phase gauge
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
					# HELP kube_persistentvolume_status_phase The phase indicates if a volume is available, bound to a claim, or released by a claim.
					# TYPE kube_persistentvolume_status_phase gauge
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
					# HELP kube_persistentvolume_info Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",nfs_path="",nfs_server="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-available",
					Labels: map[string]string{
						"fc_lun": "456",
					},
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					# HELP kube_persistentvolume_info Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",nfs_path="",nfs_server="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						GCEPersistentDisk: &v1.GCEPersistentDiskVolumeSource{
							PDName: "name",
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-available",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					# HELP kube_persistentvolume_info Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="name",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",nfs_path="",nfs_server="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						AWSElasticBlockStore: &v1.AWSElasticBlockStoreVolumeSource{
							VolumeID: "aws://eu-west-1c/vol-012d34d567890123b",
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-available",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					# HELP kube_persistentvolume_info Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="aws://eu-west-1c/vol-012d34d567890123b",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",nfs_path="",nfs_server="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						AzureDisk: &v1.AzureDiskVolumeSource{
							DiskName: "azure_disk_1",
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-available",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					# HELP kube_persistentvolume_info Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="azure_disk_1",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",nfs_path="",nfs_server="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						FC: &v1.FCVolumeSource{
							Lun:        int32ptr(123),
							TargetWWNs: []string{"0123456789abcdef", "abcdef0123456789"},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-available",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					# HELP kube_persistentvolume_info Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="123",fc_target_wwns="0123456789abcdef,abcdef0123456789",fc_wwids="",gce_persistent_disk_name="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",nfs_path="",nfs_server="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						FC: &v1.FCVolumeSource{
							WWIDs: []string{"0123456789abcdef", "abcdef0123456789"},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-available",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					# HELP kube_persistentvolume_info Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="0123456789abcdef,abcdef0123456789",gce_persistent_disk_name="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",nfs_path="",nfs_server="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						ISCSI: &v1.ISCSIPersistentVolumeSource{
							TargetPortal: "1.2.3.4:3260",
							IQN:          "iqn.my.test.server.target00",
							Lun:          int32(123),
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-available",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					# HELP kube_persistentvolume_info Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",iscsi_initiator_name="",iscsi_iqn="iqn.my.test.server.target00",iscsi_lun="123",iscsi_target_portal="1.2.3.4:3260",nfs_path="",nfs_server="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						ISCSI: &v1.ISCSIPersistentVolumeSource{
							TargetPortal:  "1.2.3.4:3260",
							IQN:           "iqn.my.test.server.target00",
							Lun:           int32(123),
							InitiatorName: &iscsiInitiatorName,
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-available",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					# HELP kube_persistentvolume_info Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",iscsi_initiator_name="iqn.my.test.initiator:112233",iscsi_iqn="iqn.my.test.server.target00",iscsi_lun="123",iscsi_target_portal="1.2.3.4:3260",nfs_path="",nfs_server="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						NFS: &v1.NFSVolumeSource{
							Server: "1.2.3.4",
							Path:   "/myPath",
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-available",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					# HELP kube_persistentvolume_info Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",nfs_path="/myPath",nfs_server="1.2.3.4",persistentvolume="test-pv-available",storageclass=""} 1
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
					# HELP kube_persistentvolume_labels Kubernetes labels converted to Prometheus labels.
					# TYPE kube_persistentvolume_labels gauge
					kube_persistentvolume_labels{persistentvolume="test-labeled-pv"} 1
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
					# HELP kube_persistentvolume_labels Kubernetes labels converted to Prometheus labels.
					# TYPE kube_persistentvolume_labels gauge
					kube_persistentvolume_labels{persistentvolume="test-unlabeled-pv"} 1
				`,
			MetricNames: []string{"kube_persistentvolume_labels"},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-claimed-pv",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumePending,
				},
				Spec: v1.PersistentVolumeSpec{
					StorageClassName: "test",
					ClaimRef: &v1.ObjectReference{
						APIVersion: "v1",
						Kind:       "PersistentVolumeClaim",
						Name:       "pv-claim",
						Namespace:  "default",
					},
				},
			},
			Want: `
					# HELP kube_persistentvolume_claim_ref Information about the Persitant Volume Claim Reference.
					# TYPE kube_persistentvolume_claim_ref gauge
					kube_persistentvolume_claim_ref{claim_namespace="default",name="pv-claim",persistentvolume="test-claimed-pv"} 1
				`,
			MetricNames: []string{"kube_persistentvolume_claim_ref"},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-unclaimed-pv",
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumeAvailable,
				},
			},
			Want: `
					# HELP kube_persistentvolume_claim_ref Information about the Persitant Volume Claim Reference.
					# TYPE kube_persistentvolume_claim_ref gauge
				`,
			MetricNames: []string{"kube_persistentvolume_claim_ref"},
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
					# HELP kube_persistentvolume_capacity_bytes Persistentvolume capacity in bytes.
					# TYPE kube_persistentvolume_capacity_bytes gauge
					kube_persistentvolume_capacity_bytes{persistentvolume="test-pv"} 5.36870912e+09
				`,
			MetricNames: []string{"kube_persistentvolume_capacity_bytes"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(persistentVolumeMetricFamilies(nil, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(persistentVolumeMetricFamilies(nil, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
