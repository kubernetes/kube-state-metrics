# Pod Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_pod_info | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `host_ip`=&lt;host-ip&gt; <br> `pod_ip`=&lt;pod-ip&gt; <br> `node`=&lt;node-name&gt;<br> `created_by_kind`=&lt;created_by_kind&gt;<br> `created_by_name`=&lt;created_by_name&gt;<br> `uid`=&lt;pod-uid&gt;<br> `priority_class`=&lt;priority_class&gt;| STABLE |
| kube_pod_start_time | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; |
| kube_pod_completion_time | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_owner | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `owner_kind`=&lt;owner kind&gt; <br> `owner_name`=&lt;owner name&gt; <br> `owner_is_controller`=&lt;whether owner is controller&gt;  | STABLE |
| kube_pod_labels | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `label_POD_LABEL`=&lt;POD_LABEL&gt;  | STABLE |
| kube_pod_status_phase | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `phase`=&lt;Pending\|Running\|Succeeded\|Failed\|Unknown&gt; | STABLE |
| kube_pod_status_ready | Gauge |  `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `condition`=&lt;true\|false\|unknown&gt; | STABLE |
| kube_pod_status_scheduled | Gauge |  `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `condition`=&lt;true\|false\|unknown&gt; | STABLE |
| kube_pod_container_info | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `image`=&lt;image-name&gt; <br> `image_id`=&lt;image-id&gt; <br> `container_id`=&lt;containerid&gt; | STABLE |
| kube_pod_container_status_waiting | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_container_status_waiting_reason | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `reason`=&lt;ContainerCreating\|CrashLoopBackOff\|ErrImagePull\|ImagePullBackOff\|CreateContainerConfigError&gt; | STABLE |
| kube_pod_container_status_running | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_container_status_terminated | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_container_status_terminated_reason | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `reason`=&lt;OOMKilled\|Error\|Completed\|ContainerCannotRun&gt; | STABLE |
| kube_pod_container_status_last_terminated_reason | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `reason`=&lt;OOMKilled\|Error\|Completed\|ContainerCannotRun&gt; | STABLE |
| kube_pod_container_status_ready | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_container_status_restarts_total | Counter | `container`=&lt;container-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `pod`=&lt;pod-name&gt; | STABLE |
| kube_pod_container_resource_requests_cpu_cores | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-name&gt; | DEPRECATED |
| kube_pod_container_resource_requests | Gauge | `resource`=&lt;resource-name&gt; <br> `unit`=&lt;resource-unit&gt; <br> `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-name&gt; | STABLE |
| kube_pod_container_resource_requests_memory_bytes | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-name&gt; | DEPRECATED |
| kube_pod_container_resource_limits_cpu_cores | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-name&gt; | DEPRECATED |
| kube_pod_container_resource_limits | Gauge | `resource`=&lt;resource-name&gt; <br> `unit`=&lt;resource-unit&gt; <br> `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-name&gt; | STABLE |
| kube_pod_container_resource_limits_memory_bytes | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-name&gt; | DEPRECATED |
| kube_pod_created | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; |
| kube_pod_spec_volumes_persistentvolumeclaims_info | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `volume`=&lt;volume-name&gt;  <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-claimname&gt; | STABLE |
| kube_pod_spec_volumes_persistentvolumeclaims_readonly | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt;  <br> `volume`=&lt;volume-name&gt;  <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-claimname&gt; | STABLE |
| kube_pod_status_scheduled_time | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
