# ResourceClaimTemplate Metrics

| Metric name                                                   | Metric type | Description | Labels/tags | Status |
| ------------------------------------------------------------- | ----------- | ----------- | ----------- | ------ |
| kube_resourceclaimtemplate_created                            | Gauge       | Unix creation timestamp | `resourceclaimtemplate`=&lt;resourceclaimtemplate-name&gt; <br> `namespace`=&lt;resourceclaimtemplate-namespace&gt; | EXPERIMENTAL |
| kube_resourceclaimtemplate_annotations                        | Gauge       | Kubernetes annotations converted to Prometheus labels controlled via [--metric-annotations-allowlist](../../developer/cli-arguments.md) | `resourceclaimtemplate`=&lt;resourceclaimtemplate-name&gt; <br> `namespace`=&lt;resourceclaimtemplate-namespace&gt; <br> `annotation_RESOURCECLAIMTEMPLATE_ANNOTATION`=&lt;RESOURCECLAIMTEMPLATE_ANNOTATION&gt; | EXPERIMENTAL |
| kube_resourceclaimtemplate_labels                             | Gauge       | Kubernetes labels converted to Prometheus labels controlled via [--metric-labels-allowlist](../../developer/cli-arguments.md) | `resourceclaimtemplate`=&lt;resourceclaimtemplate-name&gt; <br> `namespace`=&lt;resourceclaimtemplate-namespace&gt; <br> `label_RESOURCECLAIMTEMPLATE_LABEL`=&lt;RESOURCECLAIMTEMPLATE_LABEL&gt; | EXPERIMENTAL |
