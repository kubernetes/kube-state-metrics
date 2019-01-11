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
	"fmt"
	"time"

	"k8s.io/kube-state-metrics/pkg/metrics"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/robfig/cron"
)

var (
	descCronJobLabelsName          = "kube_cronjob_labels"
	descCronJobLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descCronJobLabelsDefaultLabels = []string{"namespace", "cronjob"}

	cronJobMetricFamilies = []metrics.FamilyGenerator{
		metrics.FamilyGenerator{
			Name: descCronJobLabelsName,
			Type: metrics.MetricTypeGauge,
			Help: descCronJobLabelsHelp,
			GenerateFunc: wrapCronJobFunc(func(j *batchv1beta1.CronJob) metrics.Family {
				labelKeys, labelValues := kubeLabelsToPrometheusLabels(j.Labels)
				return metrics.Family{
					&metrics.Metric{
						Name:        descCronJobLabelsName,
						LabelKeys:   labelKeys,
						LabelValues: labelValues,
						Value:       1,
					},
				}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_cronjob_info",
			Type: metrics.MetricTypeGauge,
			Help: "Info about cronjob.",
			GenerateFunc: wrapCronJobFunc(func(j *batchv1beta1.CronJob) metrics.Family {
				return metrics.Family{
					&metrics.Metric{
						Name:        "kube_cronjob_info",
						LabelKeys:   []string{"schedule", "concurrency_policy"},
						LabelValues: []string{j.Spec.Schedule, string(j.Spec.ConcurrencyPolicy)},
						Value:       1,
					},
				}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_cronjob_created",
			Type: metrics.MetricTypeGauge,
			Help: "Unix creation timestamp",
			GenerateFunc: wrapCronJobFunc(func(j *batchv1beta1.CronJob) metrics.Family {
				f := metrics.Family{}
				if !j.CreationTimestamp.IsZero() {
					f = append(f, &metrics.Metric{
						Name:        "kube_cronjob_created",
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(j.CreationTimestamp.Unix()),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_cronjob_status_active",
			Type: metrics.MetricTypeGauge,
			Help: "Active holds pointers to currently running jobs.",
			GenerateFunc: wrapCronJobFunc(func(j *batchv1beta1.CronJob) metrics.Family {
				return metrics.Family{
					&metrics.Metric{
						Name:        "kube_cronjob_status_active",
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(len(j.Status.Active)),
					},
				}
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_cronjob_status_last_schedule_time",
			Type: metrics.MetricTypeGauge,
			Help: "LastScheduleTime keeps information of when was the last time the job was successfully scheduled.",
			GenerateFunc: wrapCronJobFunc(func(j *batchv1beta1.CronJob) metrics.Family {
				f := metrics.Family{}

				if j.Status.LastScheduleTime != nil {
					f = append(f, &metrics.Metric{
						Name:        "kube_cronjob_status_last_schedule_time",
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(j.Status.LastScheduleTime.Unix()),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_cronjob_spec_suspend",
			Type: metrics.MetricTypeGauge,
			Help: "Suspend flag tells the controller to suspend subsequent executions.",
			GenerateFunc: wrapCronJobFunc(func(j *batchv1beta1.CronJob) metrics.Family {
				f := metrics.Family{}

				if j.Spec.Suspend != nil {
					f = append(f, &metrics.Metric{
						Name:        "kube_cronjob_spec_suspend",
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       boolFloat64(*j.Spec.Suspend),
					})
				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_cronjob_spec_starting_deadline_seconds",
			Type: metrics.MetricTypeGauge,
			Help: "Deadline in seconds for starting the job if it misses scheduled time for any reason.",
			GenerateFunc: wrapCronJobFunc(func(j *batchv1beta1.CronJob) metrics.Family {
				f := metrics.Family{}

				if j.Spec.StartingDeadlineSeconds != nil {
					f = append(f, &metrics.Metric{
						Name:        "kube_cronjob_spec_starting_deadline_seconds",
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(*j.Spec.StartingDeadlineSeconds),
					})

				}

				return f
			}),
		},
		metrics.FamilyGenerator{
			Name: "kube_cronjob_next_schedule_time",
			Type: metrics.MetricTypeGauge,
			Help: "Next time the cronjob should be scheduled. The time after lastScheduleTime, or after the cron job's creation time if it's never been scheduled. Use this to determine if the job is delayed.",
			GenerateFunc: wrapCronJobFunc(func(j *batchv1beta1.CronJob) metrics.Family {
				f := metrics.Family{}

				// If the cron job is suspended, don't track the next scheduled time
				nextScheduledTime, err := getNextScheduledTime(j.Spec.Schedule, j.Status.LastScheduleTime, j.CreationTimestamp)
				if err != nil {
					panic(err)
				} else if !*j.Spec.Suspend {
					f = append(f, &metrics.Metric{
						Name:        "kube_cronjob_next_schedule_time",
						LabelKeys:   []string{},
						LabelValues: []string{},
						Value:       float64(nextScheduledTime.Unix()),
					})
				}

				return f
			}),
		},
	}
)

func wrapCronJobFunc(f func(*batchv1beta1.CronJob) metrics.Family) func(interface{}) metrics.Family {
	return func(obj interface{}) metrics.Family {
		cronJob := obj.(*batchv1beta1.CronJob)

		metricFamily := f(cronJob)

		for _, m := range metricFamily {
			m.LabelKeys = append(descCronJobLabelsDefaultLabels, m.LabelKeys...)
			m.LabelValues = append([]string{cronJob.Namespace, cronJob.Name}, m.LabelValues...)
		}

		return metricFamily
	}
}

func createCronJobListWatch(kubeClient clientset.Interface, ns string) cache.ListWatch {
	return cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.BatchV1beta1().CronJobs(ns).List(opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.BatchV1beta1().CronJobs(ns).Watch(opts)
		},
	}
}

func getNextScheduledTime(schedule string, lastScheduleTime *metav1.Time, createdTime metav1.Time) (time.Time, error) {
	sched, err := cron.ParseStandard(schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("Failed to parse cron job schedule '%s': %s", schedule, err)
	}
	if !lastScheduleTime.IsZero() {
		return sched.Next((*lastScheduleTime).Time), nil
	}
	if !createdTime.IsZero() {
		return sched.Next(createdTime.Time), nil
	}
	return time.Time{}, fmt.Errorf("Created time and lastScheduleTime are both zero")
}
