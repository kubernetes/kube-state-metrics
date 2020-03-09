# Node Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_node_info | Gauge | `node`=&lt;node-address&gt; <br> `kernel_version`=&lt;kernel-version&gt; <br> `os_image`=&lt;os-image-name&gt; <br> `container_runtime_version`=&lt;container-runtime-and-version-combination&gt; <br> `kubelet_version`=&lt;kubelet-version&gt; <br> `kubeproxy_version`=&lt;kubeproxy-version&gt; <br> `pod_cidr`=&lt;pod-cidr&gt; <br> `provider_id`=&lt;provider-id&gt; | STABLE |
| kube_node_labels | Gauge | `node`=&lt;node-address&gt; <br> `label_NODE_LABEL`=&lt;NODE_LABEL&gt;  | STABLE |
| kube_node_role | Gauge | `node`=&lt;node-address&gt; <br> `role`=&lt;NODE_ROLE&gt; | EXPERIMENTAL |
| kube_node_spec_unschedulable | Gauge | `node`=&lt;node-address&gt;| STABLE |
| kube_node_spec_taint | Gauge | `node`=&lt;node-address&gt; <br> `key`=&lt;taint-key&gt; <br> `value=`&lt;taint-value&gt; <br> `effect=`&lt;taint-effect&gt; | STABLE |
| kube_node_status_capacity | Gauge | `node`=&lt;node-address&gt; <br> `resource`=&lt;resource-name&gt; <br> `unit`=&lt;resource-unit&gt;| STABLE |
| kube_node_status_allocatable | Gauge | `node`=&lt;node-address&gt; <br> `resource`=&lt;resource-name&gt; <br> `unit`=&lt;resource-unit&gt;| STABLE |
| kube_node_status_condition | Gauge | `node`=&lt;node-address&gt; <br> `condition`=&lt;node-condition&gt; <br> `status`=&lt;true\|false\|unknown&gt; | STABLE |
| kube_node_created | Gauge | `node`=&lt;node-address&gt;| STABLE |
