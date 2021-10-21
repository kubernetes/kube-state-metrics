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
	"time"

	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	descCronJobAnnotationsName     = "kube_cronjob_annotations"
	descCronJobAnnotationsHelp     = "Kubernetes annotations converted to Prometheus labels."
	descCronJobLabelsName          = "kube_cronjob_labels"
	descCronJobLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descCronJobLabelsDefaultLabels = []string{"namespace", "cronjob"}
)

func cronJobMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGenerator(
			descCronJobAnnotationsName,
			descCronJobAnnotationsHelp,
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
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
		*generator.NewFamilyGenerator(
			descCronJobLabelsName,
			descCronJobLabelsHelp,
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
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
		*generator.NewFamilyGenerator(
			"kube_cronjob_info",
			"Info about cronjob.",
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{"schedule", "concurrency_policy"},
							LabelValues: []string{j.Spec.Schedule, string(j.Spec.ConcurrencyPolicy)},
							Value:       1,
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_cronjob_created",
			"Unix creation timestamp",
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
				ms := []*metric.Metric{}
				if !j.CreationTimestamp.IsZero() {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(j.CreationTimestamp.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_cronjob_status_active",
			"Active holds pointers to currently running jobs.",
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
				return &metric.Family{
					Metrics: []*metric.Metric{
						{
							LabelKeys:   []string{},
							LabelValues: []string{},
							Value:       float64(len(j.Status.Active)),
						},
					},
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_cronjob_status_last_schedule_time",
			"LastScheduleTime keeps information of when was the last time the job was successfully scheduled.",
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
				ms := []*metric.Metric{}

				if j.Status.LastScheduleTime != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(j.Status.LastScheduleTime.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_cronjob_spec_suspend",
			"Suspend flag tells the controller to suspend subsequent executions.",
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
				ms := []*metric.Metric{}

				if j.Spec.Suspend != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       boolFloat64(*j.Spec.Suspend),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_cronjob_spec_starting_deadline_seconds",
			"Deadline in seconds for starting the job if it misses scheduled time for any reason.",
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
				ms := []*metric.Metric{}

				if j.Spec.StartingDeadlineSeconds != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(*j.Spec.StartingDeadlineSeconds),
					})

				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_cronjob_next_schedule_time",
			"Next time the cronjob should be scheduled. The time after lastScheduleTime, or after the cron job's creation time if it's never been scheduled. Use this to determine if the job is delayed.",
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
				ms := []*metric.Metric{}

				// If the cron job is suspended, don't track the next scheduled time
				nextScheduledTime, err := getNextScheduledTime(j.Spec.Schedule, j.Status.LastScheduleTime, j.CreationTimestamp)
				if err != nil {
					panic(err)
				} else if !*j.Spec.Suspend {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(nextScheduledTime.Unix()),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_cronjob_metadata_resource_version",
			"Resource version representing a specific version of the cronjob.",
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
				return &metric.Family{
					Metrics: resourceVersionMetric(j.ObjectMeta.ResourceVersion),
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_cronjob_spec_successful_job_history_limit",
			"Successful job history limit tells the controller how many completed jobs should be preserved.",
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
				ms := []*metric.Metric{}

				if j.Spec.SuccessfulJobsHistoryLimit != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(*j.Spec.SuccessfulJobsHistoryLimit),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
		*generator.NewFamilyGenerator(
			"kube_cronjob_spec_failed_job_history_limit",
			"Failed job history limit tells the controller how many failed jobs should be preserved.",
			metric.Gauge,
			"",
			wrapCronJobFunc(func(j *batchv1beta1.CronJob) *metric.Family {
				ms := []*metric.Metric{}

				if j.Spec.FailedJobsHistoryLimit != nil {
					ms = append(ms, &metric.Metric{
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(*j.Spec.FailedJobsHistoryLimit),
					})
				}

				return &metric.Family{
					Metrics: ms,
				}
			}),
		),
	}
}

func wrapCronJobFunc(f func(*batchv1beta1.CronJob) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		cronJob := obj.(*batchv1beta1.CronJob)

		metricFamily := f(cronJob)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys = append(descCronJobLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{cronJob.Namespace, cronJob.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createCronJobListWatch(kubeClient clientset.Interface, ns string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.BatchV1beta1().CronJobs(ns).List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.BatchV1beta1().CronJobs(ns).Watch(context.TODO(), opts)
		},
	}
}

func getNextScheduledTime(schedule string, lastScheduleTime *metav1.Time, createdTime metav1.Time) (time.Time, error) {
	sched, err := cron.ParseStandard(schedule)
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "Failed to parse cron job schedule '%s'", schedule)
	}
	if !lastScheduleTime.IsZero() {
		return sched.Next(lastScheduleTime.Time), nil
	}
	if !createdTime.IsZero() {
		return sched.Next(createdTime.Time), nil
	}
	return time.Time{}, errors.New("createdTime and lastScheduleTime are both zero")
}
