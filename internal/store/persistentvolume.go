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
	"context"
	"strconv"

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descPersistentVolumeClaimRefName = "kube_persistentvolume_claim_ref"
	descPersistentVolumeClaimRefHelp = "Information about the Persistent Volume Claim Reference."

	descPersistentVolumeAnnotationsName     = "kube_persistentvolume_annotations"
	descPersistentVolumeAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descPersistentVolumeLabelsName          = "kube_persistentvolume_labels"
	descPersistentVolumeLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descPersistentVolumeLabelsDefaultLabels = []string{"persistentvolume"}

	descPersistentVolumeCSIAttributesName = "kube_persistentvolume_csi_attributes"
	descPersistentVolumeCSIAttributesHelp = "CSI attributes of the Persistent Volume."
)

func persistentVolumeMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		createPersistentVolumeClaimRef(),
		createPersistentVolumeAnnotations(allowAnnotationsList),
		createPersistentVolumeLabels(allowLabelsList),
		createPersistentVolumeStatusPhase(),
		createPersistentVolumeInfo(),
		createPersistentVolumeCapacityBytes(),
		createPersistentVolumeCreated(),
		createPersistentVolumeDeletionTimestamp(),
		createPersistentVolumeCSIAttributes(),
		createPersistentVolumeMode(),
	}
}

func wrapPersistentVolumeFunc(f func(*v1.PersistentVolume) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		persistentVolume := obj.(*v1.PersistentVolume)

		metricFamily := f(persistentVolume)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descPersistentVolumeLabelsDefaultLabels, []string{persistentVolume.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createPersistentVolumeListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().PersistentVolumes().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().PersistentVolumes().Watch(context.TODO(), opts)
		},
	}
}

func createPersistentVolumeClaimRef() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		descPersistentVolumeClaimRefName,
		descPersistentVolumeClaimRefHelp,
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
			claimRef := p.Spec.ClaimRef

			if claimRef == nil {
				return &metric.Family{
					Metrics: []*metric.Metric{},
				}
			}
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						LabelKeys: []string{
							"name",
							"claim_namespace",
						},
						LabelValues: []string{
							p.Spec.ClaimRef.Name,
							p.Spec.ClaimRef.Namespace,
						},
						Value: 1,
					},
				},
			}
		}),
	)
}

func createPersistentVolumeAnnotations(allowAnnotationsList []string) generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		descPersistentVolumeAnnotationsName,
		descPersistentVolumeAnnotationsHelp,
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
			if len(allowAnnotationsList) == 0 {
				return &metric.Family{}
			}
			annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", p.Annotations, allowAnnotationsList)
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						LabelKeys:   annotationKeys,
						LabelValues: annotationValues,
						Value:       1,
					},
				},
			}
		}),
	)
}

func createPersistentVolumeLabels(allowLabelsList []string) generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		descPersistentVolumeLabelsName,
		descPersistentVolumeLabelsHelp,
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
			if len(allowLabelsList) == 0 {
				return &metric.Family{}
			}
			labelKeys, labelValues := createPrometheusLabelKeysValues("label", p.Labels, allowLabelsList)
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						LabelKeys:   labelKeys,
						LabelValues: labelValues,
						Value:       1,
					},
				},
			}
		}),
	)
}

