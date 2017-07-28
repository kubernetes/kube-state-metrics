# PersistentVolumeClaim Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_persistentvolumeclaim_status_phase| Gauge | `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `phase`=&lt;Pending\|Bound\|Lost&gt; |
