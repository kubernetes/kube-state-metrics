# RoleBinding Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_rolebinding_annotations | Gauge | `rolebinding`=&lt;rolebinding-name&gt; <br> `namespace`=&lt;rolebinding-namespace&gt; | EXPERIMENTAL
| kube_rolebinding_labels | Gauge | `rolebinding`=&lt;rolebinding-name&gt; <br> `namespace`=&lt;rolebinding-namespace&gt; | EXPERIMENTAL
| kube_rolebinding_info | Gauge | `rolebinding`=&lt;rolebinding-name&gt; <br> `namespace`=&lt;rolebinding-namespace&gt; <br> `roleref-kind`=&lt;roleref-kind&gt; <br> `roleref-name`=&lt;roleref-name&gt;| EXPERIMENTAL
| kube_rolebinding_created  | Gauge | `rolebinding`=&lt;rolebinding-name&gt; <br> `namespace`=&lt;rolebinding-namespace&gt; | EXPERIMENTAL |
| kube_rolebinding_metadata_resource_version | Gauge | `rolebinding`=&lt;rolebinding-name&gt; <br> `namespace`=&lt;rolebinding-namespace&gt; | EXPERIMENTAL |
