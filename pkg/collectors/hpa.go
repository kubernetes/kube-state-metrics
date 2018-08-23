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
	autoscaling "k8s.io/api/autoscaling/v2beta1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	descHorizontalPodAutoscalerLabelsName          = "kube_hpa_labels"
	descHorizontalPodAutoscalerLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descHorizontalPodAutoscalerLabelsDefaultLabels = []string{"namespace", "hpa"}

	descHorizontalPodAutoscalerMetadataGeneration = prometheus.NewDesc(
		"kube_hpa_metadata_generation",
		"The generation observed by the HorizontalPodAutoscaler controller.",
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerSpecMaxReplicas = prometheus.NewDesc(
		"kube_hpa_spec_max_replicas",
		"Upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.",
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerSpecMinReplicas = prometheus.NewDesc(
		"kube_hpa_spec_min_replicas",
		"Lower limit for the number of pods that can be set by the autoscaler, default 1.",
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerStatusCurrentReplicas = prometheus.NewDesc(
		"kube_hpa_status_current_replicas",
		"Current number of replicas of pods managed by this autoscaler.",
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerStatusDesiredReplicas = prometheus.NewDesc(
		"kube_hpa_status_desired_replicas",
		"Desired number of replicas of pods managed by this autoscaler.",
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerLabels = prometheus.NewDesc(
		descHorizontalPodAutoscalerLabelsName,
		descHorizontalPodAutoscalerLabelsHelp,
		descHorizontalPodAutoscalerLabelsDefaultLabels,
		nil,
	)
	descHorizontalPodAutoscalerCondition = prometheus.NewDesc(
		"kube_hpa_status_condition",
		"The condition of this autoscaler.",
		append(descHorizontalPodAutoscalerLabelsDefaultLabels, "condition", "status"),
		nil,
	)
)

type HPALister func() (autoscaling.HorizontalPodAutoscalerList, error)

func (l HPALister) List() (autoscaling.HorizontalPodAutoscalerList, error) {
	return l()
}

func RegisterHorizontalPodAutoScalerCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Autoscaling().V2beta1().HorizontalPodAutoscalers().Informer().(cache.SharedInformer))
	}

	hpaLister := HPALister(func() (hpas autoscaling.HorizontalPodAutoscalerList, err error) {
		for _, hpainf := range infs {
			for _, h := range hpainf.GetStore().List() {
				hpas.Items = append(hpas.Items, *(h.(*autoscaling.HorizontalPodAutoscaler)))
			}
		}
		return hpas, nil
	})

	registry.MustRegister(&hpaCollector{store: hpaLister, opts: opts})
	infs.Run(context.Background().Done())
}

type hpaStore interface {
	List() (hpas autoscaling.HorizontalPodAutoscalerList, err error)
}

// hpaCollector collects metrics about all Horizontal Pod Austoscalers in the cluster.
type hpaCollector struct {
	store hpaStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (hc *hpaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descHorizontalPodAutoscalerMetadataGeneration
	ch <- descHorizontalPodAutoscalerSpecMaxReplicas
	ch <- descHorizontalPodAutoscalerSpecMinReplicas
	ch <- descHorizontalPodAutoscalerStatusCurrentReplicas
	ch <- descHorizontalPodAutoscalerStatusDesiredReplicas
	ch <- descHorizontalPodAutoscalerLabels
}

// Collect implements the prometheus.Collector interface.
func (hc *hpaCollector) Collect(ch chan<- prometheus.Metric) {
	hpas, err := hc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "horizontalpodautoscaler"}).Inc()
		glog.Errorf("listing HorizontalPodAutoscalers failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "horizontalpodautoscaler"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "horizontalpodautoscaler"}).Observe(float64(len(hpas.Items)))
	for _, h := range hpas.Items {
		hc.collectHPA(ch, h)
	}

	glog.V(4).Infof("collected %d hpas", len(hpas.Items))
}

func hpaLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descHorizontalPodAutoscalerLabelsName,
		descHorizontalPodAutoscalerLabelsHelp,
		append(descHorizontalPodAutoscalerLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func (hc *hpaCollector) collectHPA(ch chan<- prometheus.Metric, h autoscaling.HorizontalPodAutoscaler) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{h.Namespace, h.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(h.Labels)
	addGauge(hpaLabelsDesc(labelKeys), 1, labelValues...)
	addGauge(descHorizontalPodAutoscalerMetadataGeneration, float64(h.ObjectMeta.Generation))
	addGauge(descHorizontalPodAutoscalerSpecMaxReplicas, float64(h.Spec.MaxReplicas))
	addGauge(descHorizontalPodAutoscalerSpecMinReplicas, float64(*h.Spec.MinReplicas))
	addGauge(descHorizontalPodAutoscalerStatusCurrentReplicas, float64(h.Status.CurrentReplicas))
	addGauge(descHorizontalPodAutoscalerStatusDesiredReplicas, float64(h.Status.DesiredReplicas))

	for _, c := range h.Status.Conditions {
		addConditionMetrics(ch, descHorizontalPodAutoscalerCondition, c.Status, h.Namespace, h.Name, string(c.Type))
	}
}
