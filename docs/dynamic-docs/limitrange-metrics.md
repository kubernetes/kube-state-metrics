# LimitRange Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_limitrange | Gauge | `limitrange`=&lt;limitrange-name&gt; <br> `namespace`=&lt;namespace&gt; <br> `resource`=&lt;ResourceName&gt; <br> `type`=&lt;Pod\|Container\|PersistentVolumeClaim&gt; <br> `constraint`=&lt;constraint&gt;| STABLE |
| kube_limitrange_created | Gauge | `limitrange`=&lt;limitrange-name&gt; <br> `namespace`=&lt;namespace&gt; | STABLE |
