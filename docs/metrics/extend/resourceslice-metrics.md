# ResourceSlice Metrics

| Metric name                                                   | Metric type | Description | Labels/tags | Status |
| ------------------------------------------------------------- | ----------- | ----------- | ----------- | ------ |
| kube_resourceslice_info                                       | Gauge       | Information about resource slice. | `resourceslice`=&lt;resourceslice-name&gt; <br> `driver`=&lt;driver-name&gt; <br> `pool`=&lt;pool-name&gt; <br> `node`=&lt;node-name&gt; <br> `node_selection`=&lt;node_name\|node_selector\|all_nodes\|per_device&gt; | EXPERIMENTAL |
| kube_resourceslice_created                                    | Gauge       | Unix creation timestamp | `resourceslice`=&lt;resourceslice-name&gt; | EXPERIMENTAL |
| kube_resourceslice_devices                                    | Gauge       | The total count of devices published by this resource slice. | `resourceslice`=&lt;resourceslice-name&gt; <br> `driver`=&lt;driver-name&gt; <br> `pool`=&lt;pool-name&gt; <br> `node`=&lt;node-name&gt; | EXPERIMENTAL |
| kube_resourceslice_annotations                                | Gauge       | Kubernetes annotations converted to Prometheus labels controlled via [--metric-annotations-allowlist](../../developer/cli-arguments.md) | `resourceslice`=&lt;resourceslice-name&gt; <br> `annotation_RESOURCESLICE_ANNOTATION`=&lt;RESOURCESLICE_ANNOTATION&gt; | EXPERIMENTAL |
| kube_resourceslice_labels                                     | Gauge       | Kubernetes labels converted to Prometheus labels controlled via [--metric-labels-allowlist](../../developer/cli-arguments.md) | `resourceslice`=&lt;resourceslice-name&gt; <br> `label_RESOURCESLICE_LABEL`=&lt;RESOURCESLICE_LABEL&gt; | EXPERIMENTAL |
