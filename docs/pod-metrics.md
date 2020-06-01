# Pod Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_pod_info | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `host_ip`=&lt;host-ip&gt; <br> `pod_ip`=&lt;pod-ip&gt; <br> `node`=&lt;node-name&gt;<br> `created_by_kind`=&lt;created_by_kind&gt;<br> `created_by_name`=&lt;created_by_name&gt;<br> `uid`=&lt;pod-uid&gt;<br> `priority_class`=&lt;priority_class&gt;<br> `host_network`=&lt;host_network&gt;| STABLE |
| kube_pod_start_time | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_completion_time | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_owner | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `owner_kind`=&lt;owner kind&gt; <br> `owner_name`=&lt;owner name&gt; <br> `owner_is_controller`=&lt;whether owner is controller&gt;  | STABLE |
| kube_pod_labels | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `label_POD_LABEL`=&lt;POD_LABEL&gt;  | STABLE |
| kube_pod_status_phase | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `phase`=&lt;Pending\|Running\|Succeeded\|Failed\|Unknown&gt; | STABLE |
| kube_pod_status_ready | Gauge |  `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `condition`=&lt;true\|false\|unknown&gt; | STABLE |
| kube_pod_status_scheduled | Gauge |  `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `condition`=&lt;true\|false\|unknown&gt; | STABLE |
| kube_pod_container_info | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `image`=&lt;image-name&gt; <br> `image_id`=&lt;image-id&gt; <br> `container_id`=&lt;containerid&gt; | STABLE |
| kube_pod_container_status_waiting | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_container_status_waiting_reason | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `reason`=&lt;ContainerCreating\|CrashLoopBackOff\|ErrImagePull\|ImagePullBackOff\|CreateContainerConfigError\|InvalidImageName\|CreateContainerError&gt; | STABLE |
| kube_pod_container_status_running | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_container_status_terminated | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_container_status_terminated_reason | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `reason`=&lt;OOMKilled\|Error\|Completed\|ContainerCannotRun\|DeadlineExceeded&gt; | STABLE |
| kube_pod_container_status_last_terminated_reason | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `reason`=&lt;OOMKilled\|Error\|Completed\|ContainerCannotRun\|DeadlineExceeded&gt; | STABLE |
| kube_pod_container_status_ready | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_container_status_restarts_total | Counter | `container`=&lt;container-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `pod`=&lt;pod-name&gt; | STABLE |
| kube_pod_container_resource_requests | Gauge | `resource`=&lt;resource-name&gt; <br> `unit`=&lt;resource-unit&gt; <br> `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-name&gt; | STABLE |
| kube_pod_container_resource_limits | Gauge | `resource`=&lt;resource-name&gt; <br> `unit`=&lt;resource-unit&gt; <br> `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-name&gt; | STABLE |
| kube_pod_overhead | Gauge | `resource`=&lt;resource-name&gt; <br> `unit`=&lt;resource-unit&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | EXPERIMENTAL |
| kube_pod_created | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_deletion_timestamp | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | EXPERIMENTAL |
| kube_pod_restart_policy | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `type`=&lt;Always|Never|OnFailure&gt; | STABLE |
| kube_pod_init_container_info | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `image`=&lt;image-name&gt; <br> `image_id`=&lt;image-id&gt; <br> `container_id`=&lt;containerid&gt; | STABLE |
| kube_pod_init_container_status_waiting | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_init_container_status_waiting_reason | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `reason`=&lt;ContainerCreating\|CrashLoopBackOff\|ErrImagePull\|ImagePullBackOff\|CreateContainerConfigError&gt; | STABLE |
| kube_pod_init_container_status_running | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_init_container_status_terminated | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_init_container_status_terminated_reason | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `reason`=&lt;OOMKilled\|Error\|Completed\|ContainerCannotRun\|DeadlineExceeded&gt; | STABLE |
| kube_pod_init_container_status_last_terminated_reason | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `reason`=&lt;OOMKilled\|Error\|Completed\|ContainerCannotRun\|DeadlineExceeded&gt; | STABLE |
| kube_pod_init_container_status_ready | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_init_container_status_restarts_total | Counter | `container`=&lt;container-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `pod`=&lt;pod-name&gt; | STABLE |
| kube_pod_init_container_resource_limits | Gauge | `resource`=&lt;resource-name&gt; <br> `unit`=&lt;resource-unit&gt; <br> `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-name&gt; | STABLE |
| kube_pod_init_container_resource_requests | Gauge | `resource`=&lt;resource-name&gt; <br> `unit`=&lt;resource-unit&gt; <br> `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-name&gt; | STABLE |
| kube_pod_spec_volumes_persistentvolumeclaims_info | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `volume`=&lt;volume-name&gt;  <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-claimname&gt; | STABLE |
| kube_pod_spec_volumes_persistentvolumeclaims_readonly | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt;  <br> `volume`=&lt;volume-name&gt;  <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-claimname&gt; | STABLE |
| kube_pod_status_reason | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `reason`=&lt;NodeLost\|Evicted\|UnexpectedAdmissionError&gt; | EXPERIMENTAL |
| kube_pod_status_scheduled_time | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |
| kube_pod_status_unschedulable | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; | STABLE |

## Useful metrics queries

### How to retrieve non-standard Pod state

It is not straightforward to get the Pod states for certain cases like "Terminating" and "Unknown" since it is not stored behind a field in the `Pod.Status`.

So to mimic the [logic](https://github.com/kubernetes/kubernetes/blob/v1.17.3/pkg/printers/internalversion/printers.go#L624) used by the `kubectl` command line, you will need to compose multiple metrics.

For example:

* To get the list of pods that are in the `Unknown` state, you can run the following PromQL query: `sum(kube_pod_status_phase{phase="Unknown"}) by (namespace, pod) or (count(kube_pod_deletion_timestamp) by (namespace, pod) * sum(kube_pod_status_reason{reason="NodeLost"}) by(namespace, pod))`

* For Pods in `Terminating` state: `count(kube_pod_deletion_timestamp) by (namespace, pod) * count(kube_pod_status_reason{reason="NodeLost"} == 0) by (namespace, pod)`

Here is an example of a Prometheus rule that can be used to alert on a Pod that has been in the `Terminated` state for more than `5m`.

```yaml
groups:
- name: Pod state
  rules:
  - alert: PodsBlockInTerminatingState
    expr: count(kube_pod_deletion_timestamp) by (namespace, pod) * count(kube_pod_status_reason{reason="NodeLost"} == 0) by (namespace, pod) > 0
    for: 5m
    labels:
      severity: page
    annotations:
      summary: Pod {{labels.namespace}}/{{labels.pod}} block in Terminating state.
```
