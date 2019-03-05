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

package collector

import (
	"fmt"
	"math"
	"testing"
	"time"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/pkg/metric"
)

var (
	SuspendTrue                      = true
	SuspendFalse                     = false
	StartingDeadlineSeconds300 int64 = 300

	// "1520742896" is "2018/3/11 12:34:56" in "Asia/Shanghai".
	ActiveRunningCronJob1LastScheduleTime          = time.Unix(1520742896, 0)
	SuspendedCronJob1LastScheduleTime              = time.Unix(1520742896+5.5*3600, 0) // 5.5 hours later
	ActiveCronJob1NoLastScheduledCreationTimestamp = time.Unix(1520742896+6.5*3600, 0)
)

func TestCronJobCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.

	hour := ActiveRunningCronJob1LastScheduleTime.Hour()
	ActiveRunningCronJob1NextScheduleTime := time.Time{}
	switch {
	case hour < 6:
		ActiveRunningCronJob1NextScheduleTime = time.Date(
			ActiveRunningCronJob1LastScheduleTime.Year(),
			ActiveRunningCronJob1LastScheduleTime.Month(),
			ActiveRunningCronJob1LastScheduleTime.Day(),
			6,
			0,
			0, 0, time.Local)
	case hour < 12:
		ActiveRunningCronJob1NextScheduleTime = time.Date(
			ActiveRunningCronJob1LastScheduleTime.Year(),
			ActiveRunningCronJob1LastScheduleTime.Month(),
			ActiveRunningCronJob1LastScheduleTime.Day(),
			12,
			0,
			0, 0, time.Local)
	case hour < 18:
		ActiveRunningCronJob1NextScheduleTime = time.Date(
			ActiveRunningCronJob1LastScheduleTime.Year(),
			ActiveRunningCronJob1LastScheduleTime.Month(),
			ActiveRunningCronJob1LastScheduleTime.Day(),
			18,
			0,
			0, 0, time.Local)
	case hour < 24:
		ActiveRunningCronJob1NextScheduleTime = time.Date(
			ActiveRunningCronJob1LastScheduleTime.Year(),
			ActiveRunningCronJob1LastScheduleTime.Month(),
			ActiveRunningCronJob1LastScheduleTime.Day(),
			24,
			0,
			0, 0, time.Local)
	}

	minute := ActiveCronJob1NoLastScheduledCreationTimestamp.Minute()
	ActiveCronJob1NoLastScheduledNextScheduleTime := time.Time{}
	switch {
	case minute < 25:
		ActiveCronJob1NoLastScheduledNextScheduleTime = time.Date(
			ActiveCronJob1NoLastScheduledCreationTimestamp.Year(),
			ActiveCronJob1NoLastScheduledCreationTimestamp.Month(),
			ActiveCronJob1NoLastScheduledCreationTimestamp.Day(),
			ActiveCronJob1NoLastScheduledCreationTimestamp.Hour(),
			25,
			0, 0, time.Local)
	default:
		ActiveCronJob1NoLastScheduledNextScheduleTime = time.Date(
			ActiveCronJob1NoLastScheduledNextScheduleTime.Year(),
			ActiveCronJob1NoLastScheduledNextScheduleTime.Month(),
			ActiveCronJob1NoLastScheduledNextScheduleTime.Day(),
			ActiveCronJob1NoLastScheduledNextScheduleTime.Hour()+1,
			25,
			0, 0, time.Local)
	}

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
	cases := []generateMetricsTestCase{
		{
			Obj: &batchv1beta1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "ActiveRunningCronJob1",
					Namespace:  "ns1",
					Generation: 1,
					Labels: map[string]string{
						"app": "example-active-running-1",
					},
				},
				Status: batchv1beta1.CronJobStatus{
					Active:           []v1.ObjectReference{{Name: "FakeJob1"}, {Name: "FakeJob2"}},
					LastScheduleTime: &metav1.Time{Time: ActiveRunningCronJob1LastScheduleTime},
				},
				Spec: batchv1beta1.CronJobSpec{
					StartingDeadlineSeconds: &StartingDeadlineSeconds300,
					ConcurrencyPolicy:       "Forbid",
					Suspend:                 &SuspendFalse,
					Schedule:                "0 */6 * * *",
				},
			},
			Want: `
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="ActiveRunningCronJob1",namespace="ns1",schedule="0 */6 * * *"} 1
				kube_cronjob_labels{cronjob="ActiveRunningCronJob1",label_app="example-active-running-1",namespace="ns1"} 1
				kube_cronjob_spec_starting_deadline_seconds{cronjob="ActiveRunningCronJob1",namespace="ns1"} 300
				kube_cronjob_spec_suspend{cronjob="ActiveRunningCronJob1",namespace="ns1"} 0
				kube_cronjob_status_active{cronjob="ActiveRunningCronJob1",namespace="ns1"} 2
				kube_cronjob_status_last_schedule_time{cronjob="ActiveRunningCronJob1",namespace="ns1"} 1.520742896e+09
` + fmt.Sprintf("kube_cronjob_next_schedule_time{cronjob=\"ActiveRunningCronJob1\",namespace=\"ns1\"} %ve+09\n",
				float64(ActiveRunningCronJob1NextScheduleTime.Unix())/math.Pow10(9)),
			MetricNames: []string{"kube_cronjob_next_schedule_time", "kube_cronjob_spec_starting_deadline_seconds", "kube_cronjob_status_active", "kube_cronjob_spec_suspend", "kube_cronjob_info", "kube_cronjob_created", "kube_cronjob_labels", "kube_cronjob_status_last_schedule_time"},
		},
		{
			Obj: &batchv1beta1.CronJob{
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
					Schedule:                "0 */3 * * *",
				},
			},
			Want: `
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="SuspendedCronJob1",namespace="ns1",schedule="0 */3 * * *"} 1
				kube_cronjob_labels{cronjob="SuspendedCronJob1",label_app="example-suspended-1",namespace="ns1"} 1
				kube_cronjob_spec_starting_deadline_seconds{cronjob="SuspendedCronJob1",namespace="ns1"} 300
				kube_cronjob_spec_suspend{cronjob="SuspendedCronJob1",namespace="ns1"} 1
				kube_cronjob_status_active{cronjob="SuspendedCronJob1",namespace="ns1"} 0
				kube_cronjob_status_last_schedule_time{cronjob="SuspendedCronJob1",namespace="ns1"} 1.520762696e+09
`,
			MetricNames: []string{"kube_cronjob_spec_starting_deadline_seconds", "kube_cronjob_status_active", "kube_cronjob_spec_suspend", "kube_cronjob_info", "kube_cronjob_created", "kube_cronjob_labels", "kube_cronjob_status_last_schedule_time"},
		},
		{
			Obj: &batchv1beta1.CronJob{
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
					Schedule:                "25 * * * *",
				},
			},
			Want: `
				kube_cronjob_spec_starting_deadline_seconds{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 300
				kube_cronjob_status_active{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 0
				kube_cronjob_spec_suspend{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 0
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1",schedule="25 * * * *"} 1
				kube_cronjob_created{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 1.520766296e+09
				kube_cronjob_labels{cronjob="ActiveCronJob1NoLastScheduled",label_app="example-active-no-last-scheduled-1",namespace="ns1"} 1
` +
				fmt.Sprintf("kube_cronjob_next_schedule_time{cronjob=\"ActiveCronJob1NoLastScheduled\",namespace=\"ns1\"} %ve+09\n",
					float64(ActiveCronJob1NoLastScheduledNextScheduleTime.Unix())/math.Pow10(9)),
			// TODO: Do we need to specify metricnames?
			MetricNames: []string{"kube_cronjob_next_schedule_time", "kube_cronjob_spec_starting_deadline_seconds", "kube_cronjob_status_active", "kube_cronjob_spec_suspend", "kube_cronjob_info", "kube_cronjob_created", "kube_cronjob_labels"},
		},
	}
	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(cronJobMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
