# Overview

kube-state-metrics is a simple service that listens to the Kubernetes API
server and generates metrics about the state of the objects. (See examples in
the Metrics section below.) It is not focused on the health of the individual
Kubernetes components, but rather on the health of the various objects inside,
such as deployments, nodes and pods.

The metrics are exported through the [Prometheus golang
client](https://github.com/prometheus/client_golang) on the HTTP endpoint `/metrics` on
the listening port (default 80). They are served either as plaintext or
protobuf depending on the `Accept` header. They are designed to be consumed
either by Prometheus itself or by a scraper that is compatible with scraping
a Prometheus client endpoint. You can also open `/metrics` in a browser to see
the raw metrics.

*Requires Kubernetes 1.2+*

## Metrics

There are many more metrics we could report, but this first pass is focused on
those that could be used for actionable alerts. Please contribute PR's for
additional metrics!

> WARNING: THESE METRIC/TAG NAMES ARE UNSTABLE AND MAY CHANGE IN A FUTURE RELEASE.

### Node Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_node_info | Gauge | `node`=&lt;node-address&gt; <br> `kernel_version`=&lt;kernel-version&gt; <br> `os_image`=&lt;os-image-name&gt; <br> `container_runtime_version`=&lt;container-runtime-and-version-combination&gt; <br> `kubelet_version`=&lt;kubelet-version&gt; <br> `kubeproxy_version`=&lt;kubeproxy-version&gt; |
| kube_node_spec_unschedulable | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_ready| Gauge | `node`=&lt;node-address&gt; <br> `condition`=&lt;true\|false\|unknown&gt; |
| kube_node_status_out_of_disk | Gauge | `node`=&lt;node-address&gt; <br> `condition`=&lt;true\|false\|unknown&gt; |
| kube_node_status_phase| Gauge | `node`=&lt;node-address&gt; <br> `phase`=&lt;Pending\|Running\|Terminated&gt; |
| kube_node_status_capacity_cpu_cores | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_capacity_memory_bytes | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_capacity_pods | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_allocatable_cpu_cores | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_allocatable_memory_bytes | Gauge | `node`=&lt;node-address&gt;|
| kube_node_status_allocatable_pods | Gauge | `node`=&lt;node-address&gt;|

### Deployment Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_deployment_status_replicas | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; |
| kube_deployment_status_replicas_available | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; |
| kube_deployment_status_replicas_unavailable | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; |
| kube_deployment_status_replicas_updated | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; |
| kube_deployment_status_replicas_observed_generation | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; |
| kube_deployment_spec_replicas | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; |
| kube_deployment_spec_paused | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; |
| kube_deployment_metadata_generation | Gauge | `deployment`=&lt;deployment-name&gt; <br> `namespace`=&lt;deployment-namespace&gt; |

### Pod Metrics

| Metric name| Metric type | Labels/tags |
| ---------- | ----------- | ----------- |
| kube_pod_info | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `host_ip`=&lt;host-ip&gt; <br> `pod_ip`=&lt;pod-ip&gt; <br> `start_time`=&lt;date-time since kubelet acknowledged pod&gt; |
| kube_pod_status_phase | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `phase`=&lt;Pending\|Running\|Succeeded\|Failed\|Unknown&gt; |
| kube_pod_status_ready | Gauge |  `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `condition`=&lt;true\|false\|unknown&gt; |
| kube_pod_status_scheduled | Gauge |  `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `condition`=&lt;true\|false\|unknown&gt; |
| kube_pod_container_info | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `image`=&lt;image-name&gt; <br> `image_id`=&lt;image-id&gt; <br> `container_id`=&lt;containerid&gt; |
| kube_pod_container_status_waiting | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; |
| kube_pod_container_status_running | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; |
| kube_pod_container_status_terminated | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; |
| kube_pod_container_status_ready | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; |
| kube_pod_container_status_restarts | Counter | `container`=&lt;container-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `pod`=&lt;pod-name&gt; |
| kube_pod_container_requested_cpu_cores | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-address&gt; |
| kube_pod_container_requested_memory_bytes | Gauge | `container`=&lt;container-name&gt; <br> `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; <br> `node`=&lt; node-address&gt; |
| kube_pod_healthcheck_num_of_failures | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; |
| kube_pod_healthcheck_seconds_since_last_failure | Gauge | `pod`=&lt;pod-name&gt; <br> `namespace`=&lt;pod-namespace&gt; |

## kube-state-metrics vs. Heapster

[Heapster](https://github.com/kubernetes/heapster) is a project which fetches
metrics (such as CPU and memory utilization) from the Kubernetes API server and
nodes and sends them to various time-series backends such as InfluxDB or Google
Cloud Monitoring.

While Heapster's focus is on forwarding metrics already generated by
Kubernetes, kube-state-metrics is focused on generating completely new metrics
from Kubernetes' object state (e.g. metrics based on deployments, replica sets,
etc.). The reason not to extend Heapster with kube-state-metrics' abilities is
because the concerns are fundamentally different - while Heapster only needs to
fetch, format and forward metrics that already exist, kube-state-metrics holds
an entire snapshot of Kubernetes state in memory and continuously generates new
metrics based off of it but has no responsibility for exporting its metrics
anywhere.

In other words, kube-state-metrics itself is designed to be another source for
Heapster (although this is not currently the case).

Additionally, some monitoring systems such as Prometheus do not use Heapster
for metric collection at all and instead implement their own. Having
kube-state-metrics as a separate project enables access to these metrics from
those monitoring systems.

# Building the Docker container
Simple run the following command in this root folder, which will create a
self-contained, statically-linked binary and build a Docker image:
```
make container
```

# Usage

Simply build and run kube-state-metrics inside a Kubernetes pod which has a
service account token that has read-only access to the Kubernetes cluster.

## Kubernetes Deployment

To deploy this project, you can simply run `kubectl apply -f kubernetes` and a
Kubernetes service and deployment will be created. The service already has a
`prometheus.io/scrape: 'true'` annotation and if you added the recommended
Prometheus service-endpoint scraping [configuration](https://raw.githubusercontent.com/prometheus/prometheus/master/documentation/examples/prometheus-kubernetes.yml), Prometheus will pick it up automatically and you can start using the generated
metrics right away.

# Development

When developing, test a metric dump against your local Kubernetes cluster by
running:

	go install
	kube-state-metrics --apiserver=<APISERVER-HERE> --in-cluster=false --port=8080

Then curl the metrics endpoint

	curl localhost:8080/metrics
