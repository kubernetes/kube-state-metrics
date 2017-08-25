# DaemonSet Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_daemonset_status_current_number_scheduled | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; |
| kube_daemonset_status_number_misscheduled | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; |
| kube_daemonset_status_desired_number_scheduled | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; |
| kube_daemonset_status_number_ready | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; |
| kube_daemonset_metadata_generation | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; |
| kube_daemonset_created | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; |
