# ReplicationController metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_replicationcontroller_status_replicas | Gauge | `replicationcontroller`=&lt;replicationcontroller-name&gt; <br> `namespace`=&lt;replicationcontroller-namespace&gt; | STABLE |
| kube_replicationcontroller_status_fully_labeled_replicas | Gauge | `replicationcontroller`=&lt;replicationcontroller-name&gt; <br> `namespace`=&lt;replicationcontroller-namespace&gt; | STABLE |
| kube_replicationcontroller_status_ready_replicas | Gauge | `replicationcontroller`=&lt;replicationcontroller-name&gt; <br> `namespace`=&lt;replicationcontroller-namespace&gt; | STABLE |
| kube_replicationcontroller_status_available_replicas | Gauge | `replicationcontroller`=&lt;replicationcontroller-name&gt; <br> `namespace`=&lt;replicationcontroller-namespace&gt; | STABLE |
| kube_replicationcontroller_status_observed_generation | Gauge | `replicationcontroller`=&lt;replicationcontroller-name&gt; <br> `namespace`=&lt;replicationcontroller-namespace&gt; | STABLE |
| kube_replicationcontroller_spec_replicas | Gauge | `replicationcontroller`=&lt;replicationcontroller-name&gt; <br> `namespace`=&lt;replicationcontroller-namespace&gt; | STABLE |
| kube_replicationcontroller_metadata_generation | Gauge | `replicationcontroller`=&lt;replicationcontroller-name&gt; <br> `namespace`=&lt;replicationcontroller-namespace&gt; | STABLE |
| kube_replicationcontroller_created | Gauge | `replicationcontroller`=&lt;replicationcontroller-name&gt; <br> `namespace`=&lt;replicationcontroller-namespace&gt; | STABLE |
| kube_replicationcontroller_owner | Gauge | `replicationcontroller`=&lt;replicationcontroller-name&gt; <br> `namespace`=&lt;replicationcontroller-namespace&gt; <br> `owner_kind`=&lt;owner kind&gt; <br> `owner_name`=&lt;owner name&gt; <br> `owner_is_controller`=&lt;whether owner is controller&gt;  | EXPERIMENTAL |
