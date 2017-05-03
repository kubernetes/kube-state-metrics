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

package main

import (
	"strconv"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	v1batch "k8s.io/client-go/pkg/apis/batch/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	descJobInfo = prometheus.NewDesc(
		"kube_job_info",
		"Information about job.",
		[]string{"namespace", "job", "parallelism", "completions", "active_deadline_seconds", "start_time", "completion_time"}, nil,
	)
	descJobStatusSucceeded = prometheus.NewDesc(
		"kube_job_status_succeeded",
		"The number of pods which reached Phase Succeeded.",
		[]string{"namespace", "job"}, nil,
	)
	descJobStatusFailed = prometheus.NewDesc(
		"kube_job_status_failed",
		"The number of pods which reached Phase Failed.",
		[]string{"namespace", "job"}, nil,
	)
	descJobStatusActive = prometheus.NewDesc(
		"kube_job_status_active",
		"The number of actively running pods.",
		[]string{"namespace", "job"}, nil,
	)
	descJobConditionComplete = prometheus.NewDesc(
		"kube_job_complete",
		"The job has completed its execution.",
		[]string{"namespace", "job", "condition"}, nil,
	)
	descJobConditionFailed = prometheus.NewDesc(
		"kube_job_failed",
		"The job has failed its execution.",
		[]string{"namespace", "job", "condition"}, nil,
	)
)

type JobLister func() ([]v1batch.Job, error)

func (l JobLister) List() ([]v1batch.Job, error) {
	return l()
}

func RegisterJobCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface) {
	client := kubeClient.BatchV1().RESTClient()
	jlw := cache.NewListWatchFromClient(client, "jobs", api.NamespaceAll, nil)
	jinf := cache.NewSharedInformer(jlw, &v1batch.Job{}, resyncPeriod)

	jobLister := JobLister(func() (jobs []v1batch.Job, err error) {
		for _, c := range jinf.GetStore().List() {
			jobs = append(jobs, *(c.(*v1batch.Job)))
		}
		return jobs, nil
	})

	registry.MustRegister(&jobCollector{store: jobLister})
	go jinf.Run(context.Background().Done())
}

type jobStore interface {
	List() (jobs []v1batch.Job, err error)
}

// jobCollector collects metrics about all jobs in the cluster.
type jobCollector struct {
	store jobStore
}

// Describe implements the prometheus.Collector interface.
func (dc *jobCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descJobInfo
	ch <- descJobStatusSucceeded
	ch <- descJobStatusFailed
	ch <- descJobStatusActive
	ch <- descJobConditionComplete
	ch <- descJobConditionFailed
}

// Collect implements the prometheus.Collector interface.
func (jc *jobCollector) Collect(ch chan<- prometheus.Metric) {
	jobs, err := jc.store.List()
	if err != nil {
		glog.Errorf("listing jobs failed: %s", err)
		return
	}
	for _, j := range jobs {
		jc.collectJob(ch, j)
	}
}

func (jc *jobCollector) collectJob(ch chan<- prometheus.Metric, j v1batch.Job) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{j.Namespace, j.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	var completionTime string
	if j.Status.CompletionTime == nil {
		completionTime = ""
	} else {
		completionTime = strconv.FormatInt(j.Status.CompletionTime.Unix(), 10)
	}

	var activeDeadlineSeconds string
	if j.Spec.ActiveDeadlineSeconds == nil {
		activeDeadlineSeconds = ""
	} else {
		activeDeadlineSeconds = strconv.FormatInt(*j.Spec.ActiveDeadlineSeconds, 10)
	}

	addGauge(descJobInfo, 1,
		strconv.FormatInt(int64(*j.Spec.Parallelism), 10),
		strconv.FormatInt(int64(*j.Spec.Completions), 10),
		activeDeadlineSeconds,
		strconv.FormatInt(j.Status.StartTime.Unix(), 10),
		completionTime)

	addGauge(descJobStatusSucceeded, float64(j.Status.Succeeded))
	addGauge(descJobStatusFailed, float64(j.Status.Failed))
	addGauge(descJobStatusActive, float64(j.Status.Active))

	for _, c := range j.Status.Conditions {
		switch c.Type {
			case v1batch.JobComplete:
				addConditionMetrics(ch, descJobConditionComplete, c.Status, j.Namespace, j.Name)
			case v1batch.JobFailed:
				addConditionMetrics(ch, descJobConditionFailed, c.Status, j.Namespace, j.Name)
		}
	}
}
