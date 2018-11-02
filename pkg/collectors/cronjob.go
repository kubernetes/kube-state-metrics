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

	descCronJobLabels = metrics.NewMetricFamilyDef(
		descCronJobLabelsName,
		descCronJobLabelsHelp,
		descCronJobLabelsDefaultLabels, nil,
	)

	descCronJobInfo = metrics.NewMetricFamilyDef(
		"kube_cronjob_info",
		"Info about cronjob.",
		append(descCronJobLabelsDefaultLabels, "schedule", "concurrency_policy"),
		nil,
	)
	descCronJobCreated = metrics.NewMetricFamilyDef(
		"kube_cronjob_created",
		"Unix creation timestamp",
		descCronJobLabelsDefaultLabels,
		nil,
	)
	descCronJobStatusActive = metrics.NewMetricFamilyDef(
		"kube_cronjob_status_active",
		"Active holds pointers to currently running jobs.",
		descCronJobLabelsDefaultLabels,
		nil,
	)
	descCronJobStatusLastScheduleTime = metrics.NewMetricFamilyDef(
		"kube_cronjob_status_last_schedule_time",
		"LastScheduleTime keeps information of when was the last time the job was successfully scheduled.",
		descCronJobLabelsDefaultLabels,
		nil,
	)
	descCronJobSpecSuspend = metrics.NewMetricFamilyDef(
		"kube_cronjob_spec_suspend",
		"Suspend flag tells the controller to suspend subsequent executions.",
		descCronJobLabelsDefaultLabels,
		nil,
	)
	descCronJobSpecStartingDeadlineSeconds = metrics.NewMetricFamilyDef(
		"kube_cronjob_spec_starting_deadline_seconds",
		"Deadline in seconds for starting the job if it misses scheduled time for any reason.",
		descCronJobLabelsDefaultLabels,
		nil,
	)
	descCronJobNextScheduledTime = metrics.NewMetricFamilyDef(
		"kube_cronjob_next_schedule_time",
		"Next time the cronjob should be scheduled. The time after lastScheduleTime, or after the cron job's creation time if it's never been scheduled. Use this to determine if the job is delayed.",
		descCronJobLabelsDefaultLabels,
		nil,
	)
)

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

func cronJobLabelsDesc(labelKeys []string) *metrics.MetricFamilyDef {
	return metrics.NewMetricFamilyDef(
		descCronJobLabelsName,
		descCronJobLabelsHelp,
		append(descCronJobLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func generateCronJobMetrics(obj interface{}) []*metrics.Metric {
	ms := []*metrics.Metric{}

	// TODO: Refactor
	jPointer := obj.(*batchv1beta1.CronJob)
	j := *jPointer

	addGauge := func(desc *metrics.MetricFamilyDef, v float64, lv ...string) {
		lv = append([]string{j.Namespace, j.Name}, lv...)

		m, err := metrics.NewMetric(desc.Name, desc.LabelKeys, lv, v)
		if err != nil {
			panic(err)
		}

		ms = append(ms, m)
	}

	if j.Spec.StartingDeadlineSeconds != nil {
		addGauge(descCronJobSpecStartingDeadlineSeconds, float64(*j.Spec.StartingDeadlineSeconds))
	}

	// If the cron job is suspended, don't track the next scheduled time
	nextScheduledTime, err := getNextScheduledTime(j.Spec.Schedule, j.Status.LastScheduleTime, j.CreationTimestamp)
	if err != nil {
		panic(err)
	} else if !*j.Spec.Suspend {
		addGauge(descCronJobNextScheduledTime, float64(nextScheduledTime.Unix()))
	}

	addGauge(descCronJobInfo, 1, j.Spec.Schedule, string(j.Spec.ConcurrencyPolicy))

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(j.Labels)
	addGauge(cronJobLabelsDesc(labelKeys), 1, labelValues...)

	if !j.CreationTimestamp.IsZero() {
		addGauge(descCronJobCreated, float64(j.CreationTimestamp.Unix()))
	}
	addGauge(descCronJobStatusActive, float64(len(j.Status.Active)))
	if j.Spec.Suspend != nil {
		addGauge(descCronJobSpecSuspend, boolFloat64(*j.Spec.Suspend))
	}

	if j.Status.LastScheduleTime != nil {
		addGauge(descCronJobStatusLastScheduleTime, float64(j.Status.LastScheduleTime.Unix()))
	}

	return ms
}
