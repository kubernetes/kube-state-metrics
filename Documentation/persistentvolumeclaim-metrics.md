# PersistentVolumeClaim Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_persistentvolumeclaim_status_phase| Gauge | `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `storageclass`=&lt;persistentvolumeclaim-storageclassname&gt; <br> `phase`=&lt;Pending\|Bound\|Lost&gt; |
| kube_persistentvolumeclaim_resource_requests_storage| Gauge | `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `storageclass`=&lt;persistentvolumeclaim-storageclassname&gt; |

Note:

- A special `<none>` string will be used if PVC has no storage class.
