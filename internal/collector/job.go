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

package collector

import (
	"k8s.io/kube-state-metrics/pkg/metric"

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

	jobMetricFamilies = []metric.FamilyGenerator{
		{
			Name: descJobLabelsName,
			Type: metric.MetricTypeGauge,
			Help: descJobLabelsHelp,
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(j.Labels)
				return metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					},
				}
			}),
		},
		{
			Name: "kube_job_info",
			Type: metric.MetricTypeGauge,
			Help: "Information about job.",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				return metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: 1,
						},
					},
				}
			}),
		},
		{
			Name: "kube_job_created",
			Type: metric.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				ms := []*metric.Metric{}

				if !j.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(j.CreationTimestamp.Unix()),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_job_spec_parallelism",
			Type: metric.MetricTypeGauge,
			Help: "The maximum desired number of pods the job should run at any given time.",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				ms := []*metric.Metric{}

				if j.Spec.Parallelism != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(*j.Spec.Parallelism),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_job_spec_completions",
			Type: metric.MetricTypeGauge,
			Help: "The desired number of successfully finished pods the job should be run with.",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				ms := []*metric.Metric{}

				if j.Spec.Completions != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(*j.Spec.Completions),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_job_spec_active_deadline_seconds",
			Type: metric.MetricTypeGauge,
			Help: "The duration in seconds relative to the startTime that the job may be active before the system tries to terminate it.",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				ms := []*metric.Metric{}

				if j.Spec.ActiveDeadlineSeconds != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(*j.Spec.ActiveDeadlineSeconds),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_job_status_succeeded",
			Type: metric.MetricTypeGauge,
			Help: "The number of pods which reached Phase Succeeded.",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				return metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(j.Status.Succeeded),
						},
					},
				}
			}),
		},
		{
			Name: "kube_job_status_failed",
			Type: metric.MetricTypeGauge,
			Help: "The number of pods which reached Phase Failed.",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				return metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(j.Status.Failed),
						},
					},
				}
			}),
		},
		{
			Name: "kube_job_status_active",
			Type: metric.MetricTypeGauge,
			Help: "The number of actively running pods.",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				return metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(j.Status.Active),
						},
					},
				}
			}),
		},
		{
			Name: "kube_job_complete",
			Type: metric.MetricTypeGauge,
			Help: "The job has completed its execution.",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				ms := []*metric.Metric{}
				for _, c := range j.Status.Conditions {
					if c.Type == v1batch.JobComplete {
						metrics := addConditionMetrics(c.Status)
						for _, m := range metrics {
							metric := m
							metric.LabelKeys = []string{"condition"}
							ms = append(ms, metric)
						}
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_job_failed",
			Type: metric.MetricTypeGauge,
			Help: "The job has failed its execution.",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				ms := []*metric.Metric{}

				for _, c := range j.Status.Conditions {
					if c.Type == v1batch.JobFailed {
						metrics := addConditionMetrics(c.Status)
						for _, m := range metrics {
							metric := m
							metric.LabelKeys = []string{"condition"}
							ms = append(ms, m)
						}
					}
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_job_status_start_time",
			Type: metric.MetricTypeGauge,
			Help: "StartTime represents time when the job was acknowledged by the Job Manager.",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				ms := []*metric.Metric{}

				if j.Status.StartTime != nil {
					ms = append(ms, &metric.Metric{

						Value: float64(j.Status.StartTime.Unix()),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
		{
			Name: "kube_job_status_completion_time",
			Type: metric.MetricTypeGauge,
			Help: "CompletionTime represents time when the job was completed.",
			GenerateFunc: wrapJobFunc(func(j *v1batch.Job) metric.Family {
				ms := []*metric.Metric{}
				if j.Status.CompletionTime != nil {
					ms = append(ms, &metric.Metric{

						Value: float64(j.Status.CompletionTime.Unix()),
					})
				}

				return metric.Family{
					Metrics: ms,
				}
			}),
		},
	}
)

func wrapJobFunc(f func(*v1batch.Job) metric.Family) func(interface{}) metric.Family {
	return func(obj interface{}) metric.Family {
		job := obj.(*v1batch.Job)

		metricFamily := f(job)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descJobLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{job.Namespace, job.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

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
