# Stateful Set Metrics
 
| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_statefulset_status_replicas | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  |
| kube_statefulset_status_observed_generation | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  |
| kube_statefulset_replicas | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  |
| kube_statefulset_metadata_generation | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  |