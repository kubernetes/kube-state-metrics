# PersistentVolumeClaim Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_persistentvolumeclaim_annotations | Gauge | `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `annotation_PERSISTENTVOLUMECLAIM_ANNOTATION`=&lt;PERSISTENTVOLUMECLAIM_ANNOATION&gt;  | EXPERIMENTAL |
| kube_persistentvolumeclaim_access_mode | Gauge | `access_mode`=&lt;persistentvolumeclaim-access-mode&gt; <br>`namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; | STABLE |
| kube_persistentvolumeclaim_info | Gauge | `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `storageclass`=&lt;persistentvolumeclaim-storageclassname&gt;<br>`volumename`=&lt;volumename&gt; | STABLE |
| kube_persistentvolumeclaim_labels | Gauge | `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `label_PERSISTENTVOLUMECLAIM_LABEL`=&lt;PERSISTENTVOLUMECLAIM_LABEL&gt;  | STABLE |
| kube_persistentvolumeclaim_resource_requests_storage_bytes | Gauge | `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; | STABLE |
| kube_persistentvolumeclaim_status_condition | Gauge | `namespace` =&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `type`=&lt;persistentvolumeclaim-condition-type&gt; <br> `status`=&lt;true\|false\|unknown&gt;  | EXPERIMENTAL |
| kube_persistentvolumeclaim_status_phase | Gauge | `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `phase`=&lt;Pending\|Bound\|Lost&gt; | STABLE |

Note:

- A special `<none>` string will be used if PVC has no storage class.
