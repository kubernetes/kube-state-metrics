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
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	v2batch "k8s.io/client-go/pkg/apis/batch/v2alpha1"
	"k8s.io/client-go/tools/cache"
)

type timeNowT func() time.Time

var (
	timeNow timeNowT // so that we can mock out `time.Now()` in tests
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
	descCronJobSchedulingDelay = prometheus.NewDesc(
		"kube_cronjob_scheduling_delay",
		"Number of seconds the cron job is delayed scheduling",
		[]string{"namespace", "cronjob"}, nil,
	)
)

func init() {
	timeNow = time.Now
}

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
	ch <- descCronJobSchedulingDelay
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

func getSchedulingDelaySeconds(schedule string, lastScheduleTime time.Time) float64 {
	sched, err := cron.ParseStandard(schedule)
	if err != nil {
		glog.Errorf("Failed to parse cron job schedule '%s': %s", schedule, err)
		return 0
	}
	nextScheduleTime := sched.Next(lastScheduleTime)
	delay := timeNow().Sub(nextScheduleTime).Seconds()
	if delay < 0 {
		return 0
	} else {
		return delay
	}
}

func (jc *cronJobCollector) collectCronJob(ch chan<- prometheus.Metric, j v2batch.CronJob) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{j.Namespace, j.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addCounter := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{j.Namespace, j.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, lv...)
	}

	if j.Spec.StartingDeadlineSeconds != nil {
		addGauge(descCronJobSpecStartingDeadlineSeconds, float64(*j.Spec.StartingDeadlineSeconds))
	}

	// If the cron job is suspended, don't track delays
	if j.Status.LastScheduleTime != nil && !*j.Spec.Suspend {
		addGauge(descCronJobSchedulingDelay, getSchedulingDelaySeconds(j.Spec.Schedule, j.Status.LastScheduleTime.Time))
	}

	addGauge(descCronJobInfo, 1, j.Spec.Schedule, string(j.Spec.ConcurrencyPolicy))
	addGauge(descCronJobStatusActive, float64(len(j.Status.Active)))
	addGauge(descCronJobSpecSuspend, boolFloat64(*j.Spec.Suspend))

	if j.Status.LastScheduleTime != nil {
		addCounter(descCronJobStatusLastScheduleTime, float64(j.Status.LastScheduleTime.Unix()))
	}
}
