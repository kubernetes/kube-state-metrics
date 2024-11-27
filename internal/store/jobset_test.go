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
	"testing"

	jobsetv1alpha2 "sigs.k8s.io/jobset/api/jobset/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestJobSetStore(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_jobset_annotations Kubernetes annotations converted to Prometheus labels.
		# TYPE kube_jobset_annotations gauge
		# HELP kube_jobset_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_jobset_labels gauge
		# HELP kube_jobset_specified_replicas The Number of specified replicas per replicated jobs in a jobset.
		# TYPE kube_jobset_specified_replicas gauge
		# HELP kube_jobset_status_replicas The Number of replicas in ready/succeeded/failed/active/suspended status per replicated jobs in a jobset.
		# TYPE kube_jobset_status_replicas gauge
		# HELP kube_jobset_status_condition The current status conditions of a jobset.
		# TYPE kube_jobset_status_condition gauge
		`

	cases := []generateMetricsTestCase{
		{
			Obj: &jobsetv1alpha2.JobSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "RunningJobSet1",
					Namespace:         "ns1",
					Labels: map[string]string{
						"app": "example-running-1",
					},
				},
				Status: jobsetv1alpha2.JobSetStatus{
					ReplicatedJobsStatus: []jobsetv1alpha2.ReplicatedJobStatus{
						{
							Name:      "replicated-jobs-1",
							Ready:     1,
							Succeeded: 2,
							Failed:    3,
							Active:    4,
							Suspended: 5,

						},
						{
							Name:      "replicated-jobs-2",
							Ready:     6,
							Succeeded: 7,
							Failed:    8,
							Active:    9,
							Suspended: 10,
						},
					},
					Conditions: []metav1.Condition{
						{Type: string(jobsetv1alpha2.JobSetCompleted), Status: metav1.ConditionTrue},
						{Type: string(jobsetv1alpha2.JobSetFailed), Status: metav1.ConditionTrue},
					},
				},
				Spec: jobsetv1alpha2.JobSetSpec{
					ReplicatedJobs: []jobsetv1alpha2.ReplicatedJob{
						{
							Name:     "replicated-jobs-1",
							Replicas: 20,
						},
						{
							Name:     "replicated-jobs-2",
							Replicas: 30,
						},
					},
				},
			},
			Want: metadata + `
				kube_jobset_specified_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-1"} 20
				kube_jobset_specified_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-2"} 30
				kube_jobset_status_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-1",status="ready"} 1
				kube_jobset_status_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-2",status="ready"} 6
				kube_jobset_status_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-1",status="succeeded"} 2
				kube_jobset_status_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-2",status="succeeded"} 7
				kube_jobset_status_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-1",status="failed"} 3
				kube_jobset_status_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-2",status="failed"} 8
				kube_jobset_status_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-1",status="active"} 4
				kube_jobset_status_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-2",status="active"} 9
				kube_jobset_status_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-1",status="suspended"} 5
				kube_jobset_status_replicas{jobset_name="RunningJobSet1",namespace="ns1",replicated_job_name="replicated-jobs-2",status="suspended"} 10
				kube_jobset_status_condition{jobset_name="RunningJobSet1",namespace="ns1",condition="Completed",status="true"} 1
				kube_jobset_status_condition{jobset_name="RunningJobSet1",namespace="ns1",condition="Failed",status="true"} 1
				kube_jobset_status_condition{jobset_name="RunningJobSet1",namespace="ns1",condition="Completed",status="false"} 0
				kube_jobset_status_condition{jobset_name="RunningJobSet1",namespace="ns1",condition="Failed",status="false"} 0
				kube_jobset_status_condition{jobset_name="RunningJobSet1",namespace="ns1",condition="Completed",status="unknown"} 0
				kube_jobset_status_condition{jobset_name="RunningJobSet1",namespace="ns1",condition="Failed",status="unknown"} 0
`,
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(jobSetMetricFamilies(nil, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(jobSetMetricFamilies(nil, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
