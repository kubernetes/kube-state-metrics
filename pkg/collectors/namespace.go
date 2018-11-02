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
	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descNamespaceLabelsName          = "kube_namespace_labels"
	descNamespaceLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descNamespaceLabelsDefaultLabels = []string{"namespace"}

	descNamespaceAnnotationsName          = "kube_namespace_annotations"
	descNamespaceAnnotationsHelp          = "Kubernetes annotations converted to Prometheus labels."
	descNamespaceAnnotationsDefaultLabels = []string{"namespace"}

	descNamespaceCreated = metrics.NewMetricFamilyDef(
		"kube_namespace_created",
		"Unix creation timestamp",
		descNamespaceLabelsDefaultLabels,
		nil,
	)
	descNamespaceLabels = metrics.NewMetricFamilyDef(
		descNamespaceLabelsName,
		descNamespaceLabelsHelp,
		descNamespaceLabelsDefaultLabels,
		nil,
	)
	descNamespaceAnnotations = metrics.NewMetricFamilyDef(
		descNamespaceAnnotationsName,
		descNamespaceAnnotationsHelp,
		descNamespaceAnnotationsDefaultLabels,
		nil,
	)
	descNamespacePhase = metrics.NewMetricFamilyDef(
		"kube_namespace_status_phase",
		"kubernetes namespace status phase.",
		append(descNamespaceLabelsDefaultLabels, "phase"),
		nil,
	)
)

func createNamespaceListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.CoreV1().Namespaces().List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.CoreV1().Namespaces().Watch(opts)
		},
	}
}

func generateNamespaceMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	nPointer := obj.(*v1.Namespace)
	n := *nPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{n.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}

	addGauge(descNamespacePhase, boolFloat64(n.Status.Phase == v1.NamespaceActive), string(v1.NamespaceActive))
	addGauge(descNamespacePhase, boolFloat64(n.Status.Phase == v1.NamespaceTerminating), string(v1.NamespaceTerminating))

	if !n.CreationTimestamp.IsZero() {
		addGauge(descNamespaceCreated, float64(n.CreationTimestamp.Unix()))
	}

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(n.Labels)
	addGauge(namespaceLabelsDesc(labelKeys), 1, labelValues...)

	annnotationKeys, annotationValues := kubeAnnotationsToPrometheusAnnotations(n.Annotations)
	addGauge(namespaceAnnotationsDesc(annnotationKeys), 1, annotationValues...)

	return ms
}

func namespaceLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descNamespaceLabelsName,
		descNamespaceLabelsHelp,
		append(descNamespaceLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func namespaceAnnotationsDesc(annotationKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descNamespaceAnnotationsName,
		descNamespaceAnnotationsHelp,
		append(descNamespaceAnnotationsDefaultLabels, annotationKeys...),
		nil,
	)
}
