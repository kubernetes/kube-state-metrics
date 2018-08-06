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
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	descDaemonSetLabelsName          = "kube_daemonset_labels"
	descDaemonSetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descDaemonSetLabelsDefaultLabels = []string{"namespace", "daemonset"}

	descDaemonSetCreated = prometheus.NewDesc(
		"kube_daemonset_created",
		"Unix creation timestamp",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetCurrentNumberScheduled = prometheus.NewDesc(
		"kube_daemonset_status_current_number_scheduled",
		"The number of nodes running at least one daemon pod and are supposed to.",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetDesiredNumberScheduled = prometheus.NewDesc(
		"kube_daemonset_status_desired_number_scheduled",
		"The number of nodes that should be running the daemon pod.",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetNumberAvailable = prometheus.NewDesc(
		"kube_daemonset_status_number_available",
		"The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and available",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetNumberMisscheduled = prometheus.NewDesc(
		"kube_daemonset_status_number_misscheduled",
		"The number of nodes running a daemon pod but are not supposed to.",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetNumberReady = prometheus.NewDesc(
		"kube_daemonset_status_number_ready",
		"The number of nodes that should be running the daemon pod and have one or more of the daemon pod running and ready.",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetNumberUnavailable = prometheus.NewDesc(
		"kube_daemonset_status_number_unavailable",
		"The number of nodes that should be running the daemon pod and have none of the daemon pod running and available",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetUpdatedNumberScheduled = prometheus.NewDesc(
		"kube_daemonset_updated_number_scheduled",
		"The total number of nodes that are running updated daemon pod",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetMetadataGeneration = prometheus.NewDesc(
		"kube_daemonset_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
	descDaemonSetLabels = prometheus.NewDesc(
		descDaemonSetLabelsName,
		descDaemonSetLabelsHelp,
		descDaemonSetLabelsDefaultLabels,
		nil,
	)
)

type DaemonSetLister func() ([]v1beta1.DaemonSet, error)

func (l DaemonSetLister) List() ([]v1beta1.DaemonSet, error) {
	return l()
}

func RegisterDaemonSetCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Extensions().V1beta1().DaemonSets().Informer().(cache.SharedInformer))
	}

	dsLister := DaemonSetLister(func() (daemonsets []v1beta1.DaemonSet, err error) {
		for _, dsinf := range infs {
			for _, c := range dsinf.GetStore().List() {
				daemonsets = append(daemonsets, *(c.(*v1beta1.DaemonSet)))
			}
		}
		return daemonsets, nil
	})

	registry.MustRegister(&daemonsetCollector{store: dsLister, opts: opts})
	infs.Run(context.Background().Done())
}

type daemonsetStore interface {
	List() (daemonsets []v1beta1.DaemonSet, err error)
}

// daemonsetCollector collects metrics about all daemonsets in the cluster.
type daemonsetCollector struct {
	store daemonsetStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (dc *daemonsetCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descDaemonSetCreated
	ch <- descDaemonSetCurrentNumberScheduled
	ch <- descDaemonSetNumberAvailable
	ch <- descDaemonSetNumberMisscheduled
	ch <- descDaemonSetNumberUnavailable
	ch <- descDaemonSetDesiredNumberScheduled
	ch <- descDaemonSetNumberReady
	ch <- descDaemonSetUpdatedNumberScheduled
	ch <- descDaemonSetMetadataGeneration
	ch <- descDaemonSetLabels
}

// Collect implements the prometheus.Collector interface.
func (dc *daemonsetCollector) Collect(ch chan<- prometheus.Metric) {
	dss, err := dc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "daemonset"}).Inc()
		glog.Errorf("listing daemonsets failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "daemonset"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "daemonset"}).Observe(float64(len(dss)))
	for _, d := range dss {
		dc.collectDaemonSet(ch, d)
	}

	glog.V(4).Infof("collected %d daemonsets", len(dss))
}

func DaemonSetLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descDaemonSetLabelsName,
		descDaemonSetLabelsHelp,
		append(descDaemonSetLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func (dc *daemonsetCollector) collectDaemonSet(ch chan<- prometheus.Metric, d v1beta1.DaemonSet) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{d.Namespace, d.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	if !d.CreationTimestamp.IsZero() {
		addGauge(descDaemonSetCreated, float64(d.CreationTimestamp.Unix()))
	}
	addGauge(descDaemonSetCurrentNumberScheduled, float64(d.Status.CurrentNumberScheduled))
	addGauge(descDaemonSetNumberAvailable, float64(d.Status.NumberAvailable))
	addGauge(descDaemonSetNumberUnavailable, float64(d.Status.NumberUnavailable))
	addGauge(descDaemonSetNumberMisscheduled, float64(d.Status.NumberMisscheduled))
	addGauge(descDaemonSetDesiredNumberScheduled, float64(d.Status.DesiredNumberScheduled))
	addGauge(descDaemonSetNumberReady, float64(d.Status.NumberReady))
	addGauge(descDaemonSetUpdatedNumberScheduled, float64(d.Status.UpdatedNumberScheduled))
	addGauge(descDaemonSetMetadataGeneration, float64(d.ObjectMeta.Generation))

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(d.ObjectMeta.Labels)
	addGauge(DaemonSetLabelsDesc(labelKeys), 1, labelValues...)
}
