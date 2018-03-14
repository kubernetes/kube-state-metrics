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
	"context"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	descReplicationControllerCreated = prometheus.NewDesc(
		"kube_replicationcontroller_created",
		"Unix creation timestamp",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
	descReplicationControllerStatusReplicas = prometheus.NewDesc(
		"kube_replicationcontroller_status_replicas",
		"The number of replicas per ReplicationController.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
	descReplicationControllerStatusFullyLabeledReplicas = prometheus.NewDesc(
		"kube_replicationcontroller_status_fully_labeled_replicas",
		"The number of fully labeled replicas per ReplicationController.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
	descReplicationControllerStatusReadyReplicas = prometheus.NewDesc(
		"kube_replicationcontroller_status_ready_replicas",
		"The number of ready replicas per ReplicationController.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
	descReplicationControllerStatusAvailableReplicas = prometheus.NewDesc(
		"kube_replicationcontroller_status_available_replicas",
		"The number of available replicas per ReplicationController.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
	descReplicationControllerStatusObservedGeneration = prometheus.NewDesc(
		"kube_replicationcontroller_status_observed_generation",
		"The generation observed by the ReplicationController controller.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
	descReplicationControllerSpecReplicas = prometheus.NewDesc(
		"kube_replicationcontroller_spec_replicas",
		"Number of desired pods for a ReplicationController.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
	descReplicationControllerMetadataGeneration = prometheus.NewDesc(
		"kube_replicationcontroller_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		[]string{"namespace", "replicationcontroller"}, nil,
	)
)

type ReplicationControllerLister func() ([]v1.ReplicationController, error)

func (l ReplicationControllerLister) List() ([]v1.ReplicationController, error) {
	return l()
}

func RegisterReplicationControllerCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface, namespaces []string) {
	client := kubeClient.CoreV1().RESTClient()
	glog.Infof("collect replicationcontroller with %s", client.APIVersion())

	rcinfs := NewSharedInformerList(client, "replicationcontrollers", namespaces, &v1.ReplicationController{})

	replicationControllerLister := ReplicationControllerLister(func() (rcs []v1.ReplicationController, err error) {
		for _, rcinf := range *rcinfs {
			for _, c := range rcinf.GetStore().List() {
				rcs = append(rcs, *(c.(*v1.ReplicationController)))
			}
		}
		return rcs, nil
	})

	registry.MustRegister(&replicationcontrollerCollector{store: replicationControllerLister})
	rcinfs.Run(context.Background().Done())
}

type replicationcontrollerStore interface {
	List() (replicationcontrollers []v1.ReplicationController, err error)
}

type replicationcontrollerCollector struct {
	store replicationcontrollerStore
}

// Describe implements the prometheus.Collector interface.
func (dc *replicationcontrollerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descReplicationControllerCreated
	ch <- descReplicationControllerStatusReplicas
	ch <- descReplicationControllerStatusFullyLabeledReplicas
	ch <- descReplicationControllerStatusReadyReplicas
	ch <- descReplicationControllerStatusAvailableReplicas
	ch <- descReplicationControllerStatusObservedGeneration
	ch <- descReplicationControllerSpecReplicas
	ch <- descReplicationControllerMetadataGeneration
}

// Collect implements the prometheus.Collector interface.
func (dc *replicationcontrollerCollector) Collect(ch chan<- prometheus.Metric) {
	rcs, err := dc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "replicationcontroller"}).Inc()
		glog.Errorf("listing replicationcontrollers failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "replicationcontroller"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "replicationcontroller"}).Observe(float64(len(rcs)))
	for _, d := range rcs {
		dc.collectReplicationController(ch, d)
	}

	glog.V(4).Infof("collected %d replicationcontrollers", len(rcs))
}

func (dc *replicationcontrollerCollector) collectReplicationController(ch chan<- prometheus.Metric, d v1.ReplicationController) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{d.Namespace, d.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	if !d.CreationTimestamp.IsZero() {
		addGauge(descReplicationControllerCreated, float64(d.CreationTimestamp.Unix()))
	}
	addGauge(descReplicationControllerStatusReplicas, float64(d.Status.Replicas))
	addGauge(descReplicationControllerStatusFullyLabeledReplicas, float64(d.Status.FullyLabeledReplicas))
	addGauge(descReplicationControllerStatusReadyReplicas, float64(d.Status.ReadyReplicas))
	addGauge(descReplicationControllerStatusAvailableReplicas, float64(d.Status.AvailableReplicas))
	addGauge(descReplicationControllerStatusObservedGeneration, float64(d.Status.ObservedGeneration))
	if d.Spec.Replicas != nil {
		addGauge(descReplicationControllerSpecReplicas, float64(*d.Spec.Replicas))
	}
	addGauge(descReplicationControllerMetadataGeneration, float64(d.ObjectMeta.Generation))
}