func createPersistentVolumeStatusPhase() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_persistentvolume_status_phase",
		"The phase indicates if a volume is available, bound to a claim, or released by a claim.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
			phase := p.Status.Phase

			if phase == "" {
				return &metric.Family{
					Metrics: []*metric.Metric{},
				}
			}

			// Set current phase to 1, others to 0 if it is set.
			ms := []*metric.Metric{
				{
					LabelValues: []string{string(v1.VolumePending)},
					Value:       boolFloat64(phase == v1.VolumePending),
				},
				{
					LabelValues: []string{string(v1.VolumeAvailable)},
					Value:       boolFloat64(phase == v1.VolumeAvailable),
				},
				{
					LabelValues: []string{string(v1.VolumeBound)},
					Value:       boolFloat64(phase == v1.VolumeBound),
				},
				{
					LabelValues: []string{string(v1.VolumeReleased)},
					Value:       boolFloat64(phase == v1.VolumeReleased),
				},
				{
					LabelValues: []string{string(v1.VolumeFailed)},
					Value:       boolFloat64(phase == v1.VolumeFailed),
				},
			}

			for _, m := range ms {
				m.LabelKeys = []string{"phase"}
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPersistentVolumeInfo() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_persistentvolume_info",
		"Information about persistentvolume.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
			var (
				gcePDDiskName,
				ebsVolumeID,
				azureDiskName,
				fcWWIDs, fcLun, fcTargetWWNs,
				iscsiTargetPortal, iscsiIQN, iscsiLun, iscsiInitiatorName,
				nfsServer, nfsPath,
				csiDriver, csiVolumeHandle,
				localFS, localPath,
				hostPath, hostPathType string
			)

			switch {
			case p.Spec.GCEPersistentDisk != nil:
				gcePDDiskName = p.Spec.GCEPersistentDisk.PDName
			case p.Spec.AWSElasticBlockStore != nil:
				ebsVolumeID = p.Spec.AWSElasticBlockStore.VolumeID
			case p.Spec.AzureDisk != nil:
				azureDiskName = p.Spec.AzureDisk.DiskName
			case p.Spec.FC != nil:
				if p.Spec.FC.Lun != nil {
					fcLun = strconv.FormatInt(int64(*p.Spec.FC.Lun), 10)
				}
				for _, wwn := range p.Spec.FC.TargetWWNs {
					if len(fcTargetWWNs) != 0 {
						fcTargetWWNs += ","
					}
					fcTargetWWNs += wwn
				}
				for _, wwid := range p.Spec.FC.WWIDs {
					if len(fcWWIDs) != 0 {
						fcWWIDs += ","
					}
					fcWWIDs += wwid
				}
			case p.Spec.ISCSI != nil:
				iscsiTargetPortal = p.Spec.ISCSI.TargetPortal
				iscsiIQN = p.Spec.ISCSI.IQN
				iscsiLun = strconv.FormatInt(int64(p.Spec.ISCSI.Lun), 10)
				if p.Spec.ISCSI.InitiatorName != nil {
					iscsiInitiatorName = *p.Spec.ISCSI.InitiatorName
				}
			case p.Spec.NFS != nil:
				nfsServer = p.Spec.NFS.Server
				nfsPath = p.Spec.NFS.Path
			case p.Spec.CSI != nil:
				csiDriver = p.Spec.CSI.Driver
				csiVolumeHandle = p.Spec.CSI.VolumeHandle
			case p.Spec.Local != nil:
				localPath = p.Spec.Local.Path
				if p.Spec.Local.FSType != nil {
					localFS = *p.Spec.Local.FSType
				}
			case p.Spec.HostPath != nil:
				hostPath = p.Spec.HostPath.Path
				if p.Spec.HostPath.Type != nil {
					hostPathType = string(*p.Spec.HostPath.Type)
				}
			}

			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						LabelKeys: []string{
							"storageclass",
							"gce_persistent_disk_name",
							"ebs_volume_id",
							"azure_disk_name",
							"fc_wwids",
							"fc_lun",
							"fc_target_wwns",
							"iscsi_target_portal",
							"iscsi_iqn",
							"iscsi_lun",
							"iscsi_initiator_name",
							"nfs_server",
							"nfs_path",
							"csi_driver",
							"csi_volume_handle",
							"local_path",
							"local_fs",
							"host_path",
							"host_path_type",
							"reclaim_policy",
						},
						LabelValues: []string{
							p.Spec.StorageClassName,
							gcePDDiskName,
							ebsVolumeID,
							azureDiskName,
							fcWWIDs,
							fcLun,
							fcTargetWWNs,
							iscsiTargetPortal,
							iscsiIQN,
							iscsiLun,
							iscsiInitiatorName,
							nfsServer,
							nfsPath,
							csiDriver,
							csiVolumeHandle,
							localPath,
							localFS,
							hostPath,
							hostPathType,
							string(p.Spec.PersistentVolumeReclaimPolicy),
						},
						Value: 1,
					},
				},
			}
		}),
	)
}

func createPersistentVolumeCapacityBytes() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_persistentvolume_capacity_bytes",
		"Persistentvolume capacity in bytes.",
		metric.Gauge,
		basemetrics.STABLE,
		"",
		wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
			storage := p.Spec.Capacity[v1.ResourceStorage]
			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						Value: float64(storage.Value()),
					},
				},
			}
		}),
	)
}

func createPersistentVolumeCreated() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_persistentvolume_created",
		"Unix creation timestamp",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
			ms := []*metric.Metric{}

			if !p.CreationTimestamp.IsZero() {
				ms = append(ms, &metric.Metric{
					LabelKeys:   []string{},
					LabelValues: []string{},
					Value:       float64(p.CreationTimestamp.Unix()),
				})
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPersistentVolumeDeletionTimestamp() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_persistentvolume_deletion_timestamp",
		"Unix deletion timestamp",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
			ms := []*metric.Metric{}

			if p.DeletionTimestamp != nil && !p.DeletionTimestamp.IsZero() {
				ms = append(ms, &metric.Metric{
					LabelKeys:   []string{},
					LabelValues: []string{},
					Value:       float64(p.DeletionTimestamp.Unix()),
				})
			}

			return &metric.Family{
				Metrics: ms,
			}
		}),
	)
}

func createPersistentVolumeCSIAttributes() generator.FamilyGenerator {
	return *generator.NewOptInFamilyGenerator(
		descPersistentVolumeCSIAttributesName,
		descPersistentVolumeCSIAttributesHelp,
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
			if p.Spec.CSI == nil {
				return &metric.Family{
					Metrics: []*metric.Metric{},
				}
			}

			var csiMounter, csiMapOptions string
			for k, v := range p.Spec.CSI.VolumeAttributes {
				// storage attributes handled by external CEPH CSI driver
				switch k {
				case "mapOptions":
					csiMapOptions = v
				case "mounter":
					csiMounter = v
				}
			}

			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						LabelKeys: []string{
							"csi_mounter",
							"csi_map_options",
						},
						LabelValues: []string{
							csiMounter,
							csiMapOptions,
						},
						Value: 1,
					},
				},
			}
		}),
	)
}

func createPersistentVolumeMode() generator.FamilyGenerator {
	return *generator.NewFamilyGeneratorWithStability(
		"kube_persistentvolume_volume_mode",
		"Volume Mode information for the PersistentVolume.",
		metric.Gauge,
		basemetrics.ALPHA,
		"",
		wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
			volumeMode := ""
			if p.Spec.VolumeMode != nil {
				volumeMode = string(*p.Spec.VolumeMode)
			} else {
				volumeMode = string("Filesystem") // Filesystem is the default mode used when volumeMode parameter is omitted.
			}

			return &metric.Family{
				Metrics: []*metric.Metric{
					{
						LabelKeys:   []string{"volumemode"},
						LabelValues: []string{volumeMode},
						Value:       1,
					},
				}}
		}),
	)
}
