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
	"fmt"
	"math"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

var (
	SuspendTrue                      = true
	SuspendFalse                     = false
	StartingDeadlineSeconds300 int64 = 300
	SuccessfulJobHistoryLimit3 int32 = 3
	FailedJobHistoryLimit1     int32 = 1

	// "1520742896" is "2018/3/11 12:34:56" in "Asia/Shanghai".
	ActiveRunningCronJob1LastScheduleTime          = time.Unix(1520742896, 0)
	SuspendedCronJob1LastScheduleTime              = time.Unix(1520742896+5.5*3600, 0) // 5.5 hours later
	ActiveCronJob1NoLastScheduledCreationTimestamp = time.Unix(1520742896+6.5*3600, 0)
	TimeZone                                       = "Asia/Shanghai"
)

func calculateNextSchedule6h(timestamp time.Time, timezone string) time.Time {
	loc, _ := time.LoadLocation(timezone)
	hour := timestamp.In(loc).Hour()
	switch {
	case hour < 6:
		return time.Date(
			timestamp.Year(),
			timestamp.Month(),
			timestamp.Day(),
			6,
			0,
			0, 0, loc)
	case hour < 12:
		return time.Date(
			timestamp.Year(),
			timestamp.Month(),
			timestamp.Day(),
			12,
			0,
			0, 0, loc)
	case hour < 18:
		return time.Date(
			timestamp.Year(),
			timestamp.Month(),
			timestamp.Day(),
			18,
			0,
			0, 0, loc)
	default:
		return time.Date(
			timestamp.Year(),
			timestamp.Month(),
			timestamp.Day()+1,
			0,
			0,
			0, 0, loc)
	}
}

