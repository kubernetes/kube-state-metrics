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
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	v1batch "k8s.io/client-go/pkg/apis/batch/v1"
)

var (
	Parallelism1             int32 = 1
	Completions1             int32 = 1
	ActiveDeadlineSeconds900 int64 = 900

	RunningJob1StartTime, _    = time.Parse(time.RFC3339, "2017-05-26T12:00:07Z")
	SuccessfulJob1StartTime, _ = time.Parse(time.RFC3339, "2017-05-26T12:00:07Z")
	FailedJob1StartTime, _     = time.Parse(time.RFC3339, "2017-05-26T14:00:07Z")
	SuccessfulJob2StartTime, _ = time.Parse(time.RFC3339, "2017-05-26T12:10:07Z")

	SuccessfulJob1CompletionTime, _ = time.Parse(time.RFC3339, "2017-05-26T13:00:07Z")
	FailedJob1CompletionTime, _     = time.Parse(time.RFC3339, "2017-05-26T15:00:07Z")
	SuccessfulJob2CompletionTime, _ = time.Parse(time.RFC3339, "2017-05-26T13:10:07Z")
)

type mockJobStore struct {
	f func() ([]v1batch.Job, error)
}

func (js mockJobStore) List() (jobs []v1batch.Job, err error) {
	return js.f()
}

func TestJobCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_job_complete The job has completed its execution.
		# TYPE kube_job_complete gauge
		# HELP kube_job_failed The job has failed its execution.
		# TYPE kube_job_failed gauge
		# HELP kube_job_info Information about job.
		# TYPE kube_job_info gauge
		# HELP kube_job_spec_active_deadline_seconds The duration in seconds relative to the startTime that the job may be active before the system tries to terminate it.
		# TYPE kube_job_spec_active_deadline_seconds gauge
		# HELP kube_job_spec_completions The desired number of successfully finished pods the job should be run with.
		# TYPE kube_job_spec_completions gauge
		# HELP kube_job_spec_parallelism The maximum desired number of pods the job should run at any given time.
		# TYPE kube_job_spec_parallelism gauge
		# HELP kube_job_status_active The number of actively running pods.
		# TYPE kube_job_status_active gauge
		# HELP kube_job_status_completion_time CompletionTime represents time when the job was completed.
		# TYPE kube_job_status_completion_time counter
		# HELP kube_job_status_failed The number of pods which reached Phase Failed.
		# TYPE kube_job_status_failed gauge
		# HELP kube_job_status_start_time StartTime represents time when the job was acknowledged by the Job Manager.
		# TYPE kube_job_status_start_time counter
		# HELP kube_job_status_succeeded The number of pods which reached Phase Succeeded.
		# TYPE kube_job_status_succeeded gauge
	`
	cases := []struct {
		jobs []v1batch.Job
		want string
	}{
		{
			jobs: []v1batch.Job{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "RunningJob1",
						Namespace:  "ns1",
						Generation: 1,
					},
					Status: v1batch.JobStatus{
						Active:         1,
						Failed:         0,
						Succeeded:      0,
						CompletionTime: nil,
						StartTime:      &metav1.Time{Time: RunningJob1StartTime},
					},
					Spec: v1batch.JobSpec{
						ActiveDeadlineSeconds: &ActiveDeadlineSeconds900,
						Parallelism:           &Parallelism1,
						Completions:           &Completions1,
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:       "SuccessfulJob1",
						Namespace:  "ns1",
						Generation: 1,
					},
					Status: v1batch.JobStatus{
						Active:         0,
						Failed:         0,
						Succeeded:      1,
						CompletionTime: &metav1.Time{Time: SuccessfulJob1CompletionTime},
						StartTime:      &metav1.Time{Time: SuccessfulJob1StartTime},
						Conditions: []v1batch.JobCondition{
							{Type: v1batch.JobComplete, Status: v1.ConditionTrue},
						},
					},
					Spec: v1batch.JobSpec{
						ActiveDeadlineSeconds: &ActiveDeadlineSeconds900,
						Parallelism:           &Parallelism1,
						Completions:           &Completions1,
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:       "FailedJob1",
						Namespace:  "ns1",
						Generation: 1,
					},
					Status: v1batch.JobStatus{
						Active:         0,
						Failed:         1,
						Succeeded:      0,
						CompletionTime: &metav1.Time{Time: FailedJob1CompletionTime},
						StartTime:      &metav1.Time{Time: FailedJob1StartTime},
						Conditions: []v1batch.JobCondition{
							{Type: v1batch.JobFailed, Status: v1.ConditionTrue},
						},
					},
					Spec: v1batch.JobSpec{
						ActiveDeadlineSeconds: &ActiveDeadlineSeconds900,
						Parallelism:           &Parallelism1,
						Completions:           &Completions1,
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:       "SuccessfulJob2NoActiveDeadlineSeconds",
						Namespace:  "ns1",
						Generation: 1,
					},
					Status: v1batch.JobStatus{
						Active:         0,
						Failed:         0,
						Succeeded:      1,
						CompletionTime: &metav1.Time{Time: SuccessfulJob2CompletionTime},
						StartTime:      &metav1.Time{Time: SuccessfulJob2StartTime},
						Conditions: []v1batch.JobCondition{
							{Type: v1batch.JobComplete, Status: v1.ConditionTrue},
						},
					},
					Spec: v1batch.JobSpec{
						ActiveDeadlineSeconds: nil,
						Parallelism:           &Parallelism1,
						Completions:           &Completions1,
					},
				},
			},
			want: metadata + `
				kube_job_complete{condition="false",job="SuccessfulJob1",namespace="ns1"} 0
				kube_job_complete{condition="false",job="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 0

				kube_job_complete{condition="true",job="SuccessfulJob1",namespace="ns1"} 1
				kube_job_complete{condition="true",job="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1

				kube_job_complete{condition="unknown",job="SuccessfulJob1",namespace="ns1"} 0
				kube_job_complete{condition="unknown",job="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 0

				kube_job_failed{condition="false",job="FailedJob1",namespace="ns1"} 0

				kube_job_failed{condition="true",job="FailedJob1",namespace="ns1"} 1

				kube_job_failed{condition="unknown",job="FailedJob1",namespace="ns1"} 0

				kube_job_info{job="RunningJob1",namespace="ns1"} 1
				kube_job_info{job="SuccessfulJob1",namespace="ns1"} 1
				kube_job_info{job="FailedJob1",namespace="ns1"} 1
				kube_job_info{job="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1

				kube_job_spec_active_deadline_seconds{job="RunningJob1",namespace="ns1"} 900
				kube_job_spec_active_deadline_seconds{job="SuccessfulJob1",namespace="ns1"} 900
				kube_job_spec_active_deadline_seconds{job="FailedJob1",namespace="ns1"} 900

				kube_job_spec_completions{job="RunningJob1",namespace="ns1"} 1
				kube_job_spec_completions{job="SuccessfulJob1",namespace="ns1"} 1
				kube_job_spec_completions{job="FailedJob1",namespace="ns1"} 1
				kube_job_spec_completions{job="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1

				kube_job_spec_parallelism{job="RunningJob1",namespace="ns1"} 1
				kube_job_spec_parallelism{job="SuccessfulJob1",namespace="ns1"} 1
				kube_job_spec_parallelism{job="FailedJob1",namespace="ns1"} 1
				kube_job_spec_parallelism{job="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1

				kube_job_status_active{job="RunningJob1",namespace="ns1"} 1
				kube_job_status_active{job="SuccessfulJob1",namespace="ns1"} 0
				kube_job_status_active{job="FailedJob1",namespace="ns1"} 0
				kube_job_status_active{job="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 0

				kube_job_status_completion_time{job="SuccessfulJob1",namespace="ns1"} 1.495803607e+09
				kube_job_status_completion_time{job="FailedJob1",namespace="ns1"} 1.495810807e+09
				kube_job_status_completion_time{job="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1.495804207e+09

				kube_job_status_failed{job="RunningJob1",namespace="ns1"} 0
				kube_job_status_failed{job="SuccessfulJob1",namespace="ns1"} 0
				kube_job_status_failed{job="FailedJob1",namespace="ns1"} 1
				kube_job_status_failed{job="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 0

				kube_job_status_start_time{job="RunningJob1",namespace="ns1"} 1.495800007e+09
				kube_job_status_start_time{job="SuccessfulJob1",namespace="ns1"} 1.495800007e+09
				kube_job_status_start_time{job="FailedJob1",namespace="ns1"} 1.495807207e+09
				kube_job_status_start_time{job="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1.495800607e+09

				kube_job_status_succeeded{job="RunningJob1",namespace="ns1"} 0
				kube_job_status_succeeded{job="SuccessfulJob1",namespace="ns1"} 1
				kube_job_status_succeeded{job="FailedJob1",namespace="ns1"} 0
				kube_job_status_succeeded{job="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1
			`,
		},
	}
	for _, c := range cases {
		jc := &jobCollector{
			store: mockJobStore{
				f: func() ([]v1batch.Job, error) { return c.jobs, nil },
			},
		}
		if err := gatherAndCompare(jc, c.want, nil); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
