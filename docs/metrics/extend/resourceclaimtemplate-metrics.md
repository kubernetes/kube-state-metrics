# ResourceClaimTemplate Metrics

| Metric name                                                   | Metric type | Description | Labels/tags | Status |
| ------------------------------------------------------------- | ----------- | ----------- | ----------- | ------ |
| kube_resourceclaimtemplate_info                               | Gauge       | Information about resource claim template. | `resourceclaimtemplate`=&lt;resourceclaimtemplate-name&gt; <br> `namespace`=&lt;resourceclaimtemplate-namespace&gt; | EXPERIMENTAL |
| kube_resourceclaimtemplate_created                            | Gauge       | Unix creation timestamp | `resourceclaimtemplate`=&lt;resourceclaimtemplate-name&gt; <br> `namespace`=&lt;resourceclaimtemplate-namespace&gt; | EXPERIMENTAL |
