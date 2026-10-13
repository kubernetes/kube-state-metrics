# ResourceSlice Metrics

| Metric name                                                   | Metric type | Description | Labels/tags | Status |
| ------------------------------------------------------------- | ----------- | ----------- | ----------- | ------ |
| kube_resourceslice_info                                       | Gauge       | Information about resource slice. | `resourceslice`=&lt;resourceslice-name&gt; <br> `driver`=&lt;driver-name&gt; <br> `pool_name`=&lt;pool-name&gt; <br> `node_name`=&lt;node-name&gt; <br> `node_selection`=&lt;node_name\|node_selector\|all_nodes\|per_device&gt; | EXPERIMENTAL |
| kube_resourceslice_created                                    | Gauge       | Unix creation timestamp | `resourceslice`=&lt;resourceslice-name&gt; | EXPERIMENTAL |
| kube_resourceslice_devices                                    | Gauge       | The total count of devices published by this resource slice. | `resourceslice`=&lt;resourceslice-name&gt; <br> `driver`=&lt;driver-name&gt; <br> `pool_name`=&lt;pool-name&gt; <br> `node_name`=&lt;node-name&gt; | EXPERIMENTAL |