func calculateNextSchedule25m(timestamp time.Time, timezone string) time.Time {
	loc, _ := time.LoadLocation(timezone)
	minute := timestamp.In(loc).Minute()
	switch {
	case minute < 25:
		return time.Date(
			timestamp.Year(),
			timestamp.Month(),
			timestamp.Day(),
			timestamp.Hour(),
			25,
			0, 0, loc)
	default:
		return time.Date(
			timestamp.Year(),
			timestamp.Month(),
			timestamp.Day(),
			timestamp.Hour()+1,
			25,
			0, 0, loc)
	}

}
func TestCronJobStore(t *testing.T) {

	ActiveRunningCronJob1NextScheduleTime := calculateNextSchedule6h(ActiveRunningCronJob1LastScheduleTime, "Local")
	ActiveRunningCronJobWithTZ1NextScheduleTime := calculateNextSchedule6h(ActiveRunningCronJob1LastScheduleTime, TimeZone)

	ActiveCronJob1NoLastScheduledNextScheduleTime := calculateNextSchedule25m(ActiveCronJob1NoLastScheduledCreationTimestamp, "Local")

	cases := []generateMetricsTestCase{
		{
			AllowAnnotationsList: []string{
				"app.k8s.io/owner",
			},
			Obj: &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "ActiveRunningCronJobWithTZ1",
					Namespace:       "ns1",
					Generation:      1,
					ResourceVersion: "11111",
					Labels: map[string]string{
						"app": "example-active-running-with-tz-1",
					},
					Annotations: map[string]string{
						"app":              "mysql-server",
						"app.k8s.io/owner": "@foo",
					},
				},
				Status: batchv1.CronJobStatus{
					Active:             []v1.ObjectReference{{Name: "FakeJob1"}, {Name: "FakeJob2"}},
					LastScheduleTime:   &metav1.Time{Time: ActiveRunningCronJob1LastScheduleTime},
					LastSuccessfulTime: nil,
				},
				Spec: batchv1.CronJobSpec{
					StartingDeadlineSeconds:    &StartingDeadlineSeconds300,
					ConcurrencyPolicy:          "Forbid",
					Suspend:                    &SuspendFalse,
					Schedule:                   "0 */6 * * *",
					SuccessfulJobsHistoryLimit: &SuccessfulJobHistoryLimit3,
					FailedJobsHistoryLimit:     &FailedJobHistoryLimit1,
					TimeZone:                   &TimeZone,
				},
			},
			Want: `
				# HELP kube_cronjob_created [STABLE] Unix creation timestamp
				# HELP kube_cronjob_info [STABLE] Info about cronjob.
				# HELP kube_cronjob_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_cronjob_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_cronjob_next_schedule_time [STABLE] Next time the cronjob should be scheduled. The time after lastScheduleTime, or after the cron job's creation time if it's never been scheduled. Use this to determine if the job is delayed.
				# HELP kube_cronjob_spec_failed_job_history_limit Failed job history limit tells the controller how many failed jobs should be preserved.
				# HELP kube_cronjob_spec_starting_deadline_seconds [STABLE] Deadline in seconds for starting the job if it misses scheduled time for any reason.
        		# HELP kube_cronjob_spec_successful_job_history_limit Successful job history limit tells the controller how many completed jobs should be preserved.
				# HELP kube_cronjob_spec_suspend [STABLE] Suspend flag tells the controller to suspend subsequent executions.
				# HELP kube_cronjob_status_active [STABLE] Active holds pointers to currently running jobs.
                # HELP kube_cronjob_metadata_resource_version [STABLE] Resource version representing a specific version of the cronjob.
				# HELP kube_cronjob_status_last_schedule_time [STABLE] LastScheduleTime keeps information of when was the last time the job was successfully scheduled.
				# TYPE kube_cronjob_created gauge
				# TYPE kube_cronjob_info gauge
				# TYPE kube_cronjob_annotations gauge
				# TYPE kube_cronjob_labels gauge
				# TYPE kube_cronjob_next_schedule_time gauge
				# TYPE kube_cronjob_spec_failed_job_history_limit gauge
				# TYPE kube_cronjob_spec_starting_deadline_seconds gauge
				# TYPE kube_cronjob_spec_successful_job_history_limit gauge
				# TYPE kube_cronjob_spec_suspend gauge
				# TYPE kube_cronjob_status_active gauge
                # TYPE kube_cronjob_metadata_resource_version gauge
				# TYPE kube_cronjob_status_last_schedule_time gauge
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="ActiveRunningCronJobWithTZ1",namespace="ns1",schedule="0 */6 * * *",timezone="Asia/Shanghai"} 1
				kube_cronjob_annotations{annotation_app_k8s_io_owner="@foo",cronjob="ActiveRunningCronJobWithTZ1",namespace="ns1"} 1
				kube_cronjob_spec_failed_job_history_limit{cronjob="ActiveRunningCronJobWithTZ1",namespace="ns1"} 1
				kube_cronjob_spec_starting_deadline_seconds{cronjob="ActiveRunningCronJobWithTZ1",namespace="ns1"} 300
				kube_cronjob_spec_successful_job_history_limit{cronjob="ActiveRunningCronJobWithTZ1",namespace="ns1"} 3
				kube_cronjob_spec_suspend{cronjob="ActiveRunningCronJobWithTZ1",namespace="ns1"} 0
				kube_cronjob_status_active{cronjob="ActiveRunningCronJobWithTZ1",namespace="ns1"} 2
                kube_cronjob_metadata_resource_version{cronjob="ActiveRunningCronJobWithTZ1",namespace="ns1"} 11111
				kube_cronjob_status_last_schedule_time{cronjob="ActiveRunningCronJobWithTZ1",namespace="ns1"} 1.520742896e+09
` + fmt.Sprintf("kube_cronjob_next_schedule_time{cronjob=\"ActiveRunningCronJobWithTZ1\",namespace=\"ns1\"} %ve+09\n",
				float64(ActiveRunningCronJobWithTZ1NextScheduleTime.Unix())/math.Pow10(9)),
			MetricNames: []string{
				"kube_cronjob_next_schedule_time",
				"kube_cronjob_spec_starting_deadline_seconds",
				"kube_cronjob_status_active",
				"kube_cronjob_metadata_resource_version",
				"kube_cronjob_spec_suspend",
				"kube_cronjob_info",
				"kube_cronjob_created",
				"kube_cronjob_annotations",
				"kube_cronjob_labels",
				"kube_cronjob_status_last_schedule_time",
				"kube_cronjob_spec_successful_job_history_limit",
				"kube_cronjob_spec_failed_job_history_limit",
			},
		},
		{
			AllowAnnotationsList: []string{
				"app.k8s.io/owner",
			},
			Obj: &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "ActiveRunningCronJob1",
					Namespace:       "ns1",
					Generation:      1,
					ResourceVersion: "11111",
					Labels: map[string]string{
						"app": "example-active-running-1",
					},
					Annotations: map[string]string{
						"app":              "mysql-server",
						"app.k8s.io/owner": "@foo",
					},
				},
				Status: batchv1.CronJobStatus{
					Active:             []v1.ObjectReference{{Name: "FakeJob1"}, {Name: "FakeJob2"}},
					LastScheduleTime:   &metav1.Time{Time: ActiveRunningCronJob1LastScheduleTime},
					LastSuccessfulTime: nil,
				},
				Spec: batchv1.CronJobSpec{
					StartingDeadlineSeconds:    &StartingDeadlineSeconds300,
					ConcurrencyPolicy:          "Forbid",
					Suspend:                    &SuspendFalse,
					Schedule:                   "0 */6 * * *",
					SuccessfulJobsHistoryLimit: &SuccessfulJobHistoryLimit3,
					FailedJobsHistoryLimit:     &FailedJobHistoryLimit1,
				},
			},
			Want: `
				# HELP kube_cronjob_created [STABLE] Unix creation timestamp
				# HELP kube_cronjob_info [STABLE] Info about cronjob.
				# HELP kube_cronjob_annotations Kubernetes annotations converted to Prometheus labels.
				# HELP kube_cronjob_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_cronjob_next_schedule_time [STABLE] Next time the cronjob should be scheduled. The time after lastScheduleTime, or after the cron job's creation time if it's never been scheduled. Use this to determine if the job is delayed.
				# HELP kube_cronjob_spec_failed_job_history_limit Failed job history limit tells the controller how many failed jobs should be preserved.
				# HELP kube_cronjob_spec_starting_deadline_seconds [STABLE] Deadline in seconds for starting the job if it misses scheduled time for any reason.
        		# HELP kube_cronjob_spec_successful_job_history_limit Successful job history limit tells the controller how many completed jobs should be preserved.
				# HELP kube_cronjob_spec_suspend [STABLE] Suspend flag tells the controller to suspend subsequent executions.
				# HELP kube_cronjob_status_active [STABLE] Active holds pointers to currently running jobs.
                # HELP kube_cronjob_metadata_resource_version [STABLE] Resource version representing a specific version of the cronjob.
				# HELP kube_cronjob_status_last_schedule_time [STABLE] LastScheduleTime keeps information of when was the last time the job was successfully scheduled.
				# TYPE kube_cronjob_created gauge
				# TYPE kube_cronjob_info gauge
				# TYPE kube_cronjob_annotations gauge
				# TYPE kube_cronjob_labels gauge
				# TYPE kube_cronjob_next_schedule_time gauge
				# TYPE kube_cronjob_spec_failed_job_history_limit gauge
				# TYPE kube_cronjob_spec_starting_deadline_seconds gauge
				# TYPE kube_cronjob_spec_successful_job_history_limit gauge
				# TYPE kube_cronjob_spec_suspend gauge
				# TYPE kube_cronjob_status_active gauge
                # TYPE kube_cronjob_metadata_resource_version gauge
				# TYPE kube_cronjob_status_last_schedule_time gauge
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="ActiveRunningCronJob1",namespace="ns1",schedule="0 */6 * * *",timezone="local"} 1
				kube_cronjob_annotations{annotation_app_k8s_io_owner="@foo",cronjob="ActiveRunningCronJob1",namespace="ns1"} 1
				kube_cronjob_spec_failed_job_history_limit{cronjob="ActiveRunningCronJob1",namespace="ns1"} 1
				kube_cronjob_spec_starting_deadline_seconds{cronjob="ActiveRunningCronJob1",namespace="ns1"} 300
				kube_cronjob_spec_successful_job_history_limit{cronjob="ActiveRunningCronJob1",namespace="ns1"} 3
				kube_cronjob_spec_suspend{cronjob="ActiveRunningCronJob1",namespace="ns1"} 0
				kube_cronjob_status_active{cronjob="ActiveRunningCronJob1",namespace="ns1"} 2
                kube_cronjob_metadata_resource_version{cronjob="ActiveRunningCronJob1",namespace="ns1"} 11111
				kube_cronjob_status_last_schedule_time{cronjob="ActiveRunningCronJob1",namespace="ns1"} 1.520742896e+09
` + fmt.Sprintf("kube_cronjob_next_schedule_time{cronjob=\"ActiveRunningCronJob1\",namespace=\"ns1\"} %ve+09\n",
				float64(ActiveRunningCronJob1NextScheduleTime.Unix())/math.Pow10(9)),
			MetricNames: []string{
				"kube_cronjob_next_schedule_time",
				"kube_cronjob_spec_starting_deadline_seconds",
				"kube_cronjob_status_active",
				"kube_cronjob_metadata_resource_version",
				"kube_cronjob_spec_suspend",
				"kube_cronjob_info",
				"kube_cronjob_created",
				"kube_cronjob_annotations",
				"kube_cronjob_labels",
				"kube_cronjob_status_last_schedule_time",
				"kube_cronjob_spec_successful_job_history_limit",
				"kube_cronjob_spec_failed_job_history_limit",
			},
		},
		{
			Obj: &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "SuspendedCronJob1",
					Namespace:       "ns1",
					Generation:      1,
					ResourceVersion: "22222",
					Labels: map[string]string{
						"app": "example-suspended-1",
					},
				},
				Status: batchv1.CronJobStatus{
					Active:             []v1.ObjectReference{},
					LastScheduleTime:   &metav1.Time{Time: SuspendedCronJob1LastScheduleTime},
					LastSuccessfulTime: nil,
				},
				Spec: batchv1.CronJobSpec{
					StartingDeadlineSeconds:    &StartingDeadlineSeconds300,
					ConcurrencyPolicy:          "Forbid",
					Suspend:                    &SuspendTrue,
					Schedule:                   "0 */3 * * *",
					TimeZone:                   &TimeZone,
					SuccessfulJobsHistoryLimit: &SuccessfulJobHistoryLimit3,
					FailedJobsHistoryLimit:     &FailedJobHistoryLimit1,
				},
			},
			Want: `
				# HELP kube_cronjob_created [STABLE] Unix creation timestamp
				# HELP kube_cronjob_info [STABLE] Info about cronjob.
				# HELP kube_cronjob_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_cronjob_spec_failed_job_history_limit Failed job history limit tells the controller how many failed jobs should be preserved.
				# HELP kube_cronjob_spec_starting_deadline_seconds [STABLE] Deadline in seconds for starting the job if it misses scheduled time for any reason.
				# HELP kube_cronjob_spec_successful_job_history_limit Successful job history limit tells the controller how many completed jobs should be preserved.
				# HELP kube_cronjob_spec_suspend [STABLE] Suspend flag tells the controller to suspend subsequent executions.
				# HELP kube_cronjob_status_active [STABLE] Active holds pointers to currently running jobs.
                # HELP kube_cronjob_metadata_resource_version [STABLE] Resource version representing a specific version of the cronjob.
				# HELP kube_cronjob_status_last_schedule_time [STABLE] LastScheduleTime keeps information of when was the last time the job was successfully scheduled.
				# HELP kube_cronjob_status_last_successful_time [STABLE] LastSuccessfulTime keeps information of when was the last time the job was completed successfully.
				# TYPE kube_cronjob_created gauge
				# TYPE kube_cronjob_info gauge
				# TYPE kube_cronjob_labels gauge
				# TYPE kube_cronjob_spec_failed_job_history_limit gauge
				# TYPE kube_cronjob_spec_starting_deadline_seconds gauge
				# TYPE kube_cronjob_spec_successful_job_history_limit gauge
				# TYPE kube_cronjob_spec_suspend gauge
				# TYPE kube_cronjob_status_active gauge
                # TYPE kube_cronjob_metadata_resource_version gauge
				# TYPE kube_cronjob_status_last_schedule_time gauge
				# TYPE kube_cronjob_status_last_successful_time gauge
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="SuspendedCronJob1",namespace="ns1",schedule="0 */3 * * *",timezone="Asia/Shanghai"} 1
				kube_cronjob_spec_failed_job_history_limit{cronjob="SuspendedCronJob1",namespace="ns1"} 1
				kube_cronjob_spec_starting_deadline_seconds{cronjob="SuspendedCronJob1",namespace="ns1"} 300
				kube_cronjob_spec_successful_job_history_limit{cronjob="SuspendedCronJob1",namespace="ns1"} 3
				kube_cronjob_spec_suspend{cronjob="SuspendedCronJob1",namespace="ns1"} 1
				kube_cronjob_status_active{cronjob="SuspendedCronJob1",namespace="ns1"} 0
				kube_cronjob_metadata_resource_version{cronjob="SuspendedCronJob1",namespace="ns1"} 22222
				kube_cronjob_status_last_schedule_time{cronjob="SuspendedCronJob1",namespace="ns1"} 1.520762696e+09
`,
			MetricNames: []string{"kube_cronjob_status_last_successful_time", "kube_cronjob_spec_starting_deadline_seconds", "kube_cronjob_status_active", "kube_cronjob_metadata_resource_version", "kube_cronjob_spec_suspend", "kube_cronjob_info", "kube_cronjob_created", "kube_cronjob_labels", "kube_cronjob_status_last_schedule_time", "kube_cronjob_spec_successful_job_history_limit", "kube_cronjob_spec_failed_job_history_limit"},
		},
		{
			Obj: &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "SuspendedCronJob1",
					Namespace:       "ns1",
					Generation:      1,
					ResourceVersion: "22222",
					Labels: map[string]string{
						"app": "example-suspended-1",
					},
				},
				Status: batchv1.CronJobStatus{
					Active:             []v1.ObjectReference{},
					LastScheduleTime:   &metav1.Time{Time: SuspendedCronJob1LastScheduleTime},
					LastSuccessfulTime: &metav1.Time{Time: SuspendedCronJob1LastScheduleTime},
				},
				Spec: batchv1.CronJobSpec{
					StartingDeadlineSeconds:    &StartingDeadlineSeconds300,
					ConcurrencyPolicy:          "Forbid",
					Suspend:                    &SuspendTrue,
					Schedule:                   "0 */3 * * *",
					SuccessfulJobsHistoryLimit: &SuccessfulJobHistoryLimit3,
					FailedJobsHistoryLimit:     &FailedJobHistoryLimit1,
				},
			},
			Want: `
				# HELP kube_cronjob_created [STABLE] Unix creation timestamp
				# HELP kube_cronjob_info [STABLE] Info about cronjob.
				# HELP kube_cronjob_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_cronjob_spec_failed_job_history_limit Failed job history limit tells the controller how many failed jobs should be preserved.
				# HELP kube_cronjob_spec_starting_deadline_seconds [STABLE] Deadline in seconds for starting the job if it misses scheduled time for any reason.
				# HELP kube_cronjob_spec_successful_job_history_limit Successful job history limit tells the controller how many completed jobs should be preserved.
				# HELP kube_cronjob_spec_suspend [STABLE] Suspend flag tells the controller to suspend subsequent executions.
				# HELP kube_cronjob_status_active [STABLE] Active holds pointers to currently running jobs.
                # HELP kube_cronjob_metadata_resource_version [STABLE] Resource version representing a specific version of the cronjob.
				# HELP kube_cronjob_status_last_schedule_time [STABLE] LastScheduleTime keeps information of when was the last time the job was successfully scheduled.
				# HELP kube_cronjob_status_last_successful_time [STABLE] LastSuccessfulTime keeps information of when was the last time the job was completed successfully.
				# TYPE kube_cronjob_created gauge
				# TYPE kube_cronjob_info gauge
				# TYPE kube_cronjob_labels gauge
				# TYPE kube_cronjob_spec_failed_job_history_limit gauge
				# TYPE kube_cronjob_spec_starting_deadline_seconds gauge
				# TYPE kube_cronjob_spec_successful_job_history_limit gauge
				# TYPE kube_cronjob_spec_suspend gauge
				# TYPE kube_cronjob_status_active gauge
                # TYPE kube_cronjob_metadata_resource_version gauge
				# TYPE kube_cronjob_status_last_schedule_time gauge
				# TYPE kube_cronjob_status_last_successful_time gauge
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="SuspendedCronJob1",namespace="ns1",schedule="0 */3 * * *",timezone="local"} 1
				kube_cronjob_spec_failed_job_history_limit{cronjob="SuspendedCronJob1",namespace="ns1"} 1
				kube_cronjob_spec_starting_deadline_seconds{cronjob="SuspendedCronJob1",namespace="ns1"} 300
				kube_cronjob_spec_successful_job_history_limit{cronjob="SuspendedCronJob1",namespace="ns1"} 3
				kube_cronjob_spec_suspend{cronjob="SuspendedCronJob1",namespace="ns1"} 1
				kube_cronjob_status_active{cronjob="SuspendedCronJob1",namespace="ns1"} 0
				kube_cronjob_metadata_resource_version{cronjob="SuspendedCronJob1",namespace="ns1"} 22222
				kube_cronjob_status_last_schedule_time{cronjob="SuspendedCronJob1",namespace="ns1"} 1.520762696e+09
				kube_cronjob_status_last_successful_time{cronjob="SuspendedCronJob1",namespace="ns1"} 1.520762696e+09
`,
			MetricNames: []string{"kube_cronjob_status_last_successful_time", "kube_cronjob_spec_starting_deadline_seconds", "kube_cronjob_status_active", "kube_cronjob_metadata_resource_version", "kube_cronjob_spec_suspend", "kube_cronjob_info", "kube_cronjob_created", "kube_cronjob_labels", "kube_cronjob_status_last_schedule_time", "kube_cronjob_spec_successful_job_history_limit", "kube_cronjob_spec_failed_job_history_limit"},
		},
		{
			Obj: &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "ActiveCronJob1NoLastScheduled",
					CreationTimestamp: metav1.Time{Time: ActiveCronJob1NoLastScheduledCreationTimestamp},
					Namespace:         "ns1",
					Generation:        1,
					ResourceVersion:   "33333",
					Labels: map[string]string{
						"app": "example-active-no-last-scheduled-1",
					},
				},
				Status: batchv1.CronJobStatus{
					Active:             []v1.ObjectReference{},
					LastScheduleTime:   nil,
					LastSuccessfulTime: nil,
				},
				Spec: batchv1.CronJobSpec{
					StartingDeadlineSeconds:    &StartingDeadlineSeconds300,
					ConcurrencyPolicy:          "Forbid",
					Suspend:                    &SuspendFalse,
					Schedule:                   "25 * * * *",
					SuccessfulJobsHistoryLimit: &SuccessfulJobHistoryLimit3,
					FailedJobsHistoryLimit:     &FailedJobHistoryLimit1,
				},
			},
			Want: `
				# HELP kube_cronjob_created [STABLE] Unix creation timestamp
				# HELP kube_cronjob_info [STABLE] Info about cronjob.
				# HELP kube_cronjob_labels [STABLE] Kubernetes labels converted to Prometheus labels.
				# HELP kube_cronjob_next_schedule_time [STABLE] Next time the cronjob should be scheduled. The time after lastScheduleTime, or after the cron job's creation time if it's never been scheduled. Use this to determine if the job is delayed.
				# HELP kube_cronjob_spec_failed_job_history_limit Failed job history limit tells the controller how many failed jobs should be preserved.
				# HELP kube_cronjob_spec_starting_deadline_seconds [STABLE] Deadline in seconds for starting the job if it misses scheduled time for any reason.
				# HELP kube_cronjob_spec_successful_job_history_limit Successful job history limit tells the controller how many completed jobs should be preserved.
				# HELP kube_cronjob_spec_suspend [STABLE] Suspend flag tells the controller to suspend subsequent executions.
				# HELP kube_cronjob_status_active [STABLE] Active holds pointers to currently running jobs.
				# HELP kube_cronjob_status_last_successful_time [STABLE] LastSuccessfulTime keeps information of when was the last time the job was completed successfully.
                # HELP kube_cronjob_metadata_resource_version [STABLE] Resource version representing a specific version of the cronjob.
				# TYPE kube_cronjob_created gauge
				# TYPE kube_cronjob_info gauge
				# TYPE kube_cronjob_labels gauge
				# TYPE kube_cronjob_next_schedule_time gauge
				# TYPE kube_cronjob_spec_failed_job_history_limit gauge
				# TYPE kube_cronjob_spec_starting_deadline_seconds gauge
				# TYPE kube_cronjob_spec_successful_job_history_limit gauge
				# TYPE kube_cronjob_spec_suspend gauge
				# TYPE kube_cronjob_status_active gauge
                		# TYPE kube_cronjob_metadata_resource_version gauge
				# TYPE kube_cronjob_status_last_successful_time gauge
				kube_cronjob_spec_starting_deadline_seconds{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 300
				kube_cronjob_status_active{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 0
				kube_cronjob_metadata_resource_version{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 33333
				kube_cronjob_spec_failed_job_history_limit{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 1
				kube_cronjob_spec_successful_job_history_limit{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 3
				kube_cronjob_spec_suspend{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 0
				kube_cronjob_info{concurrency_policy="Forbid",cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1",schedule="25 * * * *",timezone="local"} 1
				kube_cronjob_created{cronjob="ActiveCronJob1NoLastScheduled",namespace="ns1"} 1.520766296e+09
` +
				fmt.Sprintf("kube_cronjob_next_schedule_time{cronjob=\"ActiveCronJob1NoLastScheduled\",namespace=\"ns1\"} %ve+09\n",
					float64(ActiveCronJob1NoLastScheduledNextScheduleTime.Unix())/math.Pow10(9)),
			MetricNames: []string{"kube_cronjob_status_last_successful_time", "kube_cronjob_next_schedule_time", "kube_cronjob_spec_starting_deadline_seconds", "kube_cronjob_status_active", "kube_cronjob_metadata_resource_version", "kube_cronjob_spec_suspend", "kube_cronjob_info", "kube_cronjob_created", "kube_cronjob_labels", "kube_cronjob_spec_successful_job_history_limit", "kube_cronjob_spec_failed_job_history_limit"},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(cronJobMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		c.Headers = generator.ExtractMetricFamilyHeaders(cronJobMetricFamilies(c.AllowAnnotationsList, c.AllowLabelsList))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}

