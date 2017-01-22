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

package main

import (
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	autoscaling "k8s.io/client-go/1.5/pkg/apis/autoscaling/v1"
)

var (
	descHorizontalPodAutoscalerMetadataGeneration = prometheus.NewDesc(
		"kube_hpa_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		[]string{"namespace", "hpa"}, nil,
	)
	descHorizontalPodAutoscalerSpecMaxReplicas = prometheus.NewDesc(
		"kube_hpa_spec_max_replicas",
		"Upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.",
		[]string{"namespace", "hpa"}, nil,
	)
	descHorizontalPodAutoscalerSpecMinReplicas = prometheus.NewDesc(
		"kube_hpa_spec_min_replicas",
		"Lower limit for the number of pods that can be set by the autoscaler, default 1.",
		[]string{"namespace", "hpa"}, nil,
	)
	descHorizontalPodAutoscalerStatusCurrentReplicas = prometheus.NewDesc(
		"kube_hpa_status_current_replicas",
		"Current number of replicas of pods managed by this autoscaler.",
		[]string{"namespace", "hpa"}, nil,
	)
	descHorizontalPodAutoscalerStatusDesiredReplicas = prometheus.NewDesc(
		"kube_hpa_status_desired_replicas",
		"Desired number of replicas of pods managed by this autoscaler.",
		[]string{"namespace", "hpa"}, nil,
	)
)

type hpaStore interface {
	List() (hpas autoscaling.HorizontalPodAutoscalerList, err error)
}

// hpaCollector collects metrics about all Horizontal Pod Austoscalers in the cluster.
type hpaCollector struct {
	store hpaStore
}

// Describe implements the prometheus.Collector interface.
func (hc *hpaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descHorizontalPodAutoscalerMetadataGeneration
	ch <- descHorizontalPodAutoscalerSpecMaxReplicas
	ch <- descHorizontalPodAutoscalerSpecMinReplicas
	ch <- descHorizontalPodAutoscalerStatusCurrentReplicas
	ch <- descHorizontalPodAutoscalerStatusDesiredReplicas
}

// Collect implements the prometheus.Collector interface.
func (hc *hpaCollector) Collect(ch chan<- prometheus.Metric) {
	hpas, err := hc.store.List()
	if err != nil {
		glog.Errorf("listing Horizontal Pod Autoscalers failed: %s", err)
		return
	}
	for _, h := range hpas.Items {
		hc.collectHPA(ch, h)
	}
}

func (hc *hpaCollector) collectHPA(ch chan<- prometheus.Metric, h autoscaling.HorizontalPodAutoscaler) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{h.Namespace, h.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addGauge(descHorizontalPodAutoscalerMetadataGeneration, float64(h.ObjectMeta.Generation))
	addGauge(descHorizontalPodAutoscalerSpecMaxReplicas, float64(h.Spec.MaxReplicas))
	addGauge(descHorizontalPodAutoscalerSpecMinReplicas, float64(*h.Spec.MinReplicas))
	addGauge(descHorizontalPodAutoscalerStatusCurrentReplicas, float64(h.Status.CurrentReplicas))
	addGauge(descHorizontalPodAutoscalerStatusDesiredReplicas, float64(h.Status.DesiredReplicas))
}
