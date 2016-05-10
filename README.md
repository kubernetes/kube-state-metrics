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

### WARNING: THESE METRIC/TAG NAMES ARE UNSTABLE AND MAY CHANGE IN A FUTURE RELEASE.

| Metric name | Labels/tags |
| ------------- | ------------- |
| nodes | ready=&lt;true|false&gt; |
| deployment_replicas | name=&lt;deployment-name&gt; namespace=&lt;deployment-namespace&gt; |
| deployment_replicas_available | name=&lt;deployment-name&gt; namespace=&lt;deployment-namespace&gt; |
| container_restarts | name=&lt;container-name&gt; namespace=&lt;pod-namespace&gt; pod_name=&lt;pod-name&gt; |

# Usage

Simply build and run kube-state-metrics inside a Kubernetes pod which has a
service account token that has read-only access to the Kubernetes cluster.

# Development

When developing, test a metric dump against your local Kubernetes cluster by running:

```
go run main.go --apiserver=<APISERVER-HERE> --in-cluster=false --port=<APISERVER-PORT> --dry-run
```
