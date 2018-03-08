# PersistentVolumeClaim Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_persistentvolume_status_phase | Gauge | `persistentvolume`=&lt;pv-name&gt; <br> `namespace`=&lt;pv-namespace&gt; <br>`phase`=&lt;Bound\|Failed\|Pending\|Available\|Released&gt;|
| kube_persistentvolume_labels | Gauge | `persistentvolume`=&lt;persistentvolume-name&gt; <br> `namespace`=&lt;persistentvolume-namespace&gt; <br> `label_PERSISTENTVOLUME_LABEL`=&lt;PERSISTENTVOLUME_LABEL&gt;  |
| kube_persistentvolume_info | Gauge | `persistentvolume`=&lt;pv-name&gt; <br> `namespace`=&lt;pv-namespace&gt;<br> `storageclass`=&lt;storageclass-name&gt; |

