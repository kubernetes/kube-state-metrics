# PersistentVolumeClaim Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_persistentvolume_status_phase | Gauge | `persistentvolume`=&lt;pv-name&gt; <br> `namespace`=&lt;pvc-namespace&gt; <br>`phase`=&lt;Bound\|Failed\|Pending\|Available\|Released&gt;<br>`volume`=&lt;pvc-namespace&gt;|
