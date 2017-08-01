# Node Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_node_info | Gauge | `node`=&lt;node-address&gt; <br> `kernel_version`=&lt;kernel-version&gt; <br> `os_image`=&lt;os-image-name&gt; <br> `container_runtime_version`=&lt;container-runtime-and-version-combination&gt; <br> `kubelet_version`=&lt;kubelet-version&gt; <br> `kubeproxy_version`=&lt;kubeproxy-version&gt; |
| kube_node_labels | Gauge | `node`=&lt;node-address&gt; <br> `label_NODE_LABEL`=&lt;NODE_LABEL&gt;  |
| kube_node_spec_unschedulable | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_phase| Gauge | `node`=&lt;node-address&gt; <br> `phase`=&lt;Pending\|Running\|Terminated&gt; |
| kube_node_status_capacity_cpu_cores | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_capacity_memory_bytes | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_capacity_pods | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_allocatable_cpu_cores | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_allocatable_memory_bytes | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_allocatable_pods | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_condition | Gauge | `node`=&lt;node-address&gt; <br> `condition`=&lt;node-condition&gt; <br> `status`=&lt;true\|false\|unknown&gt; |
