# CronJob Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_cronjob_info | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; <br> `schedule`=&lt;schedule&gt; <br> `concurrency_policy`=&lt;concurrency-policy&gt; | STABLE
| kube_cronjob_labels | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; <br> `label_CRONJOB_LABEL`=&lt;CRONJOB_LABEL&gt;  | STABLE
| kube_cronjob_created  | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; | STABLE
| kube_cronjob_next_schedule_time  | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; | STABLE
| kube_cronjob_status_active | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; | STABLE
| kube_cronjob_status_last_schedule_time | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; | STABLE
| kube_cronjob_spec_suspend | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; | STABLE
| kube_cronjob_spec_starting_deadline_seconds | Gauge | `cronjob`=&lt;cronjob-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt; | STABLE
