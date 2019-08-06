# ReplicaSet metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_replicaset_status_replicas | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; | STABLE |
| kube_replicaset_status_fully_labeled_replicas | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; | STABLE |
| kube_replicaset_status_ready_replicas | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; | STABLE |
| kube_replicaset_status_observed_generation | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; | STABLE |
| kube_replicaset_spec_replicas | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; | STABLE |
| kube_replicaset_metadata_generation | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; | STABLE |
| kube_replicaset_labels | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; | STABLE |
| kube_replicaset_created | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; | STABLE |
| kube_replicaset_owner | Gauge | `replicaset`=&lt;replicaset-name&gt; <br> `namespace`=&lt;replicaset-namespace&gt; <br> `owner_kind`=&lt;owner kind&gt; <br> `owner_name`=&lt;owner name&gt; <br> `owner_is_controller`=&lt;whether owner is controller&gt;  | STABLE |
