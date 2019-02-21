# Job Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_job_info | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
| kube_job_labels | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; <br> `label_JOB_LABEL`=&lt;JOB_LABEL&gt;  | STABLE |
| kube_job_spec_parallelism | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
| kube_job_spec_completions | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
| kube_job_spec_active_deadline_seconds | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
| kube_job_status_active | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
| kube_job_status_succeeded | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
| kube_job_status_failed | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
| kube_job_status_start_time | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
| kube_job_status_completion_time | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
| kube_job_complete | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
| kube_job_failed | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
| kube_job_created | Gauge | `job_name`=&lt;job-name&gt; <br> `namespace`=&lt;job-namespace&gt; | STABLE |
