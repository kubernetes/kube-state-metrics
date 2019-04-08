# Overview

[![Build Status](https://travis-ci.org/kubernetes/kube-state-metrics.svg?branch=master)](https://travis-ci.org/kubernetes/kube-state-metrics)  [![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes/kube-state-metrics)](https://goreportcard.com/report/github.com/kubernetes/kube-state-metrics)

kube-state-metrics is a simple service that listens to the Kubernetes API
server and generates metrics about the state of the objects. (See examples in
the Metrics section below.) It is not focused on the health of the individual
Kubernetes components, but rather on the health of the various objects inside,
such as deployments, nodes and pods.

kube-state-metrics is about generating metrics from Kubernetes API objects
without modification. This ensures, that features provided by kube-state-metrics
have the same grade of stability as the Kubernetes API objects themselves. In
turn this means, that kube-state-metrics in certain situations may not show the
exact same values as kubectl, as kubectl applies certain heuristics to display
comprehensible messages. kube-state-metrics exposes raw data unmodified from the
Kubernetes API, this way users have all the data they require and perform
heuristics as they see fit.

The metrics are exported on the HTTP endpoint `/metrics` on the listening port
(default 80). They are served as plaintext. They are designed to be consumed
either by Prometheus itself or by a scraper that is compatible with scraping a
Prometheus client endpoint. You can also open `/metrics` in a browser to see
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
- [kube-state-metrics vs. metrics-server](#kube-state-metrics-vs-metrics-server)
- [Setup](#setup)
  - [Building the Docker container](#building-the-docker-container)
- [Usage](#usage)
  - [Kubernetes Deployment](#kubernetes-deployment)
  - [Development](#development)

### Versioning

#### Kubernetes Version

kube-state-metrics uses [`client-go`](https://github.com/kubernetes/client-go) to talk with
Kubernetes clusters. The supported Kubernetes cluster version is determined by `client-go`.
The compatibility matrix for client-go and Kubernetes cluster can be found
[here](https://github.com/kubernetes/client-go#compatibility-matrix).
All additional compatibility is only best effort, or happens to still/already be supported.

#### Compatibility matrix
At most 5 kube-state-metrics releases will be recorded below.

| kube-state-metrics | client-go  | **Kubernetes 1.10** | **Kubernetes 1.11** | **Kubernetes 1.12** | **Kubernetes 1.13** | **Kubernetes 1.14** |
|--------------------|------------|---------------------|---------------------|---------------------|---------------------|---------------------|
| **v1.2.0**         |  v6.0.0    |         ✓           |         ✓           |         ✓           |         -           |         -           |
| **v1.3.1**         |  v6.0.0    |         ✓           |         ✓           |         ✓           |         -           |         -           |
| **v1.4.0**         |  v8.0.0    |         ✓           |         ✓           |         ✓           |         -           |         -           |
| **v1.5.0**         |  v8.0.0    |         ✓           |         ✓           |         ✓           |         -           |         -           |
| **v1.6.0**         |  v11.0.0   |         ✓           |         ✓           |         ✓           |         ✓           |         ✓           |
| **master**         |  v11.0.0   |         ✓           |         ✓           |         ✓           |         ✓           |         ✓           |
- `✓` Fully supported version range.
- `-` The Kubernetes cluster has features the client-go library can't use (additional API objects, etc).

#### Resource group version compatibility
Resources in Kubernetes can evolve, i.e., the group version for a resource may change from alpha to beta and finally GA
in different Kubernetes versions. For now, kube-state-metrics will only use the oldest API available in the latest
release.

#### Container Image

The latest container image can be found at:
* `quay.io/coreos/kube-state-metrics:v1.5.0`
* `k8s.gcr.io/kube-state-metrics:v1.5.0`

**Note**:
The recommended docker registry for kube-state-metrics is `quay.io`. kube-state-metrics on
`gcr.io` is only maintained on best effort as it requires external help from Google employees.

### Metrics Documentation

There are many more metrics we could report, but this first pass is focused on
those that could be used for actionable alerts. Please contribute PR's for
additional metrics!

> WARNING: THESE METRIC/TAG NAMES ARE UNSTABLE AND MAY CHANGE IN A FUTURE RELEASE.
> For now, the following metrics and collectors
>
> **metrics**
>	* kube_pod_container_resource_requests_nvidia_gpu_devices
>	* kube_pod_container_resource_limits_nvidia_gpu_devices
>	* kube_node_status_capacity_nvidia_gpu_cards
>	* kube_node_status_allocatable_nvidia_gpu_cards
>
>	are removed in kube-state-metrics v1.4.0.
>
> Any collectors and metrics based on alpha Kubernetes APIs are excluded from any stability guarantee,
> which may be changed at any given release.

See the [`docs`](docs) directory for more information on the exposed metrics.

### Kube-state-metrics self metrics
kube-state-metrics exposes its own general process metrics under `--telemetry-host` and `--telemetry-port` (default 81).

### Resource recommendation

Resource usage for kube-state-metrics changes with the Kubernetes objects(Pods/Nodes/Deployments/Secrets etc.) size of the cluster.
To some extent, the Kubernetes objects in a cluster are in direct proportion to the node number of the cluster.
[addon-resizer](https://github.com/kubernetes/autoscaler/tree/master/addon-resizer)
can watch and automatically vertically scale the dependent container up and down based on the number of nodes.
Thus kube-state-metrics uses `addon-resizer` to automatically scale its resource request. To learn more about the usage of
`addon-resizer`, please go to its [README](https://github.com/kubernetes/autoscaler/tree/master/addon-resizer/README.md#nanny-program-and-arguments).

As a general rule, you should allocate

* 200MiB memory
* 0.1 cores

For clusters of more than 100 nodes, allocate at least

* 2MiB memory per node
* 0.001 cores per node

These numbers are based on [scalability tests](https://github.com/kubernetes/kube-state-metrics/issues/124#issuecomment-318394185) at 30 pods per node.

Note that if CPU limits are set too low, kube-state-metrics' internal queues will not be able to be worked off quickly enough, resulting in increased memory consumption as the queue length grows. If you experience problems resulting from high memory allocation, try increasing the CPU limits.

### kube-state-metrics vs. metrics-server

The [metrics-server](https://github.com/kubernetes-incubator/metrics-server)
is a project that has been inspired by
[Heapster](https://github.com/kubernetes-retired/heapster) and is implemented
to serve the goals of the [Kubernetes Monitoring
Pipeline](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/monitoring_architecture.md).
It is a cluster level component which periodically scrapes metrics from all
Kubernetes nodes served by Kubelet through Summary API. The metrics are
aggregated, stored in memory and served in [Metrics API
format](https://git.k8s.io/metrics/pkg/apis/metrics/v1alpha1/types.go). The
metric-server stores the latest values only and is not responsible for
forwarding metrics to third-party destinations.

kube-state-metrics is focused on generating completely new metrics from
Kubernetes' object state (e.g. metrics based on deployments, replica sets,
etc.). It holds an entire snapshot of Kubernetes state in memory and
continuously generates new metrics based off of it. And just like the
metric-server it too is not responsibile for exporting its metrics anywhere.

Having kube-state-metrics as a separate project also enables access to these
metrics from monitoring systems such as Prometheus.

### Setup

Install this project to your `$GOPATH` using `go get`:

```
go get k8s.io/kube-state-metrics
```

#### Building the Docker container

Simply run the following command in this root folder, which will create a
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
kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud info --format='value(config.account)')
```

Note that your GCP identity is case sensitive but `gcloud info` as of Google Cloud SDK 221.0.0 is not. This means that if your IAM member contains capital letters, the above one-liner may not work for you. If you have 403 forbidden responses after running the above command and kubectl apply -f kubernetes, check the IAM member associated with your account at https://console.cloud.google.com/iam-admin/iam?project=PROJECT_ID. If it contains capital letters, you may need to set the --user flag in the command above to the case-sensitive role listed at https://console.cloud.google.com/iam-admin/iam?project=PROJECT_ID.

After running the above, if you see `Clusterrolebinding "cluster-admin-binding" created`, then you are able to continue with the setup of this service.

#### Development

When developing, test a metric dump against your local Kubernetes cluster by
running:

> Users can override the apiserver address in KUBE-CONFIG file with `--apiserver` command line.

	go install
	kube-state-metrics --port=8080 --telemetry-port=8081 --kubeconfig=<KUBE-CONFIG> --apiserver=<APISERVER>

Then curl the metrics endpoint

	curl localhost:8080/metrics

To run the e2e tests locally see the documentation in [tests/README.md](./tests/README.md).
