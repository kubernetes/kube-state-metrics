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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/tools/cache"
)

var (
	descPersistentVolumeStatusPhase = prometheus.NewDesc(
		"kube_persistentvolume_status_phase",
		"The phase indicates if a volume is available, bound to a claim, or released by a claim.",
		[]string{
			"namespace",
			"persistentvolume",
			"phase",
		}, nil,
	)
)

type PersistentVolumeLister func() (api.PersistentVolumeList, error)

func (l PersistentVolumeLister) List() (api.PersistentVolumeList, error) {
	return l()
}

func RegisterPersistentVolumeCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface) {
	client := kubeClient.CoreV1().RESTClient()
	pvlw := cache.NewListWatchFromClient(client, "persistentvolumes", api.NamespaceAll, nil)
	pvinf := cache.NewSharedInformer(pvlw, &api.PersistentVolume{}, resyncPeriod)

	persistentVolumeLister := PersistentVolumeLister(func() (pvs api.PersistentVolumeList, err error) {
		for _, pv := range pvinf.GetStore().List() {
			pvs.Items = append(pvs.Items, *(pv.(*api.PersistentVolume)))
		}
		return pvs, nil
	})

	registry.MustRegister(&persistentVolumeCollector{store: persistentVolumeLister})
	go pvinf.Run(context.Background().Done())
}

type persistentVolumeStore interface {
	List() (api.PersistentVolumeList, error)
}

// persistentVolumeCollector collects metrics about all persistentVolumes in the cluster.
type persistentVolumeCollector struct {
	store persistentVolumeStore
}

// Describe implements the prometheus.Collector interface.
func (collector *persistentVolumeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descPersistentVolumeStatusPhase
}

// Collect implements the prometheus.Collector interface.
func (collector *persistentVolumeCollector) Collect(ch chan<- prometheus.Metric) {
	persistentVolumeCollector, err := collector.store.List()
	if err != nil {
		glog.Errorf("listing persistentVolume failed: %s", err)
		return
	}

	for _, pvc := range persistentVolumeCollector.Items {
		collector.collectPersistentVolume(ch, pvc)
	}
}

func (collector *persistentVolumeCollector) collectPersistentVolume(ch chan<- prometheus.Metric, pv api.PersistentVolume) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{pv.Namespace, pv.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	// Set current phase to 1, others to 0 if it is set.
	if p := pv.Status.Phase; p != "" {
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == api.VolumePending), string(api.VolumePending))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == api.VolumeAvailable), string(api.VolumeAvailable))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == api.VolumeBound), string(api.VolumeBound))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == api.VolumeReleased), string(api.VolumeReleased))
		addGauge(descPersistentVolumeStatusPhase, boolFloat64(p == api.VolumeFailed), string(api.VolumeFailed))
	}
}
