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
	descPersistentVolumeLabelsName          = "kube_persistentvolume_labels"
	descPersistentVolumeLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descPersistentVolumeLabelsDefaultLabels = []string{"persistentvolume"}

	descPersistentVolumeLabels = prometheus.NewDesc(
		descPersistentVolumeLabelsName,
		descPersistentVolumeLabelsHelp,
		descPersistentVolumeLabelsDefaultLabels,
		nil,
	)
	descPersistentVolumeStatusPhase = prometheus.NewDesc(
		"kube_persistentvolume_status_phase",
		"The phase indicates if a volume is available, bound to a claim, or released by a claim.",
		append(descPersistentVolumeLabelsDefaultLabels, "phase"),
		nil,
	)
	descPersistentVolumeInfo = prometheus.NewDesc(
		"kube_persistentvolume_info",
		"Information about persistentvolume.",
		append(descPersistentVolumeLabelsDefaultLabels, "storageclass"),
		nil,
	)
)

type PersistentVolumeLister func() (v1.PersistentVolumeList, error)

func (pvl PersistentVolumeLister) List() (v1.PersistentVolumeList, error) {
	return pvl()
}

func RegisterPersistentVolumeCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Core().V1().PersistentVolumes().Informer().(cache.SharedInformer))
	}

	persistentVolumeLister := PersistentVolumeLister(func() (pvs v1.PersistentVolumeList, err error) {
		for _, pvinf := range infs {
			for _, pv := range pvinf.GetStore().List() {
				pvs.Items = append(pvs.Items, *(pv.(*v1.PersistentVolume)))
			}
		}
		return pvs, nil
	})

	registry.MustRegister(&persistentVolumeCollector{store: persistentVolumeLister, opts: opts})
	infs.Run(context.Background().Done())
}

type persistentVolumeStore interface {
	List() (v1.PersistentVolumeList, error)
}

// persistentVolumeCollector collects metrics about all persistentVolumes in the cluster.
type persistentVolumeCollector struct {
	store persistentVolumeStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (collector *persistentVolumeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descPersistentVolumeStatusPhase
	ch <- descPersistentVolumeInfo
	ch <- descPersistentVolumeLabels
}

func persistentVolumeLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descPersistentVolumeLabelsName,
		descPersistentVolumeLabelsHelp,
		append(descPersistentVolumeLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

// Collect implements the prometheus.Collector interface.
func (collector *persistentVolumeCollector) Collect(ch chan<- prometheus.Metric) {
	persistentVolumeCollector, err := collector.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "persistentvolume"}).Inc()
		glog.Errorf("listing persistentVolume failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "persistentvolume"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "persistentvolume"}).Observe(float64(len(persistentVolumeCollector.Items)))
	for _, pv := range persistentVolumeCollector.Items {
		collector.collectPersistentVolume(ch, pv)
	}

	glog.V(4).Infof("collected %d persistentvolumes", len(persistentVolumeCollector.Items))
}

func (collector *persistentVolumeCollector) collectPersistentVolume(ch chan<- prometheus.Metric, pv v1.PersistentVolume) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{pv.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(pv.Labels)
	addGauge(persistentVolumeLabelsDesc(labelKeys), 1, labelValues...)

	addGauge(descPersistentVolumeInfo, 1, pv.Spec.StorageClassName)
	// Set current phase to 1, others to 0 if it is set.
	if p := pv.Status.Phase; p != "" {
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == v1.VolumePending), string(v1.VolumePending))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == v1.VolumeAvailable), string(v1.VolumeAvailable))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == v1.VolumeBound), string(v1.VolumeBound))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == v1.VolumeReleased), string(v1.VolumeReleased))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == v1.VolumeFailed), string(v1.VolumeFailed))
	}
}
