/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package collector

import (
	"regexp"
	"time"

	"k8s.io/api/core/v1"

	"k8s.io/kube-state-metrics/pkg/metric"
)

var (
	resyncPeriod       = 5 * time.Minute
	invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

func boolFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// addConditionMetrics generates one metric for each possible node condition
// status. For this function to work properly, the last label in the metric
// description must be the condition.
func addConditionMetrics(cs v1.ConditionStatus) []*metric.Metric {
	return []*metric.Metric{
		{
			LabelValues: []string{"true"},
			Value:       boolFloat64(cs == v1.ConditionTrue),
		},
		{
			LabelValues: []string{"false"},
			Value:       boolFloat64(cs == v1.ConditionFalse),
		},
		{
			LabelValues: []string{"unknown"},
			Value:       boolFloat64(cs == v1.ConditionUnknown),
		},
	}
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
