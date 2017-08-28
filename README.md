# Overview

[![Build Status](https://travis-ci.org/kubernetes/kube-state-metrics.svg?branch=master)](https://travis-ci.org/kubernetes/kube-state-metrics)  [![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes/kube-state-metrics)](https://goreportcard.com/report/github.com/kubernetes/kube-state-metrics)

kube-state-metrics is a simple service that listens to the Kubernetes API
server and generates metrics about the state of the objects. (See examples in
the Metrics section below.) It is not focused on the health of the individual
Kubernetes components, but rather on the health of the various objects inside,
such as deployments, nodes and pods.

The metrics are exported through the [Prometheus golang
client](https://github.com/prometheus/client_golang) on the HTTP endpoint `/metrics` on
the listening port (default 8080). They are served either as plaintext or
protobuf depending on the `Accept` header. They are designed to be consumed
either by Prometheus itself or by a scraper that is compatible with scraping
a Prometheus client endpoint. You can also open `/metrics` in a browser to see
the raw metrics.

## Kubernetes Version

kube-state-metrics uses [`client-go`](https://github.com/kubernetes/client-go) to talk with
Kubernetes clusters. The supported Kubernetes cluster version is determined by `client-go`.
The compatibility matrix for client-go and Kubernetes cluster can be found 
[here](https://github.com/kubernetes/client-go#compatibility-matrix). 
All additional compatibility is only best effort, or happens to still/already be supported.
Currently, `client-go` is in version `v4.0.0-beta.0`.

## Container Image

The latest container image can be found at `quay.io/coreos/kube-state-metrics:v1.0.1`.

## Metrics Documentation

There are many more metrics we could report, but this first pass is focused on
those that could be used for actionable alerts. Please contribute PR's for
additional metrics!

> WARNING: THESE METRIC/TAG NAMES ARE UNSTABLE AND MAY CHANGE IN A FUTURE RELEASE.
> For now kube_pod_container_resource_requests_nvidia_gpu_devices and kube_pod_container_resource_limits_nvidia_gpu_devices are in alpha stage and will be deprecated when the kubernetes gpu support is final in 1.9 version.

See the [`Documentation`](Documentation) directory for documentation of the exposed metrics.

## Resource recommendation

Resource usage changes with the size of the cluster. As a general rule, you should allocate

* 200MiB memory
* 0.1 cores

For clusters of more than 100 nodes, allocate at least

* 2MiB memory per node
* 0.001 cores per node

These numbers are based on [scalability tests](https://github.com/kubernetes/kube-state-metrics/issues/124#issuecomment-318394185) at 30 pods per node.

## kube-state-metrics vs. Heapster

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

# Setup

Install this project to your `$GOPATH` using `go get`:

```
go get k8s.io/kube-state-metrics
```

## Building the Docker container

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
