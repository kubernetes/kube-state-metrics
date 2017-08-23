# Ingress Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_ingress_info | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt;|
| kube_ingress_metadata_generation | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;cronjob-namespace&gt;|
| kube_ingress_loadbalancer | Gauge | `ingress`=&lt;ingress-name&gt; <br> `namespace`=&lt;ingress-namespace&gt;<br> `ip`=&lt;ingress loadbalancer ip&gt;<br> `hostname`=&lt;ingress loadbalancer hostname&gt;|
