# Stateful Set Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_statefulset_annotations | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt; <br> `annotation_STATEFULSET_ANNOTATION`=&lt;STATEFULSET_ANNOTATION&gt; | EXPERIMENTAL |
| kube_statefulset_status_replicas | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  | STABLE |
| kube_statefulset_status_replicas_current | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  | STABLE |
| kube_statefulset_status_replicas_ready | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  | STABLE |
| kube_statefulset_status_replicas_available | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  | EXPERIMENTAL |
| kube_statefulset_status_replicas_updated | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  | STABLE |
| kube_statefulset_status_observed_generation | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  | STABLE |
| kube_statefulset_replicas | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  | STABLE |
| kube_statefulset_metadata_generation | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  | STABLE |
| kube_statefulset_created | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;  | STABLE |
| kube_statefulset_labels | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt; <br> `label_STATEFULSET_LABEL`=&lt;STATEFULSET_LABEL&gt; | STABLE |
| kube_statefulset_status_current_revision | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt; <br> `revision`=&lt;statefulset-current-revision&gt; | STABLE |
| kube_statefulset_status_update_revision | Gauge | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt; <br> `revision`=&lt;statefulset-update-revision&gt; | STABLE |
