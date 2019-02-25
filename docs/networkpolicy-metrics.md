# Network Policy Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_networkpolicy_info | Gauge | `networkpolicy`=&lt;networkpolicy-name&gt; <br> `namespace`=&lt;networkpolicy-namespace&gt; | STABLE |
| kube_networkpolicy_labels | Gauge | `networkpolicy`=&lt;networkpolicy-name&gt; <br> `namespace`=&lt;networkpolicy-namespace&gt; <br> `label_networkpolicy_LABEL`=&lt;networkpolicy_LABEL&gt; | STABLE |
| kube_networkpolicy_created  | Gauge | `networkpolicy`=&lt;networkpolicy-name&gt; <br> `namespace`=&lt;networkpolicy-namespace&gt; | STABLE |
| kube_networkpolicy_metadata_resource_version  | Gauge | `networkpolicy`=&lt;networkpolicy-name&gt; <br> `namespace`=&lt;networkpolicy-namespace&gt; <br> `resource_version`=&lt;networkpolicy-resource-version&gt; | STABLE |