func TestGetNextScheduledTime(t *testing.T) {

	testCases := []struct {
		schedule         string
		lastScheduleTime metav1.Time
		createdTime      metav1.Time
		timeZone         string
		expected         time.Time
	}{
		{
			schedule:         "0 */6 * * *",
			lastScheduleTime: metav1.Time{Time: ActiveRunningCronJob1LastScheduleTime},
			createdTime:      metav1.Time{Time: ActiveRunningCronJob1LastScheduleTime},
			timeZone:         "UTC",
			expected:         ActiveRunningCronJob1LastScheduleTime.Add(time.Second*4 + time.Minute*25 + time.Hour),
		},
		{
			schedule:         "0 */6 * * *",
			lastScheduleTime: metav1.Time{Time: ActiveRunningCronJob1LastScheduleTime},
			createdTime:      metav1.Time{Time: ActiveRunningCronJob1LastScheduleTime},
			timeZone:         TimeZone,
			expected:         ActiveRunningCronJob1LastScheduleTime.Add(time.Second*4 + time.Minute*25 + time.Hour*5),
		},
	}

	for _, test := range testCases {
		actual, _ := getNextScheduledTime(test.schedule, &test.lastScheduleTime, test.createdTime, &test.timeZone) // #nosec G601
		if !actual.Equal(test.expected) {
			t.Fatalf("%v: expected %v, actual %v", test.schedule, test.expected, actual)
		}
	}

}
