/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

var (
	descDaemonSetCurrentNumberScheduled = prometheus.NewDesc(
		"kube_daemonset_status_current_number_scheduled",
		"The number of nodes running at least one daemon pod and are supposed to.",
		[]string{"namespace", "daemonset"}, nil,
	)
	descDaemonSetNumberMisscheduled = prometheus.NewDesc(
		"kube_daemonset_status_number_misscheduled",
		"The number of nodes running a daemon pod but are not supposed to.",
		[]string{"namespace", "daemonset"}, nil,
	)
	descDaemonSetDesiredNumberScheduled = prometheus.NewDesc(
		"kube_daemonset_status_desired_number_scheduled",
		"The number of nodes that should be running the daemon pod.",
		[]string{"namespace", "daemonset"}, nil,
	)
	descDaemonSetNumberReady = prometheus.NewDesc(
		"kube_daemonset_status_number_ready",
		"The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and ready.",
		[]string{"namespace", "daemonset"}, nil,
	)
	descDaemonSetMetadataGeneration = prometheus.NewDesc(
		"kube_daemonset_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		[]string{"namespace", "daemonset"}, nil,
	)
)

type DaemonSetLister func() ([]v1beta1.DaemonSet, error)

func (l DaemonSetLister) List() ([]v1beta1.DaemonSet, error) {
	return l()
}

func RegisterDaemonSetCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface) {
	client := kubeClient.ExtensionsV1beta1().RESTClient()
	dslw := cache.NewListWatchFromClient(client, "daemonsets", api.NamespaceAll, nil)
	dsinf := cache.NewSharedInformer(dslw, &v1beta1.DaemonSet{}, resyncPeriod)

	dsLister := DaemonSetLister(func() (daemonsets []v1beta1.DaemonSet, err error) {
		for _, c := range dsinf.GetStore().List() {
			daemonsets = append(daemonsets, *(c.(*v1beta1.DaemonSet)))
		}
		return daemonsets, nil
	})

	registry.MustRegister(&daemonsetCollector{store: dsLister})
	go dsinf.Run(context.Background().Done())
}

type daemonsetStore interface {
	List() (daemonsets []v1beta1.DaemonSet, err error)
}

// daemonsetCollector collects metrics about all daemonsets in the cluster.
type daemonsetCollector struct {
	store daemonsetStore
}

// Describe implements the prometheus.Collector interface.
func (dc *daemonsetCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descDaemonSetCurrentNumberScheduled
	ch <- descDaemonSetNumberMisscheduled
	ch <- descDaemonSetDesiredNumberScheduled
	ch <- descDaemonSetNumberReady
	ch <- descDaemonSetMetadataGeneration
}

// Collect implements the prometheus.Collector interface.
func (dc *daemonsetCollector) Collect(ch chan<- prometheus.Metric) {
	dpls, err := dc.store.List()
	if err != nil {
		glog.Errorf("listing daemonsets failed: %s", err)
		return
	}
	for _, d := range dpls {
		dc.collectDaemonSet(ch, d)
	}
}

func (dc *daemonsetCollector) collectDaemonSet(ch chan<- prometheus.Metric, d v1beta1.DaemonSet) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{d.Namespace, d.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addGauge(descDaemonSetCurrentNumberScheduled, float64(d.Status.CurrentNumberScheduled))
	addGauge(descDaemonSetNumberMisscheduled, float64(d.Status.NumberMisscheduled))
	addGauge(descDaemonSetDesiredNumberScheduled, float64(d.Status.DesiredNumberScheduled))
	addGauge(descDaemonSetNumberReady, float64(d.Status.NumberReady))
	addGauge(descDaemonSetMetadataGeneration, float64(d.ObjectMeta.Generation))
}
