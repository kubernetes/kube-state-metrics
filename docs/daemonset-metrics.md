# DaemonSet Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_daemonset_annotations | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; <br> `annotation_DAEMONSET_ANNOTATION`=&lt;DAEMONSET_ANNOTATION&gt; | EXPERIMENTAL |
| kube_daemonset_created | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; | STABLE |
| kube_daemonset_status_current_number_scheduled | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; | STABLE |
| kube_daemonset_status_desired_number_scheduled | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; | STABLE |
| kube_daemonset_status_number_available | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; | STABLE |
| kube_daemonset_status_number_misscheduled | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; | STABLE |
| kube_daemonset_status_number_ready | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; | STABLE |
| kube_daemonset_status_number_unavailable | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; | STABLE |
| kube_daemonset_status_observed_generation | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; | STABLE |
| kube_daemonset_status_updated_number_scheduled | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; | STABLE |
| kube_daemonset_metadata_generation | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; | STABLE |
| kube_daemonset_labels | Gauge | `daemonset`=&lt;daemonset-name&gt; <br> `namespace`=&lt;daemonset-namespace&gt; <br> `label_DAEMONSET_LABEL`=&lt;DAEMONSET_LABEL&gt; | STABLE |
