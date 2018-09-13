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

package collectors

import (
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	descPersistentVolumeClaimLabelsName          = "kube_persistentvolumeclaim_labels"
	descPersistentVolumeClaimLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descPersistentVolumeClaimLabelsDefaultLabels = []string{"namespace", "persistentvolumeclaim"}

	descPersistentVolumeClaimLabels = prometheus.NewDesc(
		descPersistentVolumeClaimLabelsName,
		descPersistentVolumeClaimLabelsHelp,
		descPersistentVolumeClaimLabelsDefaultLabels,
		nil,
	)
	descPersistentVolumeClaimInfo = prometheus.NewDesc(
		"kube_persistentvolumeclaim_info",
		"Information about persistent volume claim.",
		append(descPersistentVolumeClaimLabelsDefaultLabels, "storageclass", "volumename"),
		nil,
	)
	descPersistentVolumeClaimStatusPhase = prometheus.NewDesc(
		"kube_persistentvolumeclaim_status_phase",
		"The phase the persistent volume claim is currently in.",
		append(descPersistentVolumeClaimLabelsDefaultLabels, "phase"),
		nil,
	)
	descPersistentVolumeClaimResourceRequestsStorage = prometheus.NewDesc(
		"kube_persistentvolumeclaim_resource_requests_storage_bytes",
		"The capacity of storage requested by the persistent volume claim.",
		descPersistentVolumeClaimLabelsDefaultLabels,
		nil,
	)
)

type PersistentVolumeClaimLister func() (v1.PersistentVolumeClaimList, error)

func (l PersistentVolumeClaimLister) List() (v1.PersistentVolumeClaimList, error) {
	return l()
}

func RegisterPersistentVolumeClaimCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Core().V1().PersistentVolumeClaims().Informer().(cache.SharedInformer))
	}

	persistentVolumeClaimLister := PersistentVolumeClaimLister(func() (pvcs v1.PersistentVolumeClaimList, err error) {
		for _, pvcinf := range infs {
			for _, pvc := range pvcinf.GetStore().List() {
				pvcs.Items = append(pvcs.Items, *(pvc.(*v1.PersistentVolumeClaim)))
			}
		}
		return pvcs, nil
	})

	registry.MustRegister(&persistentVolumeClaimCollector{store: persistentVolumeClaimLister, opts: opts})
	infs.Run(context.Background().Done())
}

type persistentVolumeClaimStore interface {
	List() (v1.PersistentVolumeClaimList, error)
}

// persistentVolumeClaimCollector collects metrics about all persistentVolumeClaims in the cluster.
type persistentVolumeClaimCollector struct {
	store persistentVolumeClaimStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (collector *persistentVolumeClaimCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descPersistentVolumeClaimLabels
	ch <- descPersistentVolumeClaimInfo
	ch <- descPersistentVolumeClaimStatusPhase
	ch <- descPersistentVolumeClaimResourceRequestsStorage
}

func persistentVolumeClaimLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descPersistentVolumeClaimLabelsName,
		descPersistentVolumeClaimLabelsHelp,
		append(descPersistentVolumeClaimLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

// Collect implements the prometheus.Collector interface.
func (collector *persistentVolumeClaimCollector) Collect(ch chan<- prometheus.Metric) {
	persistentVolumeClaimCollector, err := collector.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "persistentvolumeclaim"}).Inc()
		glog.Errorf("listing persistent volume claims failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "persistentvolumeclaim"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "persistentvolumeclaim"}).Observe(float64(len(persistentVolumeClaimCollector.Items)))
	for _, pvc := range persistentVolumeClaimCollector.Items {
		collector.collectPersistentVolumeClaim(ch, pvc)
	}

	glog.V(4).Infof("collected %d persistentvolumeclaims", len(persistentVolumeClaimCollector.Items))
}

// getPersistentVolumeClaimClass returns StorageClassName. If no storage class was
// requested, it returns "".
func getPersistentVolumeClaimClass(claim *v1.PersistentVolumeClaim) string {
	// Use beta annotation first
	if class, found := claim.Annotations[v1.BetaStorageClassAnnotation]; found {
		return class
	}

	if claim.Spec.StorageClassName != nil {
		return *claim.Spec.StorageClassName
	}

	// Special non-empty string to indicate absence of storage class.
	return "<none>"
}

func (collector *persistentVolumeClaimCollector) collectPersistentVolumeClaim(ch chan<- prometheus.Metric, pvc v1.PersistentVolumeClaim) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{pvc.Namespace, pvc.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(pvc.Labels)
	addGauge(persistentVolumeClaimLabelsDesc(labelKeys), 1, labelValues...)

	storageClassName := getPersistentVolumeClaimClass(&pvc)
	volumeName := pvc.Spec.VolumeName
	addGauge(descPersistentVolumeClaimInfo, 1, storageClassName, volumeName)

	// Set current phase to 1, others to 0 if it is set.
	if p := pvc.Status.Phase; p != "" {
		addGauge(descPersistentVolumeClaimStatusPhase, boolFloat64(p == v1.ClaimLost), string(v1.ClaimLost))
		addGauge(descPersistentVolumeClaimStatusPhase, boolFloat64(p == v1.ClaimBound), string(v1.ClaimBound))
		addGauge(descPersistentVolumeClaimStatusPhase, boolFloat64(p == v1.ClaimPending), string(v1.ClaimPending))
	}

	if storage, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]; ok {
		addGauge(descPersistentVolumeClaimResourceRequestsStorage, float64(storage.Value()))
	}
}
