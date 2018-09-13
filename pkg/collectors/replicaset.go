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
	"strconv"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	descReplicaSetLabelsDefaultLabels = []string{"namespace", "replicaset"}
	descReplicaSetCreated             = prometheus.NewDesc(
		"kube_replicaset_created",
		"Unix creation timestamp",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetStatusReplicas = prometheus.NewDesc(
		"kube_replicaset_status_replicas",
		"The number of replicas per ReplicaSet.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetStatusFullyLabeledReplicas = prometheus.NewDesc(
		"kube_replicaset_status_fully_labeled_replicas",
		"The number of fully labeled replicas per ReplicaSet.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetStatusReadyReplicas = prometheus.NewDesc(
		"kube_replicaset_status_ready_replicas",
		"The number of ready replicas per ReplicaSet.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetStatusObservedGeneration = prometheus.NewDesc(
		"kube_replicaset_status_observed_generation",
		"The generation observed by the ReplicaSet controller.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetSpecReplicas = prometheus.NewDesc(
		"kube_replicaset_spec_replicas",
		"Number of desired pods for a ReplicaSet.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetMetadataGeneration = prometheus.NewDesc(
		"kube_replicaset_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetOwner = prometheus.NewDesc(
		"kube_replicaset_owner",
		"Information about the ReplicaSet's owner.",
		append(descReplicaSetLabelsDefaultLabels, "owner_kind", "owner_name", "owner_is_controller"),
		nil,
	)
)

type ReplicaSetLister func() ([]v1beta1.ReplicaSet, error)

func (l ReplicaSetLister) List() ([]v1beta1.ReplicaSet, error) {
	return l()
}

func RegisterReplicaSetCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Extensions().V1beta1().ReplicaSets().Informer().(cache.SharedInformer))
	}

	replicaSetLister := ReplicaSetLister(func() (replicasets []v1beta1.ReplicaSet, err error) {
		for _, rsinf := range infs {
			for _, c := range rsinf.GetStore().List() {
				replicasets = append(replicasets, *(c.(*v1beta1.ReplicaSet)))
			}
		}
		return replicasets, nil
	})

	registry.MustRegister(&replicasetCollector{store: replicaSetLister, opts: opts})
	infs.Run(context.Background().Done())
}

type replicasetStore interface {
	List() (replicasets []v1beta1.ReplicaSet, err error)
}

// replicasetCollector collects metrics about all replicasets in the cluster.
type replicasetCollector struct {
	store replicasetStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (rsc *replicasetCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descReplicaSetCreated
	ch <- descReplicaSetStatusReplicas
	ch <- descReplicaSetStatusFullyLabeledReplicas
	ch <- descReplicaSetStatusReadyReplicas
	ch <- descReplicaSetStatusObservedGeneration
	ch <- descReplicaSetSpecReplicas
	ch <- descReplicaSetMetadataGeneration
	ch <- descReplicaSetOwner
}

// Collect implements the prometheus.Collector interface.
func (rsc *replicasetCollector) Collect(ch chan<- prometheus.Metric) {
	rss, err := rsc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "replicaset"}).Inc()
		glog.Errorf("listing replicasets failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "replicaset"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "replicaset"}).Observe(float64(len(rss)))
	for _, d := range rss {
		rsc.collectReplicaSet(ch, d)
	}

	glog.V(4).Infof("collected %d replicasets", len(rss))
}

func (rsc *replicasetCollector) collectReplicaSet(ch chan<- prometheus.Metric, d v1beta1.ReplicaSet) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{d.Namespace, d.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	if !d.CreationTimestamp.IsZero() {
		addGauge(descReplicaSetCreated, float64(d.CreationTimestamp.Unix()))
	}

	owners := d.GetOwnerReferences()
	if len(owners) == 0 {
		addGauge(descReplicaSetOwner, 1, "<none>", "<none>", "<none>")
	} else {
		for _, owner := range owners {
			if owner.Controller != nil {
				addGauge(descReplicaSetOwner, 1, owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller))
			} else {
				addGauge(descReplicaSetOwner, 1, owner.Kind, owner.Name, "false")
			}
		}
	}

	addGauge(descReplicaSetStatusReplicas, float64(d.Status.Replicas))
	addGauge(descReplicaSetStatusFullyLabeledReplicas, float64(d.Status.FullyLabeledReplicas))
	addGauge(descReplicaSetStatusReadyReplicas, float64(d.Status.ReadyReplicas))
	addGauge(descReplicaSetStatusObservedGeneration, float64(d.Status.ObservedGeneration))
	if d.Spec.Replicas != nil {
		addGauge(descReplicaSetSpecReplicas, float64(*d.Spec.Replicas))
	}
	addGauge(descReplicaSetMetadataGeneration, float64(d.ObjectMeta.Generation))
}
