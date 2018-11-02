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

	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descReplicaSetLabelsDefaultLabels = []string{"namespace", "replicaset"}
	descReplicaSetCreated             = metrics.NewMetricFamilyDef(
		"kube_replicaset_created",
		"Unix creation timestamp",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetStatusReplicas = metrics.NewMetricFamilyDef(
		"kube_replicaset_status_replicas",
		"The number of replicas per ReplicaSet.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetStatusFullyLabeledReplicas = metrics.NewMetricFamilyDef(
		"kube_replicaset_status_fully_labeled_replicas",
		"The number of fully labeled replicas per ReplicaSet.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetStatusReadyReplicas = metrics.NewMetricFamilyDef(
		"kube_replicaset_status_ready_replicas",
		"The number of ready replicas per ReplicaSet.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetStatusObservedGeneration = metrics.NewMetricFamilyDef(
		"kube_replicaset_status_observed_generation",
		"The generation observed by the ReplicaSet controller.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetSpecReplicas = metrics.NewMetricFamilyDef(
		"kube_replicaset_spec_replicas",
		"Number of desired pods for a ReplicaSet.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetMetadataGeneration = metrics.NewMetricFamilyDef(
		"kube_replicaset_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		descReplicaSetLabelsDefaultLabels,
		nil,
	)
	descReplicaSetOwner = metrics.NewMetricFamilyDef(
		"kube_replicaset_owner",
		"Information about the ReplicaSet's owner.",
		append(descReplicaSetLabelsDefaultLabels, "owner_kind", "owner_name", "owner_is_controller"),
		nil,
	)
)

func createReplicaSetListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.ExtensionsV1beta1().ReplicaSets(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.ExtensionsV1beta1().ReplicaSets(ns).Watch(opts)
		},
	}
}

func generateReplicaSetMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	rPointer := obj.(*v1beta1.ReplicaSet)
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
		addGauge(descReplicaSetCreated, float64(r.CreationTimestamp.Unix()))
	}

	owners := r.GetOwnerReferences()
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

	addGauge(descReplicaSetStatusReplicas, float64(r.Status.Replicas))
	addGauge(descReplicaSetStatusFullyLabeledReplicas, float64(r.Status.FullyLabeledReplicas))
	addGauge(descReplicaSetStatusReadyReplicas, float64(r.Status.ReadyReplicas))
	addGauge(descReplicaSetStatusObservedGeneration, float64(r.Status.ObservedGeneration))
	if r.Spec.Replicas != nil {
		addGauge(descReplicaSetSpecReplicas, float64(*r.Spec.Replicas))
	}
	addGauge(descReplicaSetMetadataGeneration, float64(r.ObjectMeta.Generation))

	return ms
}
