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
	"k8s.io/kube-state-metrics/pkg/metrics"
)

type Store interface {
	GetAll() []*metrics.Metric
}

// Collector represents a kube-state-metrics metric collector. It is stripped
// down version of the Prometheus client_golang collector.
type Collector struct {
	Store Store
}

func NewCollector(s Store) *Collector {
	return &Collector{s}
}

// Collect returns all metrics of the underlying store of the collector.
func (c *Collector) Collect() []*metrics.Metric {
	return c.Store.GetAll()
}

func NewMetricFamilyDef(name, help string, labelKeys []string, constLabels prometheus.Labels) *MetricFamilyDef {
	return &MetricFamilyDef{name, help, labelKeys, constLabels}
}

// MetricFamilyDef represents a metric family definition
type MetricFamilyDef struct {
	Name        string
	Help        string
	LabelKeys   []string
	ConstLabels prometheus.Labels
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
func addConditionMetrics(desc *metricFamilyDef, cs v1.ConditionStatus, lv ...string) []*metrics.Metric {
	ms := []*metrics.Metric{}
	m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, append(lv, "true"), boolFloat64(cs == v1.ConditionTrue))
	if err != nil {
		panic(err)
	}
	ms = append(ms, m)
	m, err = metrics.NewMetric(desc.Name, desc.LabelKeys, append(lv, "false"), boolFloat64(cs == v1.ConditionFalse))
	if err != nil {
		panic(err)
	}
	ms = append(ms, m)
	m, err = metrics.NewMetric(desc.Name, desc.LabelKeys, append(lv, "unknown"), boolFloat64(cs == v1.ConditionUnknown))
	if err != nil {
		panic(err)
	}
	ms = append(ms, m)

	return ms
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
