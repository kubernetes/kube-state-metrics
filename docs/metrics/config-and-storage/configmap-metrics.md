# ConfigMap Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_configmap_annotations | Gauge | `configmap`=&lt;configmap-name&gt; <br> `namespace`=&lt;configmap-namespace&gt; <br> `annotation_CONFIGMAP_ANNOTATION`=&lt;CONFIGMAP_ANNOTATION&gt; | EXPERIMENTAL
| kube_configmap_labels | Gauge | `configmap`=&lt;configmap-name&gt; <br> `namespace`=&lt;configmap-namespace&gt; <br> `label_CONFIGMAP_LABEL`=&lt;CONFIGMAP_LABEL&gt; | STABLE
| kube_configmap_info | Gauge | `configmap`=&lt;configmap-name&gt; <br> `namespace`=&lt;configmap-namespace&gt; | STABLE |
| kube_configmap_created  | Gauge | `configmap`=&lt;configmap-name&gt; <br> `namespace`=&lt;configmap-namespace&gt; | STABLE |
| kube_configmap_metadata_resource_version | Gauge | `configmap`=&lt;configmap-name&gt; <br> `namespace`=&lt;configmap-namespace&gt; | EXPERIMENTAL |
