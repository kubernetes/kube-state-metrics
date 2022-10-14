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
	"k8s.io/utils/pointer"

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
					# HELP kube_persistentvolume_status_phase [STABLE] The phase indicates if a volume is available, bound to a claim, or released by a claim.
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
					# HELP kube_persistentvolume_status_phase [STABLE] The phase indicates if a volume is available, bound to a claim, or released by a claim.
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
					# HELP kube_persistentvolume_status_phase [STABLE] The phase indicates if a volume is available, bound to a claim, or released by a claim.
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
					# HELP kube_persistentvolume_status_phase [STABLE] The phase indicates if a volume is available, bound to a claim, or released by a claim.
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
					# HELP kube_persistentvolume_status_phase [STABLE] The phase indicates if a volume is available, bound to a claim, or released by a claim.
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
					# HELP kube_persistentvolume_status_phase [STABLE] The phase indicates if a volume is available, bound to a claim, or released by a claim.
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="name",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="aws://eu-west-1c/vol-012d34d567890123b",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="azure_disk_1",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="123",fc_target_wwns="0123456789abcdef,abcdef0123456789",fc_wwids="",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="0123456789abcdef,abcdef0123456789",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="iqn.my.test.server.target00",iscsi_lun="123",iscsi_target_portal="1.2.3.4:3260",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="iqn.my.test.initiator:112233",iscsi_iqn="iqn.my.test.server.target00",iscsi_lun="123",iscsi_target_portal="1.2.3.4:3260",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="",local_fs="",nfs_path="/myPath",nfs_server="1.2.3.4",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						CSI: &v1.CSIPersistentVolumeSource{
							Driver:       "test-driver",
							VolumeHandle: "test-volume-handle",
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="test-driver",csi_volume_handle="test-volume-handle",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						Local: &v1.LocalVolumeSource{
							FSType: pointer.String("ext4"),
							Path:   "/mnt/data",
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="/mnt/data",local_fs="ext4",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						Local: &v1.LocalVolumeSource{
							Path: "/mnt/data",
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="/mnt/data",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/mnt/data",
							Type: hostPathTypePointer(v1.HostPathDirectory),
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="/mnt/data",host_path_type="Directory",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
				`,
			MetricNames: []string{"kube_persistentvolume_info"},
		},
		{
			Obj: &v1.PersistentVolume{
				Spec: v1.PersistentVolumeSpec{
					PersistentVolumeSource: v1.PersistentVolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/mnt/data",
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
					# HELP kube_persistentvolume_info [STABLE] Information about persistentvolume.
					# TYPE kube_persistentvolume_info gauge
					kube_persistentvolume_info{azure_disk_name="",ebs_volume_id="",fc_lun="",fc_target_wwns="",fc_wwids="",gce_persistent_disk_name="",host_path="/mnt/data",host_path_type="",iscsi_initiator_name="",iscsi_iqn="",iscsi_lun="",iscsi_target_portal="",local_path="",local_fs="",nfs_path="",nfs_server="",csi_driver="",csi_volume_handle="",persistentvolume="test-pv-available",storageclass=""} 1
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
					# HELP kube_persistentvolume_labels [STABLE] Kubernetes labels converted to Prometheus labels.
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
					# HELP kube_persistentvolume_labels [STABLE] Kubernetes labels converted to Prometheus labels.
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
					# HELP kube_persistentvolume_claim_ref [STABLE] Information about the Persistent Volume Claim Reference.
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
					# HELP kube_persistentvolume_claim_ref [STABLE] Information about the Persistent Volume Claim Reference.
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
					# HELP kube_persistentvolume_capacity_bytes [STABLE] Persistentvolume capacity in bytes.
					# TYPE kube_persistentvolume_capacity_bytes gauge
					kube_persistentvolume_capacity_bytes{persistentvolume="test-pv"} 5.36870912e+09
				`,
			MetricNames: []string{"kube_persistentvolume_capacity_bytes"},
		},
		{
			AllowAnnotationsList: []string{
				"app.k8s.io/owner",
			},
			AllowLabelsList: []string{
				"app",
			},
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-allowlisted-labels-annotations",
					Annotations: map[string]string{
						"app.k8s.io/owner": "mysql-server",
						"foo":              "bar",
					},
					Labels: map[string]string{
						"app":   "mysql-server",
						"hello": "world",
					},
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumePending,
				},
			},
			Want: `
					# HELP kube_persistentvolume_annotations Kubernetes annotations converted to Prometheus labels.
					# HELP kube_persistentvolume_labels [STABLE] Kubernetes labels converted to Prometheus labels.
					# TYPE kube_persistentvolume_annotations gauge
					# TYPE kube_persistentvolume_labels gauge
					kube_persistentvolume_annotations{annotation_app_k8s_io_owner="mysql-server",persistentvolume="test-allowlisted-labels-annotations"} 1
					kube_persistentvolume_labels{label_app="mysql-server",persistentvolume="test-allowlisted-labels-annotations"} 1
`,
			MetricNames: []string{
				"kube_persistentvolume_annotations",
				"kube_persistentvolume_labels",
			},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-defaul-labels-annotations",
					Annotations: map[string]string{
						"app.k8s.io/owner": "mysql-server",
						"foo":              "bar",
					},
					Labels: map[string]string{
						"app":   "mysql-server",
						"hello": "world",
					},
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumePending,
				},
			},
			Want: `
					# HELP kube_persistentvolume_annotations Kubernetes annotations converted to Prometheus labels.
					# HELP kube_persistentvolume_labels [STABLE] Kubernetes labels converted to Prometheus labels.
					# TYPE kube_persistentvolume_annotations gauge
					# TYPE kube_persistentvolume_labels gauge
					kube_persistentvolume_annotations{persistentvolume="test-defaul-labels-annotations"} 1
					kube_persistentvolume_labels{persistentvolume="test-defaul-labels-annotations"} 1
`,
			MetricNames: []string{
				"kube_persistentvolume_annotations",
				"kube_persistentvolume_labels",
			},
		},
		{
			Obj: &v1.PersistentVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-pv-created",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
				},
				Status: v1.PersistentVolumeStatus{
					Phase: v1.VolumePending,
				},
			},
			Want: `
				# HELP kube_persistentvolume_created Unix creation timestamp
				# TYPE kube_persistentvolume_created gauge
				kube_persistentvolume_created{persistentvolume="test-pv-created"} 1.5e+09
`,
			MetricNames: []string{"kube_persistentvolume_created"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(persistentVolumeMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(persistentVolumeMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}

func hostPathTypePointer(p v1.HostPathType) *v1.HostPathType {
	return &p
}
