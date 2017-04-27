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
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	v1batch "k8s.io/client-go/pkg/apis/batch/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	descJobStatusSucceeded = prometheus.NewDesc(
		"kube_job_status_succeeded",
		"The number of pods which reached Phase Succeeded.",
		[]string{"namespace", "job_name", "created_by"}, nil,
	)
	descJobStatusFailed = prometheus.NewDesc(
		"kube_job_status_failed",
		"The number of pods which reached Phase Failed.",
		[]string{"namespace", "job_name", "created_by"}, nil,
	)
	descJobStatusActive = prometheus.NewDesc(
		"kube_job_status_active",
		"The number of actively running pods.",
		[]string{"namespace", "job_name", "created_by"}, nil,
	)
	descJobStatusConditionComplete = prometheus.NewDesc(
		"kube_job_status_condition_complete",
		"The job has completed its execution.",
		[]string{"namespace", "job_name", "created_by"}, nil,
	)
	descJobStatusConditionFailed = prometheus.NewDesc(
		"kube_job_status_condition_failed",
		"The job has failed its execution.",
		[]string{"namespace", "job_name", "created_by"}, nil,
	)
	descJobStatusStartTime = prometheus.NewDesc(
		"kube_job_status_start_time",
		"The time when the job was acknowledged by the Job Manager.",
		[]string{"namespace", "job_name", "created_by"}, nil,
	)
	descJobStatusCompletionTime = prometheus.NewDesc(
		"kube_job_status_completion_time",
		"Time when the job was completed.",
		[]string{"namespace", "job_name", "created_by"}, nil,
	)
	descJobSpecCompletions = prometheus.NewDesc(
		"kube_job_spec_completions",
		"The desired number of successfully finished pods.",
		[]string{"namespace", "job_name", "created_by"}, nil,
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
	ch <- descJobStatusSucceeded
	ch <- descJobStatusFailed
	ch <- descJobStatusActive
	ch <- descJobStatusConditionComplete
	ch <- descJobStatusConditionFailed
	ch <- descJobStatusStartTime
	ch <- descJobStatusCompletionTime
	ch <- descJobSpecCompletions
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
	addCounter := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{j.Namespace, j.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, v, lv...)
	}

    created_by     := extractCreatedBy(j.Annotations)

	addGauge(descJobStatusSucceeded, float64(j.Status.Succeeded), created_by)
	addGauge(descJobStatusFailed, float64(j.Status.Failed), created_by)
	addGauge(descJobStatusActive, float64(j.Status.Active), created_by)
	addCounter(descJobStatusStartTime, float64(j.Status.StartTime.Unix()), created_by)
	addCounter(descJobStatusCompletionTime, float64(j.Status.CompletionTime.Unix()), created_by)
	addGauge(descJobSpecCompletions, float64(*j.Spec.Completions), created_by)

	foundCondition := false
	for _, jc := range j.Status.Conditions {
		if jc.Type == v1batch.JobComplete {
			addGauge(descJobStatusConditionComplete, float64(1), created_by)
			addGauge(descJobStatusConditionFailed, float64(0), created_by)
			foundCondition = true
			break
		}

		if jc.Type == v1batch.JobFailed {
			addGauge(descJobStatusConditionComplete, float64(0), created_by)
			addGauge(descJobStatusConditionFailed, float64(1), created_by)
			foundCondition = true
			break
		}
	}

	if !foundCondition {
		addGauge(descJobStatusConditionComplete, float64(0), created_by)
		addGauge(descJobStatusConditionFailed, float64(0), created_by)
	}
}
