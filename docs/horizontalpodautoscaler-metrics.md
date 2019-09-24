# Horizontal Pod Autoscaler Metrics

| Metric name                       | Metric type | Labels/tags                                                   | Status |
| --------------------------------  | ----------- | ------------------------------------------------------------- | ------ |
| kube_hpa_metadata_generation      | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; | STABLE |
| kube_hpa_spec_max_replicas        | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; | STABLE |
| kube_hpa_spec_min_replicas        | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; | STABLE |
| kube_hpa_status_current_replicas  | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; | STABLE |
| kube_hpa_status_desired_replicas  | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; | STABLE |
| kube_hpa_status_condition         | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; <br> `condition`=&lt;hpa-condition&gt; <br> `status`=&lt;true\|false\|unknown&gt; | STABLE |
| kube_hpa_labels                   | Gauge       | `hpa`=&lt;hpa-name&gt; <br> `namespace`=&lt;hpa-namespace&gt; | STABLE |
