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
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	v1batch "k8s.io/api/batch/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	descJobLabelsName          = "kube_job_labels"
	descJobLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descJobLabelsDefaultLabels = []string{"namespace", "job_name"}

	descJobLabels = prometheus.NewDesc(
		descJobLabelsName,
		descJobLabelsHelp,
		descJobLabelsDefaultLabels,
		nil,
	)

	descJobInfo = prometheus.NewDesc(
		"kube_job_info",
		"Information about job.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobCreated = prometheus.NewDesc(
		"kube_job_created",
		"Unix creation timestamp",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobSpecParallelism = prometheus.NewDesc(
		"kube_job_spec_parallelism",
		"The maximum desired number of pods the job should run at any given time.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobSpecCompletions = prometheus.NewDesc(
		"kube_job_spec_completions",
		"The desired number of successfully finished pods the job should be run with.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobSpecActiveDeadlineSeconds = prometheus.NewDesc(
		"kube_job_spec_active_deadline_seconds",
		"The duration in seconds relative to the startTime that the job may be active before the system tries to terminate it.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobStatusSucceeded = prometheus.NewDesc(
		"kube_job_status_succeeded",
		"The number of pods which reached Phase Succeeded.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobStatusFailed = prometheus.NewDesc(
		"kube_job_status_failed",
		"The number of pods which reached Phase Failed.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobStatusActive = prometheus.NewDesc(
		"kube_job_status_active",
		"The number of actively running pods.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobConditionComplete = prometheus.NewDesc(
		"kube_job_complete",
		"The job has completed its execution.",
		append(descJobLabelsDefaultLabels, "condition"),
		nil,
	)
	descJobConditionFailed = prometheus.NewDesc(
		"kube_job_failed",
		"The job has failed its execution.",
		append(descJobLabelsDefaultLabels, "condition"),
		nil,
	)
	descJobStatusStartTime = prometheus.NewDesc(
		"kube_job_status_start_time",
		"StartTime represents time when the job was acknowledged by the Job Manager.",
		descJobLabelsDefaultLabels,
		nil,
	)
	descJobStatusCompletionTime = prometheus.NewDesc(
		"kube_job_status_completion_time",
		"CompletionTime represents time when the job was completed.",
		descJobLabelsDefaultLabels,
		nil,
	)
)

type JobLister func() ([]v1batch.Job, error)

func (l JobLister) List() ([]v1batch.Job, error) {
	return l()
}

func RegisterJobCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Batch().V1().Jobs().Informer().(cache.SharedInformer))
	}

	jobLister := JobLister(func() (jobs []v1batch.Job, err error) {
		for _, jinf := range infs {
			for _, c := range jinf.GetStore().List() {
				jobs = append(jobs, *(c.(*v1batch.Job)))
			}
		}
		return jobs, nil
	})

	registry.MustRegister(&jobCollector{store: jobLister, opts: opts})
	infs.Run(context.Background().Done())
}

type jobStore interface {
	List() (jobs []v1batch.Job, err error)
}

// jobCollector collects metrics about all jobs in the cluster.
type jobCollector struct {
	store jobStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (dc *jobCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descJobInfo
	ch <- descJobCreated
	ch <- descJobLabels
	ch <- descJobSpecParallelism
	ch <- descJobSpecCompletions
	ch <- descJobSpecActiveDeadlineSeconds
	ch <- descJobStatusSucceeded
	ch <- descJobStatusFailed
	ch <- descJobStatusActive
	ch <- descJobConditionComplete
	ch <- descJobConditionFailed
	ch <- descJobStatusStartTime
	ch <- descJobStatusCompletionTime
}

// Collect implements the prometheus.Collector interface.
func (jc *jobCollector) Collect(ch chan<- prometheus.Metric) {
	jobs, err := jc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "job"}).Inc()
		glog.Errorf("listing jobs failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "job"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "job"}).Observe(float64(len(jobs)))
	for _, j := range jobs {
		jc.collectJob(ch, j)
	}

	glog.V(4).Infof("collected %d jobs", len(jobs))
}

func jobLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descJobLabelsName,
		descJobLabelsHelp,
		append(descJobLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func (jc *jobCollector) collectJob(ch chan<- prometheus.Metric, j v1batch.Job) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{j.Namespace, j.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
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
			addConditionMetrics(ch, descJobConditionComplete, c.Status, j.Namespace, j.Name)
		case v1batch.JobFailed:
			addConditionMetrics(ch, descJobConditionFailed, c.Status, j.Namespace, j.Name)
		}
	}
}
