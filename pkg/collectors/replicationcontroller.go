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
	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descReplicationControllerLabelsDefaultLabels = []string{"namespace", "replicationcontroller"}

	descReplicationControllerCreated = metrics.NewMetricFamilyDef(
		"kube_replicationcontroller_created",
		"Unix creation timestamp",
		descReplicationControllerLabelsDefaultLabels,
		nil,
	)
	descReplicationControllerStatusReplicas = metrics.NewMetricFamilyDef(
		"kube_replicationcontroller_status_replicas",
		"The number of replicas per ReplicationController.",
		descReplicationControllerLabelsDefaultLabels,
		nil,
	)
	descReplicationControllerStatusFullyLabeledReplicas = metrics.NewMetricFamilyDef(
		"kube_replicationcontroller_status_fully_labeled_replicas",
		"The number of fully labeled replicas per ReplicationController.",
		descReplicationControllerLabelsDefaultLabels,
		nil,
	)
	descReplicationControllerStatusReadyReplicas = metrics.NewMetricFamilyDef(
		"kube_replicationcontroller_status_ready_replicas",
		"The number of ready replicas per ReplicationController.",
		descReplicationControllerLabelsDefaultLabels,
		nil,
	)
	descReplicationControllerStatusAvailableReplicas = metrics.NewMetricFamilyDef(
		"kube_replicationcontroller_status_available_replicas",
		"The number of available replicas per ReplicationController.",
		descReplicationControllerLabelsDefaultLabels,
		nil,
	)
	descReplicationControllerStatusObservedGeneration = metrics.NewMetricFamilyDef(
		"kube_replicationcontroller_status_observed_generation",
		"The generation observed by the ReplicationController controller.",
		descReplicationControllerLabelsDefaultLabels,
		nil,
	)
	descReplicationControllerSpecReplicas = metrics.NewMetricFamilyDef(
		"kube_replicationcontroller_spec_replicas",
		"Number of desired pods for a ReplicationController.",
		descReplicationControllerLabelsDefaultLabels,
		nil,
	)
	descReplicationControllerMetadataGeneration = metrics.NewMetricFamilyDef(
		"kube_replicationcontroller_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		descReplicationControllerLabelsDefaultLabels,
		nil,
	)
)

func createReplicationControllerListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().ReplicationControllers(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().ReplicationControllers(ns).Watch(opts)
		},
	}
}
func generateReplicationControllerMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	rPointer := obj.(*v1.ReplicationController)
	r := *rPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{r.Namespace, r.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}
	if !r.CreationTimestamp.IsZero() {
		addGauge(descReplicationControllerCreated, float64(r.CreationTimestamp.Unix()))
	}
	addGauge(descReplicationControllerStatusReplicas, float64(r.Status.Replicas))
	addGauge(descReplicationControllerStatusFullyLabeledReplicas, float64(r.Status.FullyLabeledReplicas))
	addGauge(descReplicationControllerStatusReadyReplicas, float64(r.Status.ReadyReplicas))
	addGauge(descReplicationControllerStatusAvailableReplicas, float64(r.Status.AvailableReplicas))
	addGauge(descReplicationControllerStatusObservedGeneration, float64(r.Status.ObservedGeneration))
	if r.Spec.Replicas != nil {
		addGauge(descReplicationControllerSpecReplicas, float64(*r.Spec.Replicas))
	}
	addGauge(descReplicationControllerMetadataGeneration, float64(r.ObjectMeta.Generation))

	return ms
}
