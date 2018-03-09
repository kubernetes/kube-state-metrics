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

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	SuspendTrue                bool  = true
	SuspendFalse               bool  = false
	StartingDeadlineSeconds300 int64 = 300

	ActiveRunningCronJob1LastScheduleTime          = time.Unix(1500000000, 0)
	SuspendedCronJob1LastScheduleTime              = time.Unix(1500000000+5.5*3600, 0) // 5.5 hours later
	ActiveCronJob1NoLastScheduledCreationTimestamp = time.Unix(1500000000+6.5*3600, 0)
)

type mockCronJobStore struct {
	f func() ([]batchv1beta1.CronJob, error)
}

func (cjs mockCronJobStore) List() (cronJobs []batchv1beta1.CronJob, err error) {
	return cjs.f()
}

func TestCronJobCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_cronjob_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_cronjob_labels gauge
		# HELP kube_cronjob_info Info about cronjob.
		# TYPE kube_cronjob_info gauge
		# HELP kube_cronjob_created Unix creation timestamp
		# TYPE kube_cronjob_created gauge
		# HELP kube_cronjob_spec_starting_deadline_seconds Deadline in seconds for starting the job if it misses scheduled time for any reason.
		# TYPE kube_cronjob_spec_starting_deadline_seconds gauge
		# HELP kube_cronjob_spec_suspend Suspend flag tells the controller to suspend subsequent executions.
		# TYPE kube_cronjob_spec_suspend gauge
		# HELP kube_cronjob_status_active Active holds pointers to currently running jobs.
		# TYPE kube_cronjob_status_active gauge
		# HELP kube_cronjob_status_last_schedule_time LastScheduleTime keeps information of when was the last time the job was successfully scheduled.
		# TYPE kube_cronjob_status_last_schedule_time gauge
		# HELP kube_cronjob_next_schedule_time Next time the cronjob should be scheduled. The time after lastScheduleTime, or after the cron job's creation time if it's never been scheduled. Use this to determine if the job is delayed.
		# TYPE kube_cronjob_next_schedule_time gauge
	`
	cases := []struct {
		cronJobs []batchv1beta1.CronJob
		want     string
	}{
		{
			cronJobs: []batchv1beta1.CronJob{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "ActiveRunningCronJob1",
						Namespace:  "ns1",
						Generation: 1,
						Labels: map[string]string{
							"app": "example-active-running-1",
						},
					},
					Status: batchv1beta1.CronJobStatus{
						Active:           []v1.ObjectReference{v1.ObjectReference{Name: "FakeJob1"}, v1.ObjectReference{Name: "FakeJob2"}},
						LastScheduleTime: &metav1.Time{Time: ActiveRunningCronJob1LastScheduleTime},
					},
					Spec: batchv1beta1.CronJobSpec{
						StartingDeadlineSeconds: &StartingDeadlineSeconds300,
						ConcurrencyPolicy:       "Forbid",
						Suspend:                 &SuspendFalse,
						Schedule:                "0 */6 * * * *",
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:       "SuspendedCronJob1",
						Namespace:  "ns1",
						Generation: 1,
						Labels: map[string]string{
							"app": "example-suspended-1",
						},
					},
					Status: batchv1beta1.CronJobStatus{
						Active:           []v1.ObjectReference{},
						LastScheduleTime: &metav1.Time{Time: SuspendedCronJob1LastScheduleTime},
					},
					Spec: batchv1beta1.CronJobSpec{
						StartingDeadlineSeconds: &StartingDeadlineSeconds300,
						ConcurrencyPolicy:       "Forbid",
						Suspend:                 &SuspendTrue,
						Schedule:                "0 */3 * * * *",
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:              "ActiveCronJob1NoLastScheduled",
						CreationTimestamp: metav1.Time{Time: ActiveCronJob1NoLastScheduledCreationTimestamp},
						Namespace:         "ns1",
						Generation:        1,
						Labels: map[string]string{
							"app": "example-active-no-last-scheduled-1",
						},
					},
					Status: batchv1beta1.CronJobStatus{
						Active:           []v1.ObjectReference{},
						LastScheduleTime: nil,
					},
					Spec: batchv1beta1.CronJobSpec{
						StartingDeadlineSeconds: &StartingDeadlineSeconds300,
						ConcurrencyPolicy:       "Forbid",
						Suspend:                 &SuspendFalse,
						Schedule:                "25 * * * * *",
					},
				},
			},
			want: metadata + `
				kube_cronjob_created{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 1.5000234e+09

				kube_cronjob_info{concurrency_policy="Forbid",cronjob="ActiveRunningCronJob1",namespace="ns1",schedule="0 */6 * * * *"} 1
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="SuspendedCronJob1",namespace="ns1",schedule="0 */3 * * * *"} 1
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1",schedule="25 * * * * *"} 1

				kube_cronjob_labels{cronjob="ActiveCronJob1NoLastScheduled",label_app="example-active-no-last-scheduled-1",namespace="ns1"} 1
				kube_cronjob_labels{cronjob="ActiveRunningCronJob1",label_app="example-active-running-1",namespace="ns1"} 1
				kube_cronjob_labels{cronjob="SuspendedCronJob1",label_app="example-suspended-1",namespace="ns1"} 1

				kube_cronjob_next_schedule_time{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 1.500023425e+09
				kube_cronjob_next_schedule_time{cronjob="ActiveRunningCronJob1",namespace="ns1"} 1.50000012e+09

				kube_cronjob_spec_starting_deadline_seconds{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 300
				kube_cronjob_spec_starting_deadline_seconds{cronjob="ActiveRunningCronJob1",namespace="ns1"} 300
				kube_cronjob_spec_starting_deadline_seconds{cronjob="SuspendedCronJob1",namespace="ns1"} 300

				kube_cronjob_spec_suspend{cronjob="ActiveRunningCronJob1",namespace="ns1"} 0
				kube_cronjob_spec_suspend{cronjob="SuspendedCronJob1",namespace="ns1"} 1
				kube_cronjob_spec_suspend{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 0

				kube_cronjob_status_active{cronjob="ActiveRunningCronJob1",namespace="ns1"} 2
				kube_cronjob_status_active{cronjob="SuspendedCronJob1",namespace="ns1"} 0
				kube_cronjob_status_active{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 0

				kube_cronjob_status_last_schedule_time{cronjob="ActiveRunningCronJob1",namespace="ns1"} 1.5e+09
				kube_cronjob_status_last_schedule_time{cronjob="SuspendedCronJob1",namespace="ns1"} 1.5000198e+09
			`,
		},
	}
	for _, c := range cases {
		cjc := &cronJobCollector{
			store: mockCronJobStore{
				f: func() ([]batchv1beta1.CronJob, error) { return c.cronJobs, nil },
			},
		}
		if err := gatherAndCompare(cjc, c.want, nil); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
