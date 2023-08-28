# ClusterRole Metrics

| Metric name                                | Metric type | Description                                                                                                               | Labels/tags                            | Status       |
| ------------------------------------------ | ----------- | ------------------------------------------------------------------------------------------------------------------------- | -------------------------------------- | ------------ |
| kube_clusterrole_annotations               | Gauge       | Kubernetes annotations converted to Prometheus labels controlled via [--metric-annotations-allowlist](./cli-arguments.md) | `clusterrole`=&lt;clusterrole-name&gt; | EXPERIMENTAL |
| kube_clusterrole_labels                    | Gauge       | Kubernetes labels converted to Prometheus labels controlled via [--metric-labels-allowlist](./cli-arguments.md)           | `clusterrole`=&lt;clusterrole-name&gt; | EXPERIMENTAL |
| kube_clusterrole_info                      | Gauge       |                                                                                                                           | `clusterrole`=&lt;clusterrole-name&gt; | EXPERIMENTAL |
| kube_clusterrole_created                   | Gauge       |                                                                                                                           | `clusterrole`=&lt;clusterrole-name&gt; | EXPERIMENTAL |
| kube_clusterrole_metadata_resource_version | Gauge       |                                                                                                                           | `clusterrole`=&lt;clusterrole-name&gt; | EXPERIMENTAL |
