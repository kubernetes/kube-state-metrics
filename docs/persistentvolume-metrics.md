# PersistentVolume Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_persistentvolume_annotations | Gauge | `persistentvolume`=&lt;persistentvolume-name&gt; <br> `annotation_PERSISTENTVOLUME_ANNOTATION`=&lt;PERSISTENTVOLUME_ANNOTATION&gt;  | EXPERIMENTAL |
| kube_persistentvolume_capacity_bytes | Gauge | `persistentvolume`=&lt;pv-name&gt; | STABLE |
| kube_persistentvolume_status_phase | Gauge | `persistentvolume`=&lt;pv-name&gt; <br>`phase`=&lt;Bound\|Failed\|Pending\|Available\|Released&gt;| STABLE |
| kube_persistentvolume_claim_ref | Gauge | `persistentvolume`=&lt;pv-name&gt; <br>`claim_namespace`=&lt;<namespace>&gt; <br>`name`=&lt;<name>&gt; | STABLE |
| kube_persistentvolume_labels | Gauge | `persistentvolume`=&lt;persistentvolume-name&gt; <br> `label_PERSISTENTVOLUME_LABEL`=&lt;PERSISTENTVOLUME_LABEL&gt;  | STABLE |
| kube_persistentvolume_info | Gauge | `persistentvolume`=&lt;pv-name&gt; <br> `storageclass`=&lt;storageclass-name&gt; <br> `gce_persistent_disk_name`=&lt;pd-name&gt; <br> `ebs_volume_id`=&lt;ebs-volume-id&gt; <br> `azure_disk_name`=&lt;azure-disk-name&gt; <br> `fc_wwids`=&lt;fc-wwids-comma-separated&gt; <br> `fc_lun`=&lt;fc-lun&gt; <br> `fc_target_wwns`=&lt;fc-target-wwns-comma-separated&gt; <br> `iscsi_target_portal`=&lt;iscsi-target-portal&gt; <br> `iscsi_iqn`=&lt;iscsi-iqn&gt; <br> `iscsi_lun`=&lt;iscsi-lun&gt; <br> `iscsi_initiator_name`=&lt;iscsi-initiator-name&gt; <br> `nfs_server`=&lt;nfs-server&gt; <br> `nfs_path`=&lt;nfs-path&gt; | STABLE |

