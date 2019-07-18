# PodDisruptionBudget Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_poddisruptionbudget_created | Gauge | `poddisruptionbudget`=&lt;pdb-name&gt; <br> `namespace`=&lt;pdb-namespace&gt;  | STABLE
| kube_poddisruptionbudget_status_current_healthy | Gauge | `poddisruptionbudget`=&lt;pdb-name&gt; <br> `namespace`=&lt;pdb-namespace&gt;  | STABLE
| kube_poddisruptionbudget_status_desired_healthy | Gauge | `poddisruptionbudget`=&lt;pdb-name&gt; <br> `namespace`=&lt;pdb-namespace&gt;  | STABLE
| kube_poddisruptionbudget_status_pod_disruptions_allowed | Gauge | `poddisruptionbudget`=&lt;pdb-name&gt; <br> `namespace`=&lt;pdb-namespace&gt;  | STABLE
| kube_poddisruptionbudget_status_expected_pods | Gauge | `poddisruptionbudget`=&lt;pdb-name&gt; <br> `namespace`=&lt;pdb-namespace&gt;  | STABLE
| kube_poddisruptionbudget_status_observed_generation | Gauge | `poddisruptionbudget`=&lt;pdb-name&gt; <br> `namespace`=&lt;pdb-namespace&gt;  | STABLE
| kube_poddisruptionbudget_annotations | Gauge | `annotation_PODDISRUPTIONBUDGET_ANNOTATION`=&lt;PODDISRUPTIONBUDGET_ANNOTATION&gt; <br> `poddisruptionbudget`=&lt;poddisruptionbudget-name&gt; <br> `namespace`=&lt;poddisruptionbudget-namespace&gt; | EXPERIMENTAL |
