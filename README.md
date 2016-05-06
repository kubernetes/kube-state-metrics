# Overview

kube-state-metrics generates metrics about the state of the object inside of a
Kubernetes cluster. It is not focused on the health of the Kubernetes
components individually, but rather on the health of the various objects
inside, such as deployments, nodes and pods.

*Requires Kubernetes 1.2+*

# Usage

Simply build and run kube-state-metrics inside a Kubernetes pod which has a
service account token that has read-only access to the Kubernetes cluster.

## Metrics

There are many more metrics we could report, but this first pass is focused on
those that could result in actionable alerts. Please contribute PR's for
additional metrics!

### WARNING: THESE METRIC/TAG NAMES ARE UNSTABLE AND MAY CHANGE IN A FUTURE RELEASE.

* nodes ready=&lt;true|false&gt;
* deployment_replicas name=&lt;deployment-name&gt; namespace=&lt;deployment-namespace&gt;
* deployment_replicas_available name=&lt;deployment-name&gt; namespace=&lt;deployment-namespace&gt;
* container_restarts name=&lt;container-name&gt; namespace=&lt;pod-namespace&gt; pod_name=&lt;pod-name&gt;

# Development

When developing, test a metric dump against your local Kubernetes cluster by running:

```
go run main.go --apiserver=&lt;APISERVER-HERE&gt; --in-cluster=false --port=&lt;APISERVER-PORT&gt; --dry-run
```
