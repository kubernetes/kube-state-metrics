# Horizontal Pod Autoscaler Metrics

| Metic name                       | Metric type | Labels/tags                                                   |
| -------------------------------- | ----------- | ------------------------------------------------------------- |
| kube_hpa_metadata_generation     | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; |
| kube_hpa_spec_max_replicas       | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; |
| kube_hpa_spec_min_replicas       | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; |
| kube_hpa_status_current_replicas | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; |
| kube_hpa_status_desired_replicas | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; |
