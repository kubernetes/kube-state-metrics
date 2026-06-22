# ValidatingAdmissionPolicyBinding Metrics

| Metric name                                                          | Metric type | Description | Labels/tags | Status |
| -------------------------------------------------------------------- | ----------- | ----------- | ----------- | ------ |
| kube_validatingadmissionpolicybinding_info                           | Gauge       |             | `validatingadmissionpolicybinding`=&lt;validatingadmissionpolicybinding-name&gt; <br> `namespace`=&lt;validatingadmissionpolicybinding-namespace&gt; <br> `policy_name`=&lt;policy-name&gt; <br> `param_name`=&lt;param-name&gt; <br> `param_namespace`=&lt;param-namespace&gt; <br> `param_not_found_action`=&lt;param-not-found-action&gt; | EXPERIMENTAL |
| kube_validatingadmissionpolicybinding_created                        | Gauge       |             | `validatingadmissionpolicybinding`=&lt;validatingadmissionpolicybinding-name&gt; <br> `namespace`=&lt;validatingadmissionpolicybinding-namespace&gt; | EXPERIMENTAL |
| kube_validatingadmissionpolicybinding_validation_action              | Gauge       |             | `validatingadmissionpolicybinding`=&lt;validatingadmissionpolicybinding-name&gt; <br> `namespace`=&lt;validatingadmissionpolicybinding-namespace&gt; <br> `action`=&lt;action&gt; | EXPERIMENTAL |
