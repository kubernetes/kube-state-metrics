# Ingress Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_ingress_info | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt; | STABLE |
| kube_ingress_labels | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt; <br> `label_INGRESS_LABEL`=&lt;INGRESS_LABEL&gt; | STABLE |
| kube_ingress_created  | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt; | STABLE |
| kube_ingress_metadata_resource_version  | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt; <br> `resource_version`=&lt;ingress-resource-version&gt; | STABLE |
