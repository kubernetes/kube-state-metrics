# Job Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_job_info | Gauge | `job`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; <br> `parallelism`=&lt;parallelism&gt; <br> `completions`=&lt;completions&gt; <br> `active_deadline_seconds`=&lt;active-deadline-seconds&gt; |
| kube_job_status_active | Gauge | `job`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; |
| kube_job_status_succeeded | Gauge | `job`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; |
| kube_job_status_failed | Gauge | `job`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; |
| kube_job_status_start_time | Counter | `job`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; |
| kube_job_status_completion_time | Counter | `job`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; |
| kube_job_complete | Gauge | `job`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; |
| kube_job_failed | Gauge | `job`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; |
