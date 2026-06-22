# MutatingAdmissionPolicy Metrics

| Metric name                                                   | Metric type | Description | Labels/tags | Status |
| ------------------------------------------------------------- | ----------- | ----------- | ----------- | ------ |
| kube_mutatingadmissionpolicy_info                          | Gauge       |             | `mutatingadmissionpolicy`=&lt;mutatingadmissionpolicy-name&gt; <br> `namespace`=&lt;mutatingadmissionpolicy-namespace&gt; <br> `param_api_version`=&lt;param-api-version&gt; <br> `param_kind`=&lt;param-kind&gt; <br> `failure_policy`=&lt;failure-policy&gt; <br> `reinvocation_policy`=&lt;reinvocation-policy&gt; | EXPERIMENTAL |
| kube_mutatingadmissionpolicy_created                       | Gauge       |             | `mutatingadmissionpolicy`=&lt;mutatingadmissionpolicy-name&gt; <br> `namespace`=&lt;mutatingadmissionpolicy-namespace&gt; | EXPERIMENTAL |
