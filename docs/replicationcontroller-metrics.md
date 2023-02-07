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
