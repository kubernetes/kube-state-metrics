# Deployment Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_deployment_status_replicas | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; | STABLE |
| kube_deployment_status_replicas_available | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; | STABLE |
| kube_deployment_status_replicas_unavailable | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; | STABLE |
| kube_deployment_status_replicas_updated | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; | STABLE |
| kube_deployment_status_observed_generation | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; | STABLE |
| kube_deployment_status_condition | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; <br> `condition`=&lt;deployment-condition&gt; <br> `status`=&lt;true\|false\|unknown&gt; | STABLE |
| kube_deployment_spec_replicas | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; | STABLE |
| kube_deployment_spec_paused | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; | STABLE |
| kube_deployment_spec_strategy_rollingupdate_max_unavailable | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; | STABLE |
| kube_deployment_spec_strategy_rollingupdate_max_surge | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; | STABLE |
| kube_deployment_metadata_generation | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; | STABLE |
| kube_deployment_labels | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; <br> `label_DEPLOYMENT_LABEL`=&lt;DEPLOYMENT_LABEL&gt; | STABLE |
| kube_deployment_created | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; | STABLE |
