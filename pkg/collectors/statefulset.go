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
	"k8s.io/api/apps/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	descStatefulSetLabelsName          = "kube_statefulset_labels"
	descStatefulSetLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descStatefulSetLabelsDefaultLabels = []string{"namespace", "statefulset"}

	descStatefulSetCreated = prometheus.NewDesc(
		"kube_statefulset_created",
		"Unix creation timestamp",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetStatusReplicas = prometheus.NewDesc(
		"kube_statefulset_status_replicas",
		"The number of replicas per StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetStatusReplicasCurrent = prometheus.NewDesc(
		"kube_statefulset_status_replicas_current",
		"The number of current replicas per StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetStatusReplicasReady = prometheus.NewDesc(
		"kube_statefulset_status_replicas_ready",
		"The number of ready replicas per StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetStatusReplicasUpdated = prometheus.NewDesc(
		"kube_statefulset_status_replicas_updated",
		"The number of updated replicas per StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetStatusObservedGeneration = prometheus.NewDesc(
		"kube_statefulset_status_observed_generation",
		"The generation observed by the StatefulSet controller.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetSpecReplicas = prometheus.NewDesc(
		"kube_statefulset_replicas",
		"Number of desired pods for a StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetMetadataGeneration = prometheus.NewDesc(
		"kube_statefulset_metadata_generation",
		"Sequence number representing a specific generation of the desired state for the StatefulSet.",
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
	descStatefulSetLabels = prometheus.NewDesc(
		descStatefulSetLabelsName,
		descStatefulSetLabelsHelp,
		descStatefulSetLabelsDefaultLabels,
		nil,
	)
)

type StatefulSetLister func() ([]v1beta1.StatefulSet, error)

func (l StatefulSetLister) List() ([]v1beta1.StatefulSet, error) {
	return l()
}

func RegisterStatefulSetCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Apps().V1beta1().StatefulSets().Informer().(cache.SharedInformer))
	}

	statefulSetLister := StatefulSetLister(func() (statefulSets []v1beta1.StatefulSet, err error) {
		for _, dinf := range infs {
			for _, c := range dinf.GetStore().List() {
				statefulSets = append(statefulSets, *(c.(*v1beta1.StatefulSet)))
			}
		}
		return statefulSets, nil
	})

	registry.MustRegister(&statefulSetCollector{store: statefulSetLister, opts: opts})
	infs.Run(context.Background().Done())
}

type statefulSetStore interface {
	List() (statefulSets []v1beta1.StatefulSet, err error)
}

type statefulSetCollector struct {
	store statefulSetStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (dc *statefulSetCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descStatefulSetCreated
	ch <- descStatefulSetStatusReplicas
	ch <- descStatefulSetStatusReplicasCurrent
	ch <- descStatefulSetStatusReplicasReady
	ch <- descStatefulSetStatusReplicasUpdated
	ch <- descStatefulSetStatusObservedGeneration
	ch <- descStatefulSetSpecReplicas
	ch <- descStatefulSetMetadataGeneration
	ch <- descStatefulSetLabels
}

// Collect implements the prometheus.Collector interface.
func (sc *statefulSetCollector) Collect(ch chan<- prometheus.Metric) {
	sss, err := sc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "statefulset"}).Inc()
		glog.Errorf("listing statefulsets failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "statefulset"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "statefulset"}).Observe(float64(len(sss)))
	for _, d := range sss {
		sc.collectStatefulSet(ch, d)
	}

	glog.V(4).Infof("collected %d statefulsets", len(sss))
}

func statefulSetLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descStatefulSetLabelsName,
		descStatefulSetLabelsHelp,
		append(descStatefulSetLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func (dc *statefulSetCollector) collectStatefulSet(ch chan<- prometheus.Metric, statefulSet v1beta1.StatefulSet) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{statefulSet.Namespace, statefulSet.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	if !statefulSet.CreationTimestamp.IsZero() {
		addGauge(descStatefulSetCreated, float64(statefulSet.CreationTimestamp.Unix()))
	}
	addGauge(descStatefulSetStatusReplicas, float64(statefulSet.Status.Replicas))
	addGauge(descStatefulSetStatusReplicasCurrent, float64(statefulSet.Status.CurrentReplicas))
	addGauge(descStatefulSetStatusReplicasReady, float64(statefulSet.Status.ReadyReplicas))
	addGauge(descStatefulSetStatusReplicasUpdated, float64(statefulSet.Status.UpdatedReplicas))
	if statefulSet.Status.ObservedGeneration != nil {
		addGauge(descStatefulSetStatusObservedGeneration, float64(*statefulSet.Status.ObservedGeneration))
	}

	if statefulSet.Spec.Replicas != nil {
		addGauge(descStatefulSetSpecReplicas, float64(*statefulSet.Spec.Replicas))
	}
	addGauge(descStatefulSetMetadataGeneration, float64(statefulSet.ObjectMeta.Generation))

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(statefulSet.Labels)
	addGauge(statefulSetLabelsDesc(labelKeys), 1, labelValues...)
}
