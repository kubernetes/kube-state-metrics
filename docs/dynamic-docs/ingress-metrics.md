# Ingress Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_ingress_info | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt; | STABLE |
| kube_ingress_labels | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt; <br> `label_INGRESS_LABEL`=&lt;INGRESS_LABEL&gt; | STABLE |
| kube_ingress_created  | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt; | STABLE |
| kube_ingress_metadata_resource_version  | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt; | EXPERIMENTAL |
| kube_ingress_path | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt; <br> `host`=&lt;ingress-host&gt; <br> `path`=&lt;ingress-path&gt; <br> `service_name`=&lt;service name for the path&gt; <br> `service_port`=&lt;service port for the path&gt; | STABLE |
| kube_ingress_tls | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt; <br> `tls_host`=&lt;tls hostname&gt; <br> `secret`=&lt;tls secret name&gt;| STABLE |
