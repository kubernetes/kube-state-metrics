# Service Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_service_info | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; <br> `cluster_ip`=&lt;service cluster ip&gt;  | STABLE |
| kube_service_labels | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; <br> `label_SERVICE_LABEL`=&lt;SERVICE_LABEL&gt;  | STABLE |
| kube_service_created | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; | STABLE |
| kube_service_spec_type | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; <br> `type`=&lt;ClusterIP\|NodePort\|LoadBalancer\|ExternalName&gt; | STABLE |
