# ServiceAccount Metrics

| Metric name                           | Metric type | Description                                                                                                                             | Unit (where applicable) | Labels/tags                                                                                                                                                                                        | Status       |
| ------------------------------------- | ----------- | --------------------------------------------------------------------------------------------------------------------------------------- | ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| kube_serviceaccount_info              | Gauge       | Information about a service account                                                                                                     |                         | `namespace`=&lt;serviceaccount-namespace&gt; <br> `serviceaccount`=&lt;serviceaccount-name&gt; <br> `uid`=&lt;serviceaccount-uid&gt; <br> `automount_token`=&lt;serviceaccount-automount-token&gt; | EXPERIMENTAL |
| kube_serviceaccount_created           | Gauge       | Unix creation timestamp                                                                                                                 |                         | `namespace`=&lt;serviceaccount-namespace&gt; <br> `serviceaccount`=&lt;serviceaccount-name&gt;                                                                                                     | EXPERIMENTAL |
| kube_serviceaccount_deleted           | Gauge       | Unix deletion timestamp                                                                                                                 |                         | `namespace`=&lt;serviceaccount-namespace&gt; <br> `serviceaccount`=&lt;serviceaccount-name&gt;                                                                                                     | EXPERIMENTAL |
| kube_serviceaccount_secret            | Gauge       | Secret being referenced by a service account                                                                                            |                         | `namespace`=&lt;serviceaccount-namespace&gt; <br> `serviceaccount`=&lt;serviceaccount-name&gt; <br> `name`=&lt;secret-name&gt;                                                                     | EXPERIMENTAL |
| kube_serviceaccount_image_pull_secret | Gauge       | Secret being referenced by a service account for the purpose of pulling images                                                          |                         | `namespace`=&lt;serviceaccount-namespace&gt; <br> `serviceaccount`=&lt;serviceaccount-name&gt; <br> `name`=&lt;secret-name&gt;                                                                     | EXPERIMENTAL |
| kube_serviceaccount_annotations       | Gauge       | Kubernetes annotations converted to Prometheus labels controlled via [--metric-annotations-allowlist](../../developer/cli-arguments.md) |                         | `namespace`=&lt;serviceaccount-namespace&gt; <br> `serviceaccount`=&lt;serviceaccount-name&gt; <br> `annotation_SERVICE_ACCOUNT_ANNOTATION`=&lt;SERVICE_ACCOUNT_ANNOTATION&gt;                     | EXPERIMENTAL |
| kube_serviceaccount_labels            | Gauge       | Kubernetes labels converted to Prometheus labels controlled via [--metric-labels-allowlist](../../developer/cli-arguments.md)           |                         | `namespace`=&lt;serviceaccount-namespace&gt; <br> `serviceaccount`=&lt;serviceaccount-name&gt; <br> `label_SERVICE_ACCOUNT_LABEL`=&lt;SERVICE_ACCOUNT_LABEL&gt;                                    | EXPERIMENTAL |

## Useful metrics queries

### How to retrieve the UID for service account metrics

The `uid` label is only exposed by `kube_serviceaccount_info`. To retrieve the UID for other service account metrics, join them with `kube_serviceaccount_info` using the following PromQL query:

```promql
kube_serviceaccount_secret
    * on(namespace, serviceaccount) group_left(uid)
      kube_serviceaccount_info
```

The join requires a unique matching `kube_serviceaccount_info` series per `(namespace, serviceaccount)`. If multiple series exist, deduplication or additional matching labels may be necessary.
