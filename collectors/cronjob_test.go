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

	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/api/unversioned"
	v2batch "k8s.io/client-go/pkg/apis/batch/v2alpha1"
)

func init() {
	// Mock time.Now() for `kube_cronjob_scheduling_delay`
	timeNow = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2017-05-26T18:08:03Z")
		return t
	}
}

var (
	SuspendTrue bool = true
	SuspendFalse bool = false
	StartingDeadlineSeconds300 int64 = 300

	ActiveRunningCronJob1LastScheduleTime, _ = time.Parse(time.RFC3339, "2017-05-26T12:00:07Z")
	SuspendedCronJob1LastScheduleTime, _ = time.Parse(time.RFC3339, "2017-05-26T17:30:00Z")
)

type mockCronJobStore struct {
	f func() ([]v2batch.CronJob, error)
}

func (cjs mockCronJobStore) List() (cronJobs []v2batch.CronJob, err error) {
	return cjs.f()
}

type delaytest struct {
	time     time.Time
	schedule string
	delay    float64
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func TestSchedulingDelay(t *testing.T) {
	var tests = []delaytest{
		{parseTime("2017-05-26T18:06:01Z"), "* * * * *", 63},
		{parseTime("2017-05-26T15:06:00Z"), "0 */3 * * *", 483},
	}
	for _, test := range tests {
		delay := getSchedulingDelaySeconds(test.schedule, test.time)
		if delay != test.delay {
			t.Errorf("Delay doesn't match. actual %d, expected %d. Schedule: %s, time: %s", delay, test.delay, test.schedule, test.time.String())
		}
	}
}

func TestCronJobCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_cronjob_info Info about cronjob.
		# TYPE kube_cronjob_info gauge
		# HELP kube_cronjob_spec_starting_deadline_seconds Deadline in seconds for starting the job if it misses scheduled time for any reason.
		# TYPE kube_cronjob_spec_starting_deadline_seconds gauge
		# HELP kube_cronjob_spec_suspend Suspend flag tells the controller to suspend subsequent executions.
		# TYPE kube_cronjob_spec_suspend gauge
		# HELP kube_cronjob_status_active Active holds pointers to currently running jobs.
		# TYPE kube_cronjob_status_active gauge
		# HELP kube_cronjob_status_last_schedule_time LastScheduleTime keeps information of when was the last time the job was successfully scheduled.
		# TYPE kube_cronjob_status_last_schedule_time counter
		# HELP kube_cronjob_scheduling_delay Number of seconds the cron job is delayed scheduling
		# TYPE kube_cronjob_scheduling_delay gauge
	`
	cases := []struct {
		cronJobs []v2batch.CronJob
		want  string
	}{
		{
			cronJobs: []v2batch.CronJob{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:       "ActiveRunningCronJob1",
						Namespace:  "ns1",
						Generation: 1,
					},
					Status: v2batch.CronJobStatus{
						Active:           	[]v1.ObjectReference{v1.ObjectReference{Name: "FakeJob1"}, v1.ObjectReference{Name: "FakeJob2"}},
						LastScheduleTime:	&unversioned.Time{Time: ActiveRunningCronJob1LastScheduleTime},
					},
					Spec: v2batch.CronJobSpec{
						StartingDeadlineSeconds:	&StartingDeadlineSeconds300,
						ConcurrencyPolicy:		"Forbid",
						Suspend:			&SuspendFalse,
						Schedule:			"0 */6 * * *",
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:       "SuspendedCronJob1",
						Namespace:  "ns1",
						Generation: 1,
					},
					Status: v2batch.CronJobStatus{
						Active:           	[]v1.ObjectReference{},
						LastScheduleTime:	&unversioned.Time{Time: SuspendedCronJob1LastScheduleTime},
					},
					Spec: v2batch.CronJobSpec{
						StartingDeadlineSeconds:	&StartingDeadlineSeconds300,
						ConcurrencyPolicy:		"Forbid",
						Suspend:			&SuspendTrue,
						Schedule:			"0 */3 * * *",
					},
				}, {
					ObjectMeta: v1.ObjectMeta{
						Name:       "ActiveCronJob1NoLastScheduled",
						Namespace:  "ns1",
						Generation: 1,
					},
					Status: v2batch.CronJobStatus{
						Active:           	[]v1.ObjectReference{},
						LastScheduleTime:	nil,
					},
					Spec: v2batch.CronJobSpec{
						StartingDeadlineSeconds:	&StartingDeadlineSeconds300,
						ConcurrencyPolicy:		"Forbid",
						Suspend:			&SuspendFalse,
						Schedule:			"25 * * * *",
					},
				},
			},
			want: metadata + `
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="ActiveRunningCronJob1",namespace="ns1",schedule="0 */6 * * *"} 1
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="SuspendedCronJob1",namespace="ns1",schedule="0 */3 * * *"} 1
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1",schedule="25 * * * *"} 1

				kube_cronjob_scheduling_delay{cronjob="ActiveRunningCronJob1",namespace="ns1"} 483

				kube_cronjob_spec_starting_deadline_seconds{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 300
				kube_cronjob_spec_starting_deadline_seconds{cronjob="ActiveRunningCronJob1",namespace="ns1"} 300
				kube_cronjob_spec_starting_deadline_seconds{cronjob="SuspendedCronJob1",namespace="ns1"} 300

				kube_cronjob_spec_suspend{cronjob="ActiveRunningCronJob1",namespace="ns1"} 0
				kube_cronjob_spec_suspend{cronjob="SuspendedCronJob1",namespace="ns1"} 1
				kube_cronjob_spec_suspend{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 0

				kube_cronjob_status_active{cronjob="ActiveRunningCronJob1",namespace="ns1"} 2
				kube_cronjob_status_active{cronjob="SuspendedCronJob1",namespace="ns1"} 0
				kube_cronjob_status_active{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 0

				kube_cronjob_status_last_schedule_time{cronjob="ActiveRunningCronJob1",namespace="ns1"} 1.495800007e+09
				kube_cronjob_status_last_schedule_time{cronjob="SuspendedCronJob1",namespace="ns1"} 1.4958198e+09
			`,
		},
	}
	for _, c := range cases {
		cjc := &cronJobCollector{
			store: mockCronJobStore{
				f: func() ([]v2batch.CronJob, error) { return c.cronJobs, nil },
			},
		}
		if err := gatherAndCompare(cjc, c.want, nil); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
