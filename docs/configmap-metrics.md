# ConfigMap Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_configmap_info | Gauge | `configmap`=&lt;configmap-name&gt; <br> `namespace`=&lt;configmap-namespace&gt; | STABLE |
| kube_configmap_created  | Gauge | `configmap`=&lt;configmap-name&gt; <br> `namespace`=&lt;configmap-namespace&gt; | STABLE |
| kube_configmap_metadata_resource_version | Gauge | `configmap`=&lt;configmap-name&gt; <br> `namespace`=&lt;configmap-namespace&gt; <br> `resource_version`=&lt;configmap-resource-version&gt; | STABLE |
