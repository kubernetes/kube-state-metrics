# ReplicaSet metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_replicaset_status_replicas | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; |
| kube_replicaset_status_fully_labeled_replicas | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; |
| kube_replicaset_status_ready_replicas | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; |
| kube_replicaset_status_observed_generation | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; |
| kube_replicaset_spec_replicas | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; |
| kube_replicaset_metadata_generation | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; |
| kube_replicaset_created | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; |
