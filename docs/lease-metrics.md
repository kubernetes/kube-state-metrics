# Lease Metrics

| Metric name           | Metric type | Description | Labels/tags                                                                                                                                                                             | Status       |
| --------------------- | ----------- | ----------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------- |
| kube_lease_owner      | Gauge       |             | `lease`=&lt;lease-name&gt; <br> `owner_kind`=&lt;onwer kind&gt; <br> `owner_name`=&lt;owner name&gt; <br> `namespace` = &lt;namespace&gt; <br> `lease_holder`=&lt;lease holder name&gt; | EXPERIMENTAL |
| kube_lease_renew_time | Gauge       |             | `lease`=&lt;lease-name&gt;  <br> `namespace` = &lt;namespace&gt;                                                                                                                        | EXPERIMENTAL |
