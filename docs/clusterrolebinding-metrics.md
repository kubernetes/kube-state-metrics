# ClusterRoleBinding Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_clusterrolebinding_annotations | Gauge | `clusterrolebinding`=&lt;clusterrolebinding-name&gt; | EXPERIMENTAL
| kube_clusterrolebinding_labels | Gauge | `clusterrolebinding`=&lt;clusterrolebinding-name&gt; | EXPERIMENTAL
| kube_clusterrolebinding_info | Gauge | `clusterrolebinding`=&lt;clusterrolebinding-name&gt; <br> `roleref-kind`=&lt;roleref-kind&gt; <br> `roleref-name`=&lt;roleref-name&gt; | EXPERIMENTAL
| kube_clusterrolebinding_created  | Gauge | `clusterrolebinding`=&lt;clusterrolebinding-name&gt; | EXPERIMENTAL |
| kube_clusterrolebinding_metadata_resource_version | Gauge | `clusterrolebinding`=&lt;clusterrolebinding-name&gt; | EXPERIMENTAL |
