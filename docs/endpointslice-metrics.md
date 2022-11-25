# Endpoint Metrics

| Metric name| Metric type | Labels/tags | Status |
| ---------- | ----------- | ----------- | ----------- |
| kube_endpointslice_annotations | Gauge | `endpointslice`=&lt;endpointslice-name&gt; <br> `namespace`=&lt;endpointslice-namespace&gt; <br> `annotation_ENDPOINTSLICE_ANNOTATION`=&lt;ENDPOINTSLICE_ANNOTATION&gt;  | EXPERIMENTAL |
| kube_endpointslice_info | Gauge | `endpointslice`=&lt;endpointslice-name&gt; <br> `namespace`=&lt;endpointslice-namespace&gt;  | EXPERIMENTAL |
| kube_endpointslice_ports | Gauge | `endpointslice`=&lt;endpointslice-name&gt; <br> `namespace`=&lt;endpointslice-namespace&gt; <br> `port_name`=&lt;endpointslice-port-name&gt; <br> `port_protocol`=&lt;endpointslice-port-protocol&gt; <br> `port_number`=&lt;endpointslice-port-number&gt; | EXPERIMENTAL |
| kube_endpointslice_endpoints | Gauge | `endpointslice`=&lt;endpointslice-name&gt; <br> `namespace`=&lt;endpointslice-namespace&gt; <br> `ready`=&lt;endpointslice-ready&gt; <br> `serving`=&lt;endpointslice-serving&gt; <br> `terminating`=&lt;endpointslice-terminating&gt; <br> `hostname`=&lt;endpointslice-hostname&gt; <br> `targetref_kind`=&lt;endpointslice-targetref-kind&gt; <br> `targetref_name`=&lt;endpointslice-targetref-name&gt; <br> `targetref_namespace`=&lt;endpointslice-targetref-namespace&gt; <br> `nodename`=&lt;endpointslice-nodename&gt; <br> `endpoint_zone`=&lt;endpointslice-zone&gt;  | EXPERIMENTAL |
| kube_endpointslice_labels | Gauge | `endpointslice`=&lt;endpointslice-name&gt; <br> `namespace`=&lt;endpointslice-namespace&gt; <br> `label_ENDPOINTSLICE_LABEL`=&lt;ENDPOINTSLICE_LABEL&gt;  | EXPERIMENTAL |
| kube_endpointslice_created | Gauge | `endpointslice`=&lt;endpointslice-name&gt; <br> `namespace`=&lt;endpointslice-namespace&gt; | EXPERIMENTAL |
