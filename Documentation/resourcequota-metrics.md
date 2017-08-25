# ResourceQuota Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_resourcequota | Gauge | `resourcequota`=&lt;quota-name&gt; <br> `namespace`=&lt;namespace&gt; <br> `resource`=&lt;ResourceName&gt; <br> `type`=&lt;quota-type&gt; |
| kube_resourcequota_created | Gauge | `resourcequota`=&lt;quota-name&gt; <br> `namespace`=&lt;namespace&gt; |
