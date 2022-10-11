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
	descPersistentVolumeClaimRefName          = "kube_persistentvolume_claim_ref"
	descPersistentVolumeClaimRefHelp          = "Information about the Persistent Volume Claim Reference."
	descPersistentVolumeClaimRefDefaultLabels = []string{"persistentvolume"}

	descPersistentVolumeAnnotationsName     = "kube_persistentvolume_annotations"
	descPersistentVolumeAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descPersistentVolumeLabelsName          = "kube_persistentvolume_labels"
	descPersistentVolumeLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descPersistentVolumeLabelsDefaultLabels = []string{"persistentvolume"}
)

func persistentVolumeMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
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
		),
		*generator.NewFamilyGenerator(
			descPersistentVolumeAnnotationsName,
			descPersistentVolumeAnnotationsHelp,
			metric.Gauge,
			"",
			wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
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
		),
		*generator.NewFamilyGeneratorWithStability(
			descPersistentVolumeLabelsName,
			descPersistentVolumeLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapPersistentVolumeFunc(func(p *v1.PersistentVolume) *metric.Family {
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
		),
		*generator.NewFamilyGeneratorWithStability(
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
		),
		*generator.NewFamilyGeneratorWithStability(
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
				case p.Spec.PersistentVolumeSource.GCEPersistentDisk != nil:
					gcePDDiskName = p.Spec.PersistentVolumeSource.GCEPersistentDisk.PDName
				case p.Spec.PersistentVolumeSource.AWSElasticBlockStore != nil:
					ebsVolumeID = p.Spec.PersistentVolumeSource.AWSElasticBlockStore.VolumeID
				case p.Spec.PersistentVolumeSource.AzureDisk != nil:
					azureDiskName = p.Spec.PersistentVolumeSource.AzureDisk.DiskName
				case p.Spec.PersistentVolumeSource.FC != nil:
					if p.Spec.PersistentVolumeSource.FC.Lun != nil {
						fcLun = strconv.FormatInt(int64(*p.Spec.PersistentVolumeSource.FC.Lun), 10)
					}
					for _, wwn := range p.Spec.PersistentVolumeSource.FC.TargetWWNs {
						if len(fcTargetWWNs) != 0 {
							fcTargetWWNs += ","
						}
						fcTargetWWNs += wwn
					}
					for _, wwid := range p.Spec.PersistentVolumeSource.FC.WWIDs {
						if len(fcWWIDs) != 0 {
							fcWWIDs += ","
						}
						fcWWIDs += wwid
					}
				case p.Spec.PersistentVolumeSource.ISCSI != nil:
					iscsiTargetPortal = p.Spec.PersistentVolumeSource.ISCSI.TargetPortal
					iscsiIQN = p.Spec.PersistentVolumeSource.ISCSI.IQN
					iscsiLun = strconv.FormatInt(int64(p.Spec.PersistentVolumeSource.ISCSI.Lun), 10)
					if p.Spec.PersistentVolumeSource.ISCSI.InitiatorName != nil {
						iscsiInitiatorName = *p.Spec.PersistentVolumeSource.ISCSI.InitiatorName
					}
				case p.Spec.PersistentVolumeSource.NFS != nil:
					nfsServer = p.Spec.PersistentVolumeSource.NFS.Server
					nfsPath = p.Spec.PersistentVolumeSource.NFS.Path
				case p.Spec.PersistentVolumeSource.CSI != nil:
					csiDriver = p.Spec.PersistentVolumeSource.CSI.Driver
					csiVolumeHandle = p.Spec.PersistentVolumeSource.CSI.VolumeHandle
				case p.Spec.PersistentVolumeSource.Local != nil:
					localPath = p.Spec.PersistentVolumeSource.Local.Path
					if p.Spec.PersistentVolumeSource.Local.FSType != nil {
						localFS = *p.Spec.PersistentVolumeSource.Local.FSType
					}
				case p.Spec.PersistentVolumeSource.HostPath != nil:
					hostPath = p.Spec.PersistentVolumeSource.HostPath.Path
					if p.Spec.PersistentVolumeSource.HostPath.Type != nil {
						hostPathType = string(*p.Spec.PersistentVolumeSource.HostPath.Type)
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
							},
							Value: 1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
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
		),
		*generator.NewFamilyGenerator(
			"kube_persistentvolume_created",
			"Unix creation timestamp",
			metric.Gauge,
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
		),
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

func createPersistentVolumeListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().PersistentVolumes().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().PersistentVolumes().Watch(context.TODO(), opts)
		},
	}
}
