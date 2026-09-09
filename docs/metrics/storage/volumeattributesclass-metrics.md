# VolumeAttributesClass Metrics

| Metric name                            | Metric type | Description                                                                                                                                  | Labels/tags                                                                                                                                                     | Status       |
| --------------------------------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------ |
| kube_volumeattributesclass_annotations | Gauge       | Kubernetes annotations converted to Prometheus labels controlled via [--metric-annotations-allowlist](../../developer/cli-arguments.md)     | `volumeattributesclass`=&lt;volumeattributesclass-name&gt; <br> `annotation_VOLUMEATTRIBUTESCLASS_ANNOTATION`=&lt;VOLUMEATTRIBUTESCLASS_ANNOTATION&gt;          | EXPERIMENTAL |
| kube_volumeattributesclass_info        | Gauge       | Information about volumeattributesclass.                                                                                                     | `volumeattributesclass`=&lt;volumeattributesclass-name&gt; <br> `driver_name`=&lt;volumeattributesclass-driverName&gt;                                          | EXPERIMENTAL |
| kube_volumeattributesclass_labels      | Gauge       | Kubernetes labels converted to Prometheus labels controlled via [--metric-labels-allowlist](../../developer/cli-arguments.md)               | `volumeattributesclass`=&lt;volumeattributesclass-name&gt; <br> `label_VOLUMEATTRIBUTESCLASS_LABEL`=&lt;VOLUMEATTRIBUTESCLASS_LABEL&gt;                         | EXPERIMENTAL |
| kube_volumeattributesclass_parameters  | Gauge       | VolumeAttributesClass parameters converted to Prometheus labels controlled via [--metric-volumeattributesclass-parameters-allowlist](../../developer/cli-arguments.md) | `volumeattributesclass`=&lt;volumeattributesclass-name&gt; <br> `parameter_VOLUMEATTRIBUTESCLASS_PARAMETER`=&lt;VOLUMEATTRIBUTESCLASS_PARAMETER&gt;             | EXPERIMENTAL |
| kube_volumeattributesclass_created     | Gauge       |                                                                                                                                                | `volumeattributesclass`=&lt;volumeattributesclass-name&gt;                                                                                                       | EXPERIMENTAL |

## kube_volumeattributesclass_parameters default allowlist

Unlike the annotations and labels allowlists, `--metric-volumeattributesclass-parameters-allowlist` ships with a
non-empty default so that the most common CSI driver parameters are exposed out of the box:

```text
iops, throughput, type, provisioned-iops, provisioned-throughput, skuName, iopsReadWrite, bandwidthMBps
```

This covers the AWS EBS, GCP Persistent Disk, and Azure Disk CSI drivers:

* AWS EBS CSI driver (`ebs.csi.aws.com`): `type`, `iops`, `throughput`

  ```yaml
  apiVersion: storage.k8s.io/v1
  kind: VolumeAttributesClass
  metadata:
    name: ebs-fast
  driverName: ebs.csi.aws.com
  parameters:
    type: gp3
    iops: "6000"
    throughput: "250"
  ```

* GCP Persistent Disk CSI driver (`pd.csi.storage.gke.io`): `provisioned-iops`, `provisioned-throughput`

  ```yaml
  apiVersion: storage.k8s.io/v1
  kind: VolumeAttributesClass
  metadata:
    name: pd-extreme
  driverName: pd.csi.storage.gke.io
  parameters:
    provisioned-iops: "6000"
    provisioned-throughput: "250"
  ```

* Azure Disk CSI driver (`disk.csi.azure.com`): `skuName`, `iopsReadWrite`, `bandwidthMBps`

  ```yaml
  apiVersion: storage.k8s.io/v1
  kind: VolumeAttributesClass
  metadata:
    name: azuredisk-premium
  driverName: disk.csi.azure.com
  parameters:
    skuName: PremiumV2_LRS
    iopsReadWrite: "6000"
    bandwidthMBps: "250"
  ```

Other CSI drivers' parameter keys are not included by default and require explicit configuration, e.g.
`--metric-volumeattributesclass-parameters-allowlist=myCustomParam,anotherParam`. This flag takes a flat
comma-separated list of parameter keys (it is not scoped by resource name like
`--metric-labels-allowlist`/`--metric-annotations-allowlist`, since it only ever applies to
`VolumeAttributesClass` objects).
