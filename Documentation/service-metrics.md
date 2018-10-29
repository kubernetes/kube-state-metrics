# Service Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_service_info | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; <br> `cluster_ip`=&lt;service cluster ip&gt; <br> `external_name`=&lt;service external name&gt; <btr> `load_balancer_ip`=&lt;service load balancer ip&gt; | STABLE |
| kube_service_labels | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; <br> `label_SERVICE_LABEL`=&lt;SERVICE_LABEL&gt;  | STABLE |
| kube_service_created | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; | STABLE |
| kube_service_spec_type | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; <br> `type`=&lt;ClusterIP\|NodePort\|LoadBalancer\|ExternalName&gt; | STABLE |
| kube_service_spec_external_ip | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; <br> `external_ip`=&lt;external-ip&gt; | STABLE |
| kube_service_status_load_balancer_ingress | Gauge | `service`=&lt;service-name&gt; <br> `namespace`=&lt;service-namespace&gt; <br> `ip`=&lt;load-balancer-ingress-ip&gt; <br> `hostname`=&lt;load-balancer-ingress-hostname&gt; | STABLE |
