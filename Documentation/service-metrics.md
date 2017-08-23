# Service Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_service_info | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt;  |
| kube_service_labels | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; <br> `label_SERVICE_LABEL`=&lt;SERVICE_LABEL&gt;  |
| kube_service_spec_service_type | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; <br> `clusterIP`=&lt;service cluster ip&gt; `type`=&lt;ClusterIP\|NodePort\|LoadBalancer&gt; |
