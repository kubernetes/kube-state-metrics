# MutatingWebhookConfiguration Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_mutatingwebhookconfiguration_info | Gauge | `mutatingwebhookconfiguration`=&lt;mutatingwebhookconfiguration-name&gt; <br> `namespace`=&lt;mutatingwebhookconfiguration-namespace&gt; | EXPERIMENTAL |
| kube_mutatingwebhookconfiguration_created  | Gauge | `mutatingwebhookconfiguration`=&lt;mutatingwebhookconfiguration-name&gt; <br> `namespace`=&lt;mutatingwebhookconfiguration-namespace&gt; | EXPERIMENTAL |
| kube_mutatingwebhookconfiguration_metadata_resource_version | Gauge | `mutatingwebhookconfiguration`=&lt;mutatingwebhookconfiguration-name&gt; <br> `namespace`=&lt;mutatingwebhookconfiguration-namespace&gt; <br> `resource_version`=&lt;mutatingwebhookconfiguration-resource-version&gt; | EXPERIMENTAL |
