# Secret Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_secret_info | Gauge | `secret`=&lt;secret-name&gt; <br> `namespace`=&lt;secret-namespace&gt; | STABLE |
| kube_secret_type | Gauge | `secret`=&lt;secret-name&gt; <br> `namespace`=&lt;secret-namespace&gt; <br> `type`=&lt;secret-type&gt; | STABLE |
| kube_secret_labels | Gauge | `secret`=&lt;secret-name&gt; <br> `namespace`=&lt;secret-namespace&gt; <br> `label_SECRET_LABEL`=&lt;SECRET_LABEL&gt; | STABLE |
| kube_secret_created  | Gauge | `secret`=&lt;secret-name&gt; <br> `namespace`=&lt;secret-namespace&gt; | STABLE |
| kube_secret_metadata_resource_version  | Gauge | `secret`=&lt;secret-name&gt; <br> `namespace`=&lt;secret-namespace&gt; <br> `resource_version`=&lt;secret-resource-version&gt; | STABLE |
