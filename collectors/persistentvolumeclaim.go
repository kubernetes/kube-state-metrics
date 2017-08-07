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
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	descPersistentVolumeClaimStatusPhase = prometheus.NewDesc(
		"kube_persistentvolumeclaim_status_phase",
		"The phase the claim is currently in.",
		[]string{
			"namespace",
			"persistentvolumeclaim",
			"phase",
		}, nil,
	)
)

type PersistentVolumeClaimLister func() (v1.PersistentVolumeClaimList, error)

func (l PersistentVolumeClaimLister) List() (v1.PersistentVolumeClaimList, error) {
	return l()
}

func RegisterPersistentVolumeClaimCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface) {
	client := kubeClient.CoreV1().RESTClient()
	pvclw := cache.NewListWatchFromClient(client, "persistentvolumeclaims", api.NamespaceAll, nil)
	pvcinf := cache.NewSharedInformer(pvclw, &v1.PersistentVolumeClaim{}, resyncPeriod)

	persistentVolumeClaimLister := PersistentVolumeClaimLister(func() (pvcs v1.PersistentVolumeClaimList, err error) {
		for _, pvc := range pvcinf.GetStore().List() {
			pvcs.Items = append(pvcs.Items, *(pvc.(*v1.PersistentVolumeClaim)))
		}
		return pvcs, nil
	})

	registry.MustRegister(&persistentVolumeClaimCollector{store: persistentVolumeClaimLister})
	go pvcinf.Run(context.Background().Done())
}

type persistentVolumeClaimStore interface {
	List() (v1.PersistentVolumeClaimList, error)
}

// persistentVolumeClaimCollector collects metrics about all persistentVolumeClaims in the cluster.
type persistentVolumeClaimCollector struct {
	store persistentVolumeClaimStore
}

// Describe implements the prometheus.Collector interface.
func (collector *persistentVolumeClaimCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descPersistentVolumeClaimStatusPhase
}

// Collect implements the prometheus.Collector interface.
func (collector *persistentVolumeClaimCollector) Collect(ch chan<- prometheus.Metric) {
	persistentVolumeClaimCollector, err := collector.store.List()
	if err != nil {
		glog.Errorf("listing limit ranges failed: %s", err)
		return
	}

	for _, pvc := range persistentVolumeClaimCollector.Items {
		collector.collectPersistentVolumeClaim(ch, pvc)
	}
}

func (collector *persistentVolumeClaimCollector) collectPersistentVolumeClaim(ch chan<- prometheus.Metric, pvc v1.PersistentVolumeClaim) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{pvc.Namespace, pvc.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	// Set current phase to 1, others to 0 if it is set.
	if p := pvc.Status.Phase; p != "" {
		addGauge(descPersistentVolumeClaimStatusPhase, boolFloat64(p == v1.ClaimLost), string(v1.ClaimLost))
		addGauge(descPersistentVolumeClaimStatusPhase, boolFloat64(p == v1.ClaimBound), string(v1.ClaimBound))
		addGauge(descPersistentVolumeClaimStatusPhase, boolFloat64(p == v1.ClaimPending), string(v1.ClaimPending))
	}
}
