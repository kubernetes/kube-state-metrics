# CronJob Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_cronjob_info | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; <br> `schedule`=&lt;schedule&gt; <br> `concurrency_policy`=&lt;concurrency-policy&gt; |
| kube_cronjob_next_schedule_time  | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; |
| kube_cronjob_status_active | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; |
| kube_cronjob_status_last_schedule_time | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; |
| kube_cronjob_spec_suspend | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; |
| kube_cronjob_spec_starting_deadline_seconds | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; |
