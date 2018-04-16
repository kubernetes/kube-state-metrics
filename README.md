# Overview

[![Build Status](https://travis-ci.org/kubernetes/kube-state-metrics.svg?branch=master)](https://travis-ci.org/kubernetes/kube-state-metrics)  [![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes/kube-state-metrics)](https://goreportcard.com/report/github.com/kubernetes/kube-state-metrics)

kube-state-metrics is a simple service that listens to the Kubernetes API
server and generates metrics about the state of the objects. (See examples in
the Metrics section below.) It is not focused on the health of the individual
Kubernetes components, but rather on the health of the various objects inside,
such as deployments, nodes and pods.

That kube-state-metrics is about generating metrics from Kubernetes API
objects without modification. This ensures, that features provided by
kube-state-metrics have the same grade of stability as the Kubernetes API
objects themselves. In turn this means, that kube-state-metrics in certain
situation may not show the exact same values as kubectl, as kubectl applies
certain heuristics to display comprehensible messages. kube-state-metrics
exposes raw data unmodified from the Kubernetes API, this way users has all the
data they require and perform heuristics as they see fit.

The metrics are exported through the [Prometheus golang
client](https://github.com/prometheus/client_golang) on the HTTP endpoint `/metrics` on
the listening port (default 80). They are served either as plaintext or
protobuf depending on the `Accept` header. They are designed to be consumed
either by Prometheus itself or by a scraper that is compatible with scraping
a Prometheus client endpoint. You can also open `/metrics` in a browser to see
the raw metrics.

## Table of Contents

- [Versioning](#versioning)
  - [Kubernetes Version](#kubernetes-version)
  - [Compatibility matrix](#compatibility-matrix)
  - [Resource group version compatibility](#resource-group-version-compatibility)
  - [Container Image](#container-image)
- [Metrics Documentation](#metrics-documentation)
- [Kube-state-metrics self metrics](#kube-state-metrics-self-metrics)
- [Resource recommendation](#resource-recommendation)
- [kube-state-metrics vs. Heapster](#kube-state-metrics-vs-heapster)
- [Setup](#setup)
  - [Building the Docker container](#building-the-docker-container)
- [Usage](#usage)
  - [Kubernetes Deployment](#kubernetes-deployment)
  - [Deployment](#deployment)

### Versioning

#### Kubernetes Version

kube-state-metrics uses [`client-go`](https://github.com/kubernetes/client-go) to talk with
Kubernetes clusters. The supported Kubernetes cluster version is determined by `client-go`.
The compatibility matrix for client-go and Kubernetes cluster can be found
[here](https://github.com/kubernetes/client-go#compatibility-matrix).
All additional compatibility is only best effort, or happens to still/already be supported.

#### Compatibility matrix
At most 5 kube-state-metrics releases will be recorded below.

| kube-state-metrics | client-go | **Kubernetes 1.4**  | **Kubernetes 1.5** | **Kubernetes 1.6** | **Kubernetes 1.7** | **Kubernetes 1.8** | **Kubernetes 1.9** |
|--------------------|-----------|---------------------|--------------------|--------------------|--------------------|--------------------|--------------------|
| **v0.5.0** |  v2.0.0-alpha.1   |          ✓          |         ✓          |        -           |         -          |         -          |         -          |
| **v1.0.x** |  4.0.0-beta.0     |          ✓          |         ✓          |        ✓           |         ✓          |         -          |         -          |
| **v1.1.0** |  release-5.0      |          ✓          |         ✓          |        ✓           |         ✓          |         ✓          |         -          |
| **v1.2.0** |  v6.0.0           |          ✓          |         ✓          |        ✓           |         ✓          |         ✓          |         ✓          |
| **v1.3.0** |  v6.0.0           |          ✓          |         ✓          |        ✓           |         ✓          |         ✓          |         ✓          |
| **master** |  v6.0.0           |          ✓          |         ✓          |        ✓           |         ✓          |         ✓          |         ✓          |
- `✓` Fully supported version range.
- `-` The Kubernetes cluster has features the client-go library can't use (additional API objects, etc).

#### Resource group version compatibility
Resources in Kubernetes can evolve, i.e., the group version for a resource may change from alpha to beta and finally GA
in different Kubernetes versions. As for now, kube-state-metrics will only use the oldest API available in the latest
release.

#### Container Image

The latest container image can be found at:
* `quay.io/coreos/kube-state-metrics:v1.3.0`
* `k8s.gcr.io/kube-state-metrics:v1.3.0`

**Note**:
The recommended docker registry for kube-state-metrics is `quay.io`. kube-state-metrics on
`gcr.io` is only maintained on best effort as it requires external help from Google employees.

### Metrics Documentation

There are many more metrics we could report, but this first pass is focused on
those that could be used for actionable alerts. Please contribute PR's for
additional metrics!

> WARNING: THESE METRIC/TAG NAMES ARE UNSTABLE AND MAY CHANGE IN A FUTURE RELEASE.
> For now the following metrics and collectors
>
> **metrics**
>	* kube_pod_container_resource_requests_nvidia_gpu_devices
>	* kube_pod_container_resource_limits_nvidia_gpu_devices
>	* kube_node_status_capacity_nvidia_gpu_cards
>	* kube_node_status_allocatable_nvidia_gpu_cards
>
>	are deprecated and will be completely removed when the Kubernetes accelerator feature support is removed in version v1.11. (Kubernetes accelerator support is already deprecated in v1.10).
>
> Any collectors and metrics based on alpha Kubernetes APIs are excluded from any stability guarantee,
> which may be changed at any given release.

See the [`Documentation`](Documentation) directory for documentation of the exposed metrics.

### Kube-state-metrics self metrics
kube-state-metrics exposes its own metrics under `--telemetry-host` and `--telemetry-port` (default 81).

| Metric name | Metric type | Description | Labels/tags |
| ----------- | ----------- | ----------- | ----------- |
| ksm_scrape_error_total   | Counter | Total scrape errors encountered when scraping a resource | `resource`=&lt;resource name&gt; |
| ksm_resources_per_scrape | Summary | Number of resources returned per scrape | `resource`=&lt;resource name&gt; |

### Resource recommendation

Resource usage changes with the size of the cluster. As a general rule, you should allocate

* 200MiB memory
* 0.1 cores

For clusters of more than 100 nodes, allocate at least

* 2MiB memory per node
* 0.001 cores per node

These numbers are based on [scalability tests](https://github.com/kubernetes/kube-state-metrics/issues/124#issuecomment-318394185) at 30 pods per node.

### kube-state-metrics vs. Heapster

[Heapster](https://github.com/kubernetes/heapster) is a project which fetches
metrics (such as CPU and memory utilization) from the Kubernetes API server and
nodes and sends them to various time-series backends such as InfluxDB or Google
Cloud Monitoring. Its most important function right now is implementing certain
metric APIs that Kubernetes components like the horizontal pod auto-scaler
query to make decisions.

While Heapster's focus is on forwarding metrics already generated by
Kubernetes, kube-state-metrics is focused on generating completely new metrics
from Kubernetes' object state (e.g. metrics based on deployments, replica sets,
etc.). The reason not to extend Heapster with kube-state-metrics' abilities is
because the concerns are fundamentally different: Heapster only needs to fetch,
format and forward metrics that already exist, in particular from Kubernetes
components, and write them into sinks, which are the actual monitoring
systems. kube-state-metrics, in contrast, holds an entire snapshot of
Kubernetes state in memory and continuously generates new metrics based off of
it but has no responsibility for exporting its metrics anywhere.

In other words, kube-state-metrics itself is designed to be another source for
Heapster (although this is not currently the case).

Additionally, some monitoring systems such as Prometheus do not use Heapster
for metric collection at all and instead implement their own, but
[Prometheus can scrape metrics from heapster itself to alert on Heapster's health](https://github.com/kubernetes/heapster/blob/master/docs/debugging.md#debuging).
Having kube-state-metrics as a separate project enables access to these metrics
from those monitoring systems.

### Setup

Install this project to your `$GOPATH` using `go get`:

```
go get k8s.io/kube-state-metrics
```

#### Building the Docker container

Simple run the following command in this root folder, which will create a
self-contained, statically-linked binary and build a Docker image:
```
make container
```

### Usage

Simply build and run kube-state-metrics inside a Kubernetes pod which has a
service account token that has read-only access to the Kubernetes cluster.

#### Kubernetes Deployment

To deploy this project, you can simply run `kubectl apply -f kubernetes` and a
Kubernetes service and deployment will be created. (Note: Adjust the apiVersion of some resource if your kubernetes cluster's version is not 1.8+, check the yaml file for more information). The service already has a
`prometheus.io/scrape: 'true'` annotation and if you added the recommended
Prometheus service-endpoint scraping [configuration](https://raw.githubusercontent.com/prometheus/prometheus/master/documentation/examples/prometheus-kubernetes.yml), Prometheus will pick it up automatically and you can start using the generated
metrics right away.

**Note:** Google Kubernetes Engine (GKE) Users - GKE has strict role permissions that will prevent the kube-state-metrics roles and role bindings from being created. To work around this, you can give your GCP identity the cluster-admin role by running the following one-liner:

```
kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud info | grep Account | cut -d '[' -f 2 | cut -d ']' -f 1)
```

After running the above, if you see `Clusterrolebinding "cluster-admin-binding" created`, then you are able to continue with the setup of this service.

#### Development

When developing, test a metric dump against your local Kubernetes cluster by
running:

> Users can override the apiserver address in KUBE-CONFIG file with `--apiserver` command line.

	go install
	kube-state-metrics --port=8080 --telemetry-port=8081 --kubeconfig=<KUBE-CONFIG> --apiserver=<APISERVER>

Then curl the metrics endpoint

	curl localhost:8080/metrics
