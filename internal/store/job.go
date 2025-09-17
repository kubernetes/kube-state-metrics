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

package store

import (
	"context"
	"strconv"

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	v1batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	descJobAnnotationsName     = "kube_job_annotations"
	descJobAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descJobLabelsName          = "kube_job_labels"
	descJobLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descJobLabelsDefaultLabels = []string{"namespace", "job_name"}
	jobFailureReasons          = []string{"BackoffLimitExceeded", "DeadlineExceeded", "Evicted"}
)

func jobMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			descJobAnnotationsName,
			descJobAnnotationsHelp,
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				if len(allowAnnotationsList) == 0 {
					return &metric.Family{}
				}
				annotationKeys, annotationValues := createPrometheusLabelKeysValues("annotation", j.Annotations, allowAnnotationsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   annotationKeys,
							LabelValues: annotationValues,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			descJobLabelsName,
			descJobLabelsHelp,
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				if len(allowLabelsList) == 0 {
					return &metric.Family{}
				}
				labelKeys, labelValues := createPrometheusLabelKeysValues("label", j.Labels, allowLabelsList)
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   labelKeys,
							LabelValues: labelValues,
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_info",
			"Information about job.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(_ *v1batch.Job) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: 1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_created",
			"Unix creation timestamp",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				ms := []*metric.Metric{}

				if !j.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						Value: float64(j.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_spec_parallelism",
			"The maximum desired number of pods the job should run at any given time.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				ms := []*metric.Metric{}

				if j.Spec.Parallelism != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(*j.Spec.Parallelism),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_spec_completions",
			"The desired number of successfully finished pods the job should be run with.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				ms := []*metric.Metric{}

				if j.Spec.Completions != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(*j.Spec.Completions),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_spec_active_deadline_seconds",
			"The duration in seconds relative to the startTime that the job may be active before the system tries to terminate it.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				ms := []*metric.Metric{}

				if j.Spec.ActiveDeadlineSeconds != nil {
					ms = append(ms, &metric.Metric{
						Value: float64(*j.Spec.ActiveDeadlineSeconds),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_status_succeeded",
			"The number of pods which reached Phase Succeeded.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(j.Status.Succeeded),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_status_failed",
			"The number of pods which reached Phase Failed and the reason for failure.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				var ms []*metric.Metric

				if float64(j.Status.Failed) == 0 {
					return &metric.Family{
						Metrics: []*metric.Metric{
							{
								Value: float64(j.Status.Failed),
							},
						},
					}
				}

				reasonKnown := false
				for _, c := range j.Status.Conditions {
					condition := c
					if condition.Type == v1batch.JobFailed {
						for _, reason := range jobFailureReasons {
							reasonKnown = reasonKnown || failureReason(&condition, reason)

							// for known reasons
							ms = append(ms, &metric.Metric{
								LabelKeys:   []string{"reason"},
								LabelValues: []string{reason},
								Value:       boolFloat64(failureReason(&condition, reason)),
							})
						}
					}
				}
				// for unknown reasons
				if !reasonKnown {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{"reason"},
						LabelValues: []string{""},
						Value:       float64(j.Status.Failed),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_status_active",
			"The number of actively running pods.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: float64(j.Status.Active),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_status_ready",
			"The number of ready pods that belong to this Job.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				value := float64(0)
				if j.Status.Ready != nil {
					value = float64(*j.Status.Ready)
				}
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							Value: value,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_complete",
			"The job has completed its execution.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
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

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_failed",
			"The job has failed its execution.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				ms := []*metric.Metric{}

				for _, c := range j.Status.Conditions {
					if c.Type == v1batch.JobFailed {
						metrics := addConditionMetrics(c.Status)
						for _, m := range metrics {
							metric := m
							metric.LabelKeys = []string{"condition"}
							ms = append(ms, metric)
						}
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_status_start_time",
			"StartTime represents time when the job was acknowledged by the Job Manager.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				ms := []*metric.Metric{}

				if j.Status.StartTime != nil {
					ms = append(ms, &metric.Metric{

						Value: float64(j.Status.StartTime.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_status_completion_time",
			"CompletionTime represents time when the job was completed.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				ms := []*metric.Metric{}
				if j.Status.CompletionTime != nil {
					ms = append(ms, &metric.Metric{

						Value: float64(j.Status.CompletionTime.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_status_suspended",
			"The number of pods which reached Phase Suspended.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				ms := []*metric.Metric{}
				for _, c := range j.Status.Conditions {
					if c.Type == v1batch.JobSuspended {
						ms = append(ms, &metric.Metric{
							Value: boolFloat64(c.Status == v1.ConditionTrue),
						})
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGeneratorWithStability(
			"kube_job_owner",
			"Information about the Job's owner.",
			metric.Gauge,
			basemetrics.STABLE,
			"",
			wrapJobFunc(func(j *v1batch.Job) *metric.Family {
				labelKeys := []string{"owner_kind", "owner_name", "owner_is_controller"}

				owners := j.GetOwnerReferences()

				if len(owners) == 0 {
					return &metric.Family{
						Metrics: []*metric.Metric{
							{
								LabelKeys:   labelKeys,
								LabelValues: []string{"", "", ""},
								Value:       1,
							},
						},
					}
				}

				ms := make([]*metric.Metric, len(owners))

				for i, owner := range owners {
					if owner.Controller != nil {
						ms[i] = &metric.Metric{
							LabelKeys:   labelKeys,
							LabelValues: []string{owner.Kind, owner.Name, strconv.FormatBool(*owner.Controller)},
							Value:       1,
						}
					} else {
						ms[i] = &metric.Metric{
							LabelKeys:   labelKeys,
							LabelValues: []string{owner.Kind, owner.Name, "false"},
							Value:       1,
						}
					}
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
}

func wrapJobFunc(f func(*v1batch.Job) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		job := obj.(*v1batch.Job)

		metricFamily := f(job)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descJobLabelsDefaultLabels, []string{job.Namespace, job.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createJobListWatch(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.BatchV1().Jobs(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.FieldSelector = fieldSelector
			return kubeClient.BatchV1().Jobs(ns).Watch(context.TODO(), opts)
		},
	}
}

func failureReason(jc *v1batch.JobCondition, reason string) bool {
	if jc == nil {
		return false
	}
	return jc.Reason == reason
}
