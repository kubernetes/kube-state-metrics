# Service Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_service_info | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt;  |
| kube_service_labels | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; <br> `label_SERVICE_LABEL`=&lt;SERVICE_LABEL&gt;  |
