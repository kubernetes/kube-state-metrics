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

	v1batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descJobLabelsName          = "kube_job_labels"
	descJobLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descJobLabelsDefaultLabels = []string{"namespace", "job_name"}

	descJobLabels = metrics.NewMetricFamilyDef(
		descJobLabelsName,
		descJobLabelsHelp,
		descJobLabelsDefaultLabels,
		nil,
	)

	descJobInfo = metrics.NewMetricFamilyDef(
		"kube_job_info",
		"Information about job.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobCreated = metrics.NewMetricFamilyDef(
		"kube_job_created",
		"Unix creation timestamp",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobSpecParallelism = metrics.NewMetricFamilyDef(
		"kube_job_spec_parallelism",
		"The maximum desired number of pods the job should run at any given time.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobSpecCompletions = metrics.NewMetricFamilyDef(
		"kube_job_spec_completions",
		"The desired number of successfully finished pods the job should be run with.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobSpecActiveDeadlineSeconds = metrics.NewMetricFamilyDef(
		"kube_job_spec_active_deadline_seconds",
		"The duration in seconds relative to the startTime that the job may be active before the system tries to terminate it.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobStatusSucceeded = metrics.NewMetricFamilyDef(
		"kube_job_status_succeeded",
		"The number of pods which reached Phase Succeeded.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobStatusFailed = metrics.NewMetricFamilyDef(
		"kube_job_status_failed",
		"The number of pods which reached Phase Failed.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobStatusActive = metrics.NewMetricFamilyDef(
		"kube_job_status_active",
		"The number of actively running pods.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobConditionComplete = metrics.NewMetricFamilyDef(
		"kube_job_complete",
		"The job has completed its execution.",
		append(descJobLabelsDefaultLabels, "condition"),
		nil,
	)
	descJobConditionFailed = metrics.NewMetricFamilyDef(
		"kube_job_failed",
		"The job has failed its execution.",
		append(descJobLabelsDefaultLabels, "condition"),
		nil,
	)
	descJobStatusStartTime = metrics.NewMetricFamilyDef(
		"kube_job_status_start_time",
		"StartTime represents time when the job was acknowledged by the Job Manager.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobStatusCompletionTime = metrics.NewMetricFamilyDef(
		"kube_job_status_completion_time",
		"CompletionTime represents time when the job was completed.",
		descJobLabelsDefaultLabels,
		nil,
	)
)

func createJobListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.BatchV1().Jobs(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.BatchV1().Jobs(ns).Watch(opts)
		},
	}
}

func jobLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descJobLabelsName,
		descJobLabelsHelp,
		append(descJobLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func generateJobMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	jPointer := obj.(*v1batch.Job)
	j := *jPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{j.Namespace, j.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}

	addGauge(descJobInfo, 1)

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(j.Labels)
	addGauge(jobLabelsDesc(labelKeys), 1, labelValues...)

	if j.Spec.Parallelism != nil {
		addGauge(descJobSpecParallelism, float64(*j.Spec.Parallelism))
	}

	if j.Spec.Completions != nil {
		addGauge(descJobSpecCompletions, float64(*j.Spec.Completions))
	}
	if !j.CreationTimestamp.IsZero() {
		addGauge(descJobCreated, float64(j.CreationTimestamp.Unix()))
	}

	if j.Spec.ActiveDeadlineSeconds != nil {
		addGauge(descJobSpecActiveDeadlineSeconds, float64(*j.Spec.ActiveDeadlineSeconds))
	}

	addGauge(descJobStatusSucceeded, float64(j.Status.Succeeded))
	addGauge(descJobStatusFailed, float64(j.Status.Failed))
	addGauge(descJobStatusActive, float64(j.Status.Active))

	if j.Status.StartTime != nil {
		addGauge(descJobStatusStartTime, float64(j.Status.StartTime.Unix()))
	}

	if j.Status.CompletionTime != nil {
		addGauge(descJobStatusCompletionTime, float64(j.Status.CompletionTime.Unix()))
	}

	for _, c := range j.Status.Conditions {
		switch c.Type {
		case v1batch.JobComplete:
			ms = append(ms, addConditionMetrics(descJobConditionComplete, c.Status, j.Namespace, j.Name)...)
		case v1batch.JobFailed:
			ms = append(ms, addConditionMetrics(descJobConditionFailed, c.Status, j.Namespace, j.Name)...)
		}
	}
	return ms
}
