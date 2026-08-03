# MutatingAdmissionPolicyBinding Metrics

| Metric name                                                          | Metric type | Description | Labels/tags | Status |
| -------------------------------------------------------------------- | ----------- | ----------- | ----------- | ------ |
| kube_mutatingadmissionpolicybinding_info                           | Gauge       |             | `mutatingadmissionpolicybinding`=&lt;mutatingadmissionpolicybinding-name&gt; <br> `namespace`=&lt;mutatingadmissionpolicybinding-namespace&gt; <br> `policy_name`=&lt;policy-name&gt; <br> `param_name`=&lt;param-name&gt; <br> `param_namespace`=&lt;param-namespace&gt; <br> `param_not_found_action`=&lt;param-not-found-action&gt; | EXPERIMENTAL |
| kube_mutatingadmissionpolicybinding_created                        | Gauge       |             | `mutatingadmissionpolicybinding`=&lt;mutatingadmissionpolicybinding-name&gt; <br> `namespace`=&lt;mutatingadmissionpolicybinding-namespace&gt; | EXPERIMENTAL |
