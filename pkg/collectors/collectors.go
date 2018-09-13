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
	"time"

	"regexp"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	resyncPeriod = 5 * time.Minute

	ScrapeErrorTotalMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ksm_scrape_error_total",
			Help: "Total scrape errors encountered when scraping a resource",
		},
		[]string{"resource"},
	)

	ResourcesPerScrapeMetric = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "ksm_resources_per_scrape",
			Help: "Number of resources returned per scrape",
		},
		[]string{"resource"},
	)

	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

var AvailableCollectors = map[string]func(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options){
	"cronjobs":                 RegisterCronJobCollector,
	"daemonsets":               RegisterDaemonSetCollector,
	"deployments":              RegisterDeploymentCollector,
	"jobs":                     RegisterJobCollector,
	"limitranges":              RegisterLimitRangeCollector,
	"nodes":                    RegisterNodeCollector,
	"pods":                     RegisterPodCollector,
	"replicasets":              RegisterReplicaSetCollector,
	"replicationcontrollers":   RegisterReplicationControllerCollector,
	"resourcequotas":           RegisterResourceQuotaCollector,
	"services":                 RegisterServiceCollector,
	"statefulsets":             RegisterStatefulSetCollector,
	"persistentvolumes":        RegisterPersistentVolumeCollector,
	"persistentvolumeclaims":   RegisterPersistentVolumeClaimCollector,
	"namespaces":               RegisterNamespaceCollector,
	"horizontalpodautoscalers": RegisterHorizontalPodAutoScalerCollector,
	"endpoints":                RegisterEndpointCollector,
	"secrets":                  RegisterSecretCollector,
	"configmaps":               RegisterConfigMapCollector,
}

type SharedInformerList []cache.SharedInformer

func (sil SharedInformerList) Run(stopCh <-chan struct{}) {
	for _, sinf := range sil {
		go sinf.Run(stopCh)
	}
}

func boolFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// addConditionMetrics generates one metric for each possible node condition
// status. For this function to work properly, the last label in the metric
// description must be the condition.
func addConditionMetrics(ch chan<- prometheus.Metric, desc *prometheus.Desc, cs v1.ConditionStatus, lv ...string) {
	ch <- prometheus.MustNewConstMetric(
		desc, prometheus.GaugeValue, boolFloat64(cs == v1.ConditionTrue),
		append(lv, "true")...,
	)
	ch <- prometheus.MustNewConstMetric(
		desc, prometheus.GaugeValue, boolFloat64(cs == v1.ConditionFalse),
		append(lv, "false")...,
	)
	ch <- prometheus.MustNewConstMetric(
		desc, prometheus.GaugeValue, boolFloat64(cs == v1.ConditionUnknown),
		append(lv, "unknown")...,
	)
}

func kubeLabelsToPrometheusLabels(labels map[string]string) ([]string, []string) {
	labelKeys := make([]string, len(labels))
	labelValues := make([]string, len(labels))
	i := 0
	for k, v := range labels {
		labelKeys[i] = "label_" + sanitizeLabelName(k)
		labelValues[i] = v
		i++
	}
	return labelKeys, labelValues
}

func kubeAnnotationsToPrometheusAnnotations(annotations map[string]string) ([]string, []string) {
	annotationKeys := make([]string, len(annotations))
	annotationValues := make([]string, len(annotations))
	i := 0
	for k, v := range annotations {
		annotationKeys[i] = "annotation_" + sanitizeLabelName(k)
		annotationValues[i] = v
		i++
	}
	return annotationKeys, annotationValues
}

func sanitizeLabelName(s string) string {
	return invalidLabelCharRE.ReplaceAllString(s, "_")
}
