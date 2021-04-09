# VolumeAttachment Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_volumeattachment_info | Gauge | `volumeattachment`=&lt;volumeattachment-name&gt; <br> `attacher`=&lt;attacher-name&gt; <br> `node`=&lt;node-name&gt; | EXPERIMENTAL |
| kube_volumeattachment_created | Gauge | `volumeattachment`=&lt;volumeattachment-name&gt; | EXPERIMENTAL |
| kube_volumeattachment_labels | Gauge | `volumeattachment`=&lt;volumeattachment-name&gt; <br> `label_VOLUMEATTACHMENT_LABEL`=&lt;VOLUMEATTACHMENT_LABEL&gt;  | EXPERIMENTAL |
| kube_volumeattachment_spec_source_persistentvolume | Gauge | `volumeattachment`=&lt;volumeattachment-name&gt; <br> `volumename`=&lt;persistentvolume-name&gt; | EXPERIMENTAL |
| kube_volumeattachment_status_attached | Gauge | `volumeattachment`=&lt;volumeattachment-name&gt; | EXPERIMENTAL |
| kube_volumeattachment_status_attachment_metadata | Gauge | `volumeattachment`=&lt;volumeattachment-name&gt; <br> `metadata_METADATA_KEY`=&lt;METADATA_VALUE&gt;  | EXPERIMENTAL |
