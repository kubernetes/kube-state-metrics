# PersistentVolumeClaim Metrics

| Metric name                                                | Metric type | Description                                                                                                               | Unit (where applicable) | Labels/tags                                                                                                                                                                                                                                                  | Status       |
| ---------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------ |
| kube_persistentvolumeclaim_annotations                     | Gauge       | Kubernetes annotations converted to Prometheus labels controlled via [--metric-annotations-allowlist](./cli-arguments.md) |                         | `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `annotation_PERSISTENTVOLUMECLAIM_ANNOTATION`=&lt;PERSISTENTVOLUMECLAIM_ANNOATION&gt;                                               | EXPERIMENTAL |
| kube_persistentvolumeclaim_access_mode                     | Gauge       |                                                                                                                           |                         | `access_mode`=&lt;persistentvolumeclaim-access-mode&gt; <br>`namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt;                                                                              | STABLE       |
| kube_persistentvolumeclaim_info                            | Gauge       |                                                                                                                           |                         | `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `storageclass`=&lt;persistentvolumeclaim-storageclassname&gt;<br>`volumename`=&lt;volumename&gt;<br>`volumemode`=&lt;volumemode&gt; | STABLE       |
| kube_persistentvolumeclaim_labels                          | Gauge       | Kubernetes labels converted to Prometheus labels controlled via [--metric-labels-allowlist](./cli-arguments.md)           |                         | `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `label_PERSISTENTVOLUMECLAIM_LABEL`=&lt;PERSISTENTVOLUMECLAIM_LABEL&gt;                                                             | STABLE       |
| kube_persistentvolumeclaim_resource_requests_storage_bytes | Gauge       |                                                                                                                           |                         | `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt;                                                                                                                                          | STABLE       |
| kube_persistentvolumeclaim_status_condition                | Gauge       |                                                                                                                           |                         | `namespace` =&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `type`=&lt;persistentvolumeclaim-condition-type&gt; <br> `status`=&lt;true\false\unknown&gt;                                       | EXPERIMENTAL |
| kube_persistentvolumeclaim_status_phase                    | Gauge       |                                                                                                                           |                         | `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt; <br> `phase`=&lt;Pending\Bound\Lost&gt;                                                                                                  | STABLE       |
| kube_persistentvolumeclaim_created                         | Gauge       | Unix creation timestamp                                                                                                   | seconds                 | `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt;                                                                                                                                          | EXPERIMENTAL |
| kube_persistentvolumeclaim_deletion_timestamp              | Gauge       | Unix deletion timestamp                                                                                                   | seconds                 | `namespace`=&lt;persistentvolumeclaim-namespace&gt; <br> `persistentvolumeclaim`=&lt;persistentvolumeclaim-name&gt;                                                                                                                                          | EXPERIMENTAL |

Note:

* An empty string will be used if PVC has no storage class.

## Useful metrics queries

### How to retrieve non-standard PVC state

It is not straightforward to get the PVC states for certain cases like "Terminating" since it is not stored behind a field in the `PersistentVolumeClaim.Status`.

So to mimic the [logic](https://github.com/kubernetes/kubernetes/blob/v1.27.2/pkg/printers/internalversion/printers.go#L1883) used by the `kubectl` command line, you will need to compose multiple metrics.

Here is an example of a Prometheus rule that can be used to alert on a PVC that has been in the `Terminating` state for more than `5m`.

```yaml
groups:
- name: PVC state
  rules:
  - alert: PVCBlockedInTerminatingState
    expr: kube_persistentvolumeclaim_deletion_timestamp * on(namespace, persistentvolumeclaim) group_left() (kube_persistentvolumeclaim_status_phase{phase="Bound"} == 1) > 0
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: PVC {{$labels.namespace}}/{{$labels.persistentvolumeclaim}} blocked in Terminating state.
```
