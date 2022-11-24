# IngressClass Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_ingressclass_annotations | Gauge | `ingressclass`=&lt;ingressclass-name&gt; <br> `annotation_INGRESSCLASS_ANNOTATION`=&lt;INGRESSCLASS_ANNOTATION&gt; | EXPERIMENTAL |
| kube_ingressclass_info | Gauge | `ingressclass`=&lt;ingressclass-name&gt; <br> `controller`=&lt;ingress-controller-name&gt; <br> | EXPERIMENTAL |
| kube_ingressclass_labels | Gauge | `ingressclass`=&lt;ingressclass-name&gt; <br> `label_INGRESSCLASS_LABEL`=&lt;INGRESSCLASS_LABEL&gt; | EXPERIMENTAL|
| kube_ingressclass_created  | Gauge | `ingressclass`=&lt;ingressclass-name&gt; | EXPERIMENTAL|
