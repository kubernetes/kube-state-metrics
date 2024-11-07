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
	"time"

	v1batch "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
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

func TestJobStore(t *testing.T) {
	var trueValue = true
	var falseValue = false

	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_job_annotations Kubernetes annotations converted to Prometheus labels.
		# TYPE kube_job_annotations gauge
		# HELP kube_job_created [STABLE] Unix creation timestamp
		# TYPE kube_job_created gauge
		# HELP kube_job_owner [STABLE] Information about the Job's owner.
		# TYPE kube_job_owner gauge
		# HELP kube_job_complete [STABLE] The job has completed its execution.
		# TYPE kube_job_complete gauge
		# HELP kube_job_failed [STABLE] The job has failed its execution.
		# TYPE kube_job_failed gauge
		# HELP kube_job_info [STABLE] Information about job.
		# TYPE kube_job_info gauge
		# HELP kube_job_labels [STABLE] Kubernetes labels converted to Prometheus labels.
		# TYPE kube_job_labels gauge
		# HELP kube_job_spec_active_deadline_seconds [STABLE] The duration in seconds relative to the startTime that the job may be active before the system tries to terminate it.
		# TYPE kube_job_spec_active_deadline_seconds gauge
		# HELP kube_job_spec_completions [STABLE] The desired number of successfully finished pods the job should be run with.
		# TYPE kube_job_spec_completions gauge
		# HELP kube_job_spec_parallelism [STABLE] The maximum desired number of pods the job should run at any given time.
		# TYPE kube_job_spec_parallelism gauge
		# HELP kube_job_status_active [STABLE] The number of actively running pods.
		# TYPE kube_job_status_active gauge
		# HELP kube_job_status_completion_time [STABLE] CompletionTime represents time when the job was completed.
		# TYPE kube_job_status_completion_time gauge
		# HELP kube_job_status_failed [STABLE] The number of pods which reached Phase Failed and the reason for failure.
		# TYPE kube_job_status_failed gauge
		# HELP kube_job_status_start_time [STABLE] StartTime represents time when the job was acknowledged by the Job Manager.
		# TYPE kube_job_status_start_time gauge
		# HELP kube_job_status_succeeded [STABLE] The number of pods which reached Phase Succeeded.
		# TYPE kube_job_status_succeeded gauge
		# HELP kube_job_status_suspended The number of pods which reached Phase Suspended.
		# TYPE kube_job_status_suspended gauge
		`

	cases := []generateMetricsTestCase{
		{
			Obj: &v1batch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "RunningJob1",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "ns1",
					Generation:        1,
					Labels: map[string]string{
						"app": "example-running-1",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind:       "CronJob",
							Name:       "cronjob-name",
							Controller: &trueValue,
						},
					},
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
			},
			Want: metadata + `
				kube_job_owner{job_name="RunningJob1",namespace="ns1",owner_is_controller="true",owner_kind="CronJob",owner_name="cronjob-name"} 1
				kube_job_created{job_name="RunningJob1",namespace="ns1"} 1.5e+09
				kube_job_info{job_name="RunningJob1",namespace="ns1"} 1
				kube_job_spec_active_deadline_seconds{job_name="RunningJob1",namespace="ns1"} 900
				kube_job_spec_completions{job_name="RunningJob1",namespace="ns1"} 1
				kube_job_spec_parallelism{job_name="RunningJob1",namespace="ns1"} 1
				kube_job_status_active{job_name="RunningJob1",namespace="ns1"} 1
				kube_job_status_failed{job_name="RunningJob1",namespace="ns1"} 0
				kube_job_status_start_time{job_name="RunningJob1",namespace="ns1"} 1.495800007e+09
				kube_job_status_succeeded{job_name="RunningJob1",namespace="ns1"} 0
`,
		},
		{
			Obj: &v1batch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "SuccessfulJob1",
					Namespace:  "ns1",
					Generation: 1,
					Labels: map[string]string{
						"app": "example-successful-1",
					},
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
			},
			Want: metadata + `
				kube_job_owner{job_name="SuccessfulJob1",namespace="ns1",owner_is_controller="",owner_kind="",owner_name=""} 1
				kube_job_complete{condition="false",job_name="SuccessfulJob1",namespace="ns1"} 0
				kube_job_complete{condition="true",job_name="SuccessfulJob1",namespace="ns1"} 1
				kube_job_complete{condition="unknown",job_name="SuccessfulJob1",namespace="ns1"} 0
				kube_job_info{job_name="SuccessfulJob1",namespace="ns1"} 1
				kube_job_spec_active_deadline_seconds{job_name="SuccessfulJob1",namespace="ns1"} 900
				kube_job_spec_completions{job_name="SuccessfulJob1",namespace="ns1"} 1
				kube_job_spec_parallelism{job_name="SuccessfulJob1",namespace="ns1"} 1
				kube_job_status_active{job_name="SuccessfulJob1",namespace="ns1"} 0
				kube_job_status_completion_time{job_name="SuccessfulJob1",namespace="ns1"} 1.495803607e+09
				kube_job_status_failed{job_name="SuccessfulJob1",namespace="ns1"} 0
				kube_job_status_start_time{job_name="SuccessfulJob1",namespace="ns1"} 1.495800007e+09
				kube_job_status_succeeded{job_name="SuccessfulJob1",namespace="ns1"} 1
`,
		},
		{
			Obj: &v1batch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "FailedJob1",
					Namespace:  "ns1",
					Generation: 1,
					Labels: map[string]string{
						"app": "example-failed-1",
					},
				},
				Status: v1batch.JobStatus{
					Active:         0,
					Failed:         1,
					Succeeded:      0,
					CompletionTime: &metav1.Time{Time: FailedJob1CompletionTime},
					StartTime:      &metav1.Time{Time: FailedJob1StartTime},
					Conditions: []v1batch.JobCondition{
						{Type: v1batch.JobFailed, Status: v1.ConditionTrue, Reason: "BackoffLimitExceeded"},
					},
				},
				Spec: v1batch.JobSpec{
					ActiveDeadlineSeconds: &ActiveDeadlineSeconds900,
					Parallelism:           &Parallelism1,
					Completions:           &Completions1,
				},
			},
			Want: metadata + `
				kube_job_owner{job_name="FailedJob1",namespace="ns1",owner_is_controller="",owner_kind="",owner_name=""} 1
				kube_job_failed{condition="false",job_name="FailedJob1",namespace="ns1"} 0
				kube_job_failed{condition="true",job_name="FailedJob1",namespace="ns1"} 1
				kube_job_failed{condition="unknown",job_name="FailedJob1",namespace="ns1"} 0
				kube_job_info{job_name="FailedJob1",namespace="ns1"} 1
				kube_job_spec_active_deadline_seconds{job_name="FailedJob1",namespace="ns1"} 900
				kube_job_spec_completions{job_name="FailedJob1",namespace="ns1"} 1
				kube_job_spec_parallelism{job_name="FailedJob1",namespace="ns1"} 1
				kube_job_status_active{job_name="FailedJob1",namespace="ns1"} 0
				kube_job_status_completion_time{job_name="FailedJob1",namespace="ns1"} 1.495810807e+09
				kube_job_status_failed{job_name="FailedJob1",namespace="ns1",reason="BackoffLimitExceeded"} 1
				kube_job_status_failed{job_name="FailedJob1",namespace="ns1",reason="DeadlineExceeded"} 0
				kube_job_status_failed{job_name="FailedJob1",namespace="ns1",reason="Evicted"} 0
				kube_job_status_start_time{job_name="FailedJob1",namespace="ns1"} 1.495807207e+09
				kube_job_status_succeeded{job_name="FailedJob1",namespace="ns1"} 0
`,
		},
		{
			Obj: &v1batch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "FailedJobWithNoConditions",
					Namespace: "ns1",
				},
				Status: v1batch.JobStatus{
					Failed: 1,
				},
				Spec: v1batch.JobSpec{
					ActiveDeadlineSeconds: &ActiveDeadlineSeconds900,
				},
			},
			Want: metadata + `
				kube_job_owner{job_name="FailedJobWithNoConditions",namespace="ns1",owner_is_controller="",owner_kind="",owner_name=""} 1
				kube_job_info{job_name="FailedJobWithNoConditions",namespace="ns1"} 1
				kube_job_spec_active_deadline_seconds{job_name="FailedJobWithNoConditions",namespace="ns1"} 900
				kube_job_status_active{job_name="FailedJobWithNoConditions",namespace="ns1"} 0
				kube_job_status_failed{job_name="FailedJobWithNoConditions",namespace="ns1",reason=""} 1
				kube_job_status_succeeded{job_name="FailedJobWithNoConditions",namespace="ns1"} 0
`,
		},
		{
			Obj: &v1batch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "SuccessfulJob2NoActiveDeadlineSeconds",
					Namespace:  "ns1",
					Generation: 1,
					Labels: map[string]string{
						"app": "example-successful-2",
					},
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
			Want: metadata + `
				kube_job_owner{job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1",owner_is_controller="",owner_kind="",owner_name=""} 1
				kube_job_complete{condition="false",job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 0
				kube_job_complete{condition="true",job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1

				kube_job_complete{condition="unknown",job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 0
				kube_job_info{job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1
				kube_job_spec_completions{job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1
				kube_job_spec_parallelism{job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1
				kube_job_status_active{job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 0
				kube_job_status_completion_time{job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1.495804207e+09
				kube_job_status_failed{job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 0
				kube_job_status_start_time{job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1.495800607e+09
				kube_job_status_succeeded{job_name="SuccessfulJob2NoActiveDeadlineSeconds",namespace="ns1"} 1
`,
		},
		{
			Obj: &v1batch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "SuspendedNoActiveDeadlineSeconds",
					Namespace:  "ns1",
					Generation: 1,
				},
				Status: v1batch.JobStatus{
					Active:    0,
					Failed:    0,
					Succeeded: 0,
					StartTime: &metav1.Time{Time: SuccessfulJob2StartTime},
					Conditions: []v1batch.JobCondition{
						{Type: v1batch.JobSuspended, Status: v1.ConditionTrue},
					},
				},
				Spec: v1batch.JobSpec{
					Suspend:     &trueValue,
					Parallelism: &Parallelism1,
					Completions: &Completions1,
				},
			},
			Want: metadata + `
				kube_job_owner{job_name="SuspendedNoActiveDeadlineSeconds",namespace="ns1",owner_is_controller="",owner_kind="",owner_name=""} 1
				kube_job_info{job_name="SuspendedNoActiveDeadlineSeconds",namespace="ns1"} 1
				kube_job_spec_completions{job_name="SuspendedNoActiveDeadlineSeconds",namespace="ns1"} 1
				kube_job_spec_parallelism{job_name="SuspendedNoActiveDeadlineSeconds",namespace="ns1"} 1
				kube_job_status_active{job_name="SuspendedNoActiveDeadlineSeconds",namespace="ns1"} 0
				kube_job_status_failed{job_name="SuspendedNoActiveDeadlineSeconds",namespace="ns1"} 0
				kube_job_status_start_time{job_name="SuspendedNoActiveDeadlineSeconds",namespace="ns1"} 1.495800607e+09
				kube_job_status_succeeded{job_name="SuspendedNoActiveDeadlineSeconds",namespace="ns1"} 0
                kube_job_status_suspended{job_name="SuspendedNoActiveDeadlineSeconds",namespace="ns1"} 1
`,
		},
		{
			Obj: &v1batch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "UnsuspendedNoActiveDeadlineSeconds",
					Namespace:  "ns1",
					Generation: 1,
				},
				Status: v1batch.JobStatus{
					Active:    0,
					Failed:    0,
					Succeeded: 0,
					StartTime: &metav1.Time{Time: SuccessfulJob2StartTime},
					Conditions: []v1batch.JobCondition{
						{Type: v1batch.JobSuspended, Status: v1.ConditionFalse},
					},
				},
				Spec: v1batch.JobSpec{
					Suspend:     &falseValue,
					Parallelism: &Parallelism1,
					Completions: &Completions1,
				},
			},
			Want: metadata + `
				kube_job_owner{job_name="UnsuspendedNoActiveDeadlineSeconds",namespace="ns1",owner_is_controller="",owner_kind="",owner_name=""} 1
				kube_job_info{job_name="UnsuspendedNoActiveDeadlineSeconds",namespace="ns1"} 1
				kube_job_spec_completions{job_name="UnsuspendedNoActiveDeadlineSeconds",namespace="ns1"} 1
				kube_job_spec_parallelism{job_name="UnsuspendedNoActiveDeadlineSeconds",namespace="ns1"} 1
				kube_job_status_active{job_name="UnsuspendedNoActiveDeadlineSeconds",namespace="ns1"} 0
				kube_job_status_failed{job_name="UnsuspendedNoActiveDeadlineSeconds",namespace="ns1"} 0
				kube_job_status_start_time{job_name="UnsuspendedNoActiveDeadlineSeconds",namespace="ns1"} 1.495800607e+09
				kube_job_status_succeeded{job_name="UnsuspendedNoActiveDeadlineSeconds",namespace="ns1"} 0
                kube_job_status_suspended{job_name="UnsuspendedNoActiveDeadlineSeconds",namespace="ns1"} 0
`,
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(jobMetricFamilies(nil, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(jobMetricFamilies(nil, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
