# PersistentVolume Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_persistentvolume_capacity_bytes | Gauge | `persistentvolume`=&lt;pv-name&gt; | STABLE |
| kube_persistentvolume_status_phase | Gauge | `persistentvolume`=&lt;pv-name&gt; <br>`phase`=&lt;Bound\|Failed\|Pending\|Available\|Released&gt;| STABLE |
| kube_persistentvolume_labels | Gauge | `persistentvolume`=&lt;persistentvolume-name&gt; <br> `label_PERSISTENTVOLUME_LABEL`=&lt;PERSISTENTVOLUME_LABEL&gt;  | STABLE |
| kube_persistentvolume_info | Gauge | `persistentvolume`=&lt;pv-name&gt; <br> `storageclass`=&lt;storageclass-name&gt; <br> `gce_persistent_disk_name`=&lt;pd-name&gt; <br> `ebs_volume_id`=&lt;ebs-volume-id&gt; | STABLE |

