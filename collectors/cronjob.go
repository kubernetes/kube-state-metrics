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

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	v2batch "k8s.io/client-go/pkg/apis/batch/v2alpha1"
	"k8s.io/client-go/tools/cache"
)

var (
	descCronJobInfo = prometheus.NewDesc(
		"kube_cronjob_info",
		"Info about cronjob.",
		[]string{"namespace", "cronjob", "schedule", "concurrency_policy"}, nil,
	)
	descCronJobStatusActive = prometheus.NewDesc(
		"kube_cronjob_status_active",
		"Active holds pointers to currently running jobs.",
		[]string{"namespace", "cronjob"}, nil,
	)
	descCronJobStatusLastScheduleTime = prometheus.NewDesc(
		"kube_cronjob_status_last_schedule_time",
		"LastScheduleTime keeps information of when was the last time the job was successfully scheduled.",
		[]string{"namespace", "cronjob"}, nil,
	)
	descCronJobSpecSuspend = prometheus.NewDesc(
		"kube_cronjob_spec_suspend",
		"Suspend flag tells the controller to suspend subsequent executions.",
		[]string{"namespace", "cronjob"}, nil,
	)
	descCronJobSpecStartingDeadlineSeconds = prometheus.NewDesc(
		"kube_cronjob_spec_starting_deadline_seconds",
		"Deadline in seconds for starting the job if it misses scheduled time for any reason.",
		[]string{"namespace", "cronjob"}, nil,
	)
	descCronJobNextScheduledTime = prometheus.NewDesc(
		"kube_cronjob_next_schedule_time",
		"Next time the cronjob should be scheduled. The time after lastScheduleTime, or after the cron job's creation time if it's never been scheduled. Use this to determine if the job is delayed.",
		[]string{"namespace", "cronjob"}, nil,
	)
)

type CronJobLister func() ([]v2batch.CronJob, error)

func (l CronJobLister) List() ([]v2batch.CronJob, error) {
	return l()
}

func RegisterCronJobCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface) {
	client := kubeClient.BatchV2alpha1().RESTClient()
	cjlw := cache.NewListWatchFromClient(client, "cronjobs", api.NamespaceAll, nil)
	cjinf := cache.NewSharedInformer(cjlw, &v2batch.CronJob{}, resyncPeriod)

	cronJobLister := CronJobLister(func() (cronjobs []v2batch.CronJob, err error) {
		for _, c := range cjinf.GetStore().List() {
			cronjobs = append(cronjobs, *(c.(*v2batch.CronJob)))
		}
		return cronjobs, nil
	})

	registry.MustRegister(&cronJobCollector{store: cronJobLister})
	go cjinf.Run(context.Background().Done())
}

type cronJobStore interface {
	List() (cronjobs []v2batch.CronJob, err error)
}

// cronJobCollector collects metrics about all cronjobs in the cluster.
type cronJobCollector struct {
	store cronJobStore
}

// Describe implements the prometheus.Collector interface.
func (dc *cronJobCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descCronJobInfo
	ch <- descCronJobStatusActive
	ch <- descCronJobStatusLastScheduleTime
	ch <- descCronJobSpecSuspend
	ch <- descCronJobSpecStartingDeadlineSeconds
	ch <- descCronJobNextScheduledTime
}

// Collect implements the prometheus.Collector interface.
func (cjc *cronJobCollector) Collect(ch chan<- prometheus.Metric) {
	cronjobs, err := cjc.store.List()
	if err != nil {
		glog.Errorf("listing cronjobs failed: %s", err)
		return
	}
	for _, cj := range cronjobs {
		cjc.collectCronJob(ch, cj)
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

func (jc *cronJobCollector) collectCronJob(ch chan<- prometheus.Metric, j v2batch.CronJob) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{j.Namespace, j.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	if j.Spec.StartingDeadlineSeconds != nil {
		addGauge(descCronJobSpecStartingDeadlineSeconds, float64(*j.Spec.StartingDeadlineSeconds))
	}

	// If the cron job is suspended, don't track the next scheduled time
	nextScheduledTime, err := getNextScheduledTime(j.Spec.Schedule, j.Status.LastScheduleTime, j.CreationTimestamp)
	if err != nil {
		glog.Errorf("%s", err)
	} else if !*j.Spec.Suspend {
		addGauge(descCronJobNextScheduledTime, float64(nextScheduledTime.Unix()))
	}

	addGauge(descCronJobInfo, 1, j.Spec.Schedule, string(j.Spec.ConcurrencyPolicy))
	addGauge(descCronJobStatusActive, float64(len(j.Status.Active)))
	if j.Spec.Suspend != nil {
		addGauge(descCronJobSpecSuspend, boolFloat64(*j.Spec.Suspend))
	}

	if j.Status.LastScheduleTime != nil {
		addGauge(descCronJobStatusLastScheduleTime, float64(j.Status.LastScheduleTime.Unix()))
	}
}
