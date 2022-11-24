# Node Metrics

| Metric name| Metric type | Description | Unit (where applicable) | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------------------- | ----------- | ------ |
| kube_node_annotations | Gauge | Kubernetes annotations converted to Prometheus labels | | `node`=&lt;node-address&gt; <br> `annotation_NODE_ANNOTATION`=&lt;NODE_ANNOTATION&gt;  | EXPERIMENTAL |
| kube_node_info | Gauge |  Information about a cluster node| |`node`=&lt;node-address&gt; <br> `kernel_version`=&lt;kernel-version&gt; <br> `os_image`=&lt;os-image-name&gt; <br> `container_runtime_version`=&lt;container-runtime-and-version-combination&gt; <br> `kubelet_version`=&lt;kubelet-version&gt; <br> `kubeproxy_version`=&lt;kubeproxy-version&gt; <br> `pod_cidr`=&lt;pod-cidr&gt; <br> `provider_id`=&lt;provider-id&gt; <br> `system_uuid`=&lt;system-uuid&gt; <br> `internal_ip`=&lt;internal-ip&gt; | STABLE |
| kube_node_labels | Gauge | Kubernetes labels converted to Prometheus labels | | `node`=&lt;node-address&gt; <br> `label_NODE_LABEL`=&lt;NODE_LABEL&gt;  | STABLE |
| kube_node_role | Gauge | The role of a cluster node | | `node`=&lt;node-address&gt; <br> `role`=&lt;NODE_ROLE&gt; | EXPERIMENTAL |
| kube_node_spec_unschedulable | Gauge | Whether a node can schedule new pods | | `node`=&lt;node-address&gt;| STABLE |
| kube_node_spec_taint | Gauge | The taint of a cluster node. | |`node`=&lt;node-address&gt; <br> `key`=&lt;taint-key&gt; <br> `value=`&lt;taint-value&gt; <br> `effect=`&lt;taint-effect&gt; | STABLE |
| kube_node_status_capacity | Gauge | The total amount of resources available for a node | `cpu`=&lt;core&gt; <br> `ephemeral_storage`=&lt;byte&gt; <br> `pods`=&lt;integer&gt; <br> `attachable_volumes_*`=&lt;byte&gt; <br> `hugepages_*`=&lt;byte&gt; <br> `memory`=&lt;byte&gt; |`node`=&lt;node-address&gt; <br> `resource`=&lt;resource-name&gt; <br> `unit`=&lt;resource-unit&gt;| STABLE |
| kube_node_status_allocatable | Gauge | The amount of resources allocatable for pods (after reserving some for system daemons) | `cpu`=&lt;core&gt; <br> `ephemeral_storage`=&lt;byte&gt; <br> `pods`=&lt;integer&gt; <br> `attachable_volumes_*`=&lt;byte&gt; <br> `hugepages_*`=&lt;byte&gt; <br> `memory`=&lt;byte&gt; |`node`=&lt;node-address&gt; <br> `resource`=&lt;resource-name&gt; <br> `unit`=&lt;resource-unit&gt;| STABLE |
| kube_node_status_condition | Gauge | The condition of a cluster node | |`node`=&lt;node-address&gt; <br> `condition`=&lt;node-condition&gt; <br> `status`=&lt;true\|false\|unknown&gt; | STABLE |
| kube_node_created | Gauge | Unix creation timestamp | seconds |`node`=&lt;node-address&gt;| STABLE |
| kube_node_deletion_timestamp | Gauge | Unix creation timestamp | seconds |`node`=&lt;node-address&gt;| EXPERIMENTAL |
