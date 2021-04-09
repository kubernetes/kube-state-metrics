# Namespace Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_namespace_created | Gauge | `namespace`=&lt;namespace-name&gt; | STABLE |
| kube_namespace_labels | Gauge | `namespace`=&lt;namespace-name&gt; <br> `label_NS_LABEL`=&lt;NS_LABEL&gt; | STABLE |
| kube_namespace_status_condition | Gauge | `namespace`=&lt;namespace-name&gt; <br> `condition`=&lt;NamespaceDeletionDiscoveryFailure\|NamespaceDeletionContentFailure\|NamespaceDeletionGroupVersionParsingFailure&gt;  <br> `status`=&lt;true\|false\|unknown&gt; | EXPERIMENTAL |
| kube_namespace_status_phase| Gauge | `namespace`=&lt;namespace-name&gt; <br> `phase`=&lt;Active\|Terminating&gt; | STABLE |
