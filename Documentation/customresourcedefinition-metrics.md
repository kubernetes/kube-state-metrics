# CustomResourceDefinition Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_customresourcedefinition_created | Gauge | `customresourcedefinition`=&lt;customresourcedefinition-name&gt; | EXPERIMENTAL |
| kube_customresourcedefinition_labels | Gauge | `customresourcedefinition`=&lt;customresourcedefinition-name&gt; <br> `label_CRD_LABEL`=&lt;CRD_LABEL&gt;  | EXPERIMENTAL |
| kube_customresourcedefinition_spec_groupversion | Gauge | `customresourcedefinition`=&lt;customresourcedefinition-name&gt; <br> `group`=&lt;customresourcedefinition-group&gt; <br> `version`=&lt;customresourcedefinition-version&gt; | EXPERIMENTAL |
| kube_customresourcedefinition_spec_scope | Gauge | `customresourcedefinition`=&lt;customresourcedefinition-name&gt; <br> `Scope`=&lt;Namespaced\|Cluster&gt;  | EXPERIMENTAL |
| kube_customresourcedefinition_status_condition | Gauge | `customresourcedefinition`=&lt;customresourcedefinition-name&gt; <br> `status`=&lt;false\|true\|unknown&gt; <br> `condition`=&lt;Established\|NamesAccepted\|Terminating&gt;  | EXPERIMENTAL |

