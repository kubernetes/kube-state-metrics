# DeviceClass Metrics

| Metric name                                                   | Metric type | Description | Labels/tags | Status |
| ------------------------------------------------------------- | ----------- | ----------- | ----------- | ------ |
| kube_deviceclass_info                                         | Gauge       | Information about device class. | `deviceclass`=&lt;deviceclass-name&gt; <br> `extended_resource_name`=&lt;extended-resource-name&gt; | EXPERIMENTAL |
| kube_deviceclass_created                                      | Gauge       | Unix creation timestamp | `deviceclass`=&lt;deviceclass-name&gt; | EXPERIMENTAL |
| kube_deviceclass_annotations                                  | Gauge       | Kubernetes annotations converted to Prometheus labels controlled via [--metric-annotations-allowlist](../../developer/cli-arguments.md) | `deviceclass`=&lt;deviceclass-name&gt; <br> `annotation_DEVICECLASS_ANNOTATION`=&lt;DEVICECLASS_ANNOTATION&gt; | EXPERIMENTAL |
| kube_deviceclass_labels                                       | Gauge       | Kubernetes labels converted to Prometheus labels controlled via [--metric-labels-allowlist](../../developer/cli-arguments.md) | `deviceclass`=&lt;deviceclass-name&gt; <br> `label_DEVICECLASS_LABEL`=&lt;DEVICECLASS_LABEL&gt; | EXPERIMENTAL |
