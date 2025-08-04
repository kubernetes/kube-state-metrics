# Overview

[![Build Status](https://github.com/kubernetes/kube-state-metrics/workflows/continuous-integration/badge.svg)](https://github.com/kubernetes/kube-state-metrics/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/kubernetes/kube-state-metrics)](https://goreportcard.com/report/github.com/kubernetes/kube-state-metrics)
[![Go Reference](https://pkg.go.dev/badge/github.com/kubernetes/kube-state-metrics.svg)](https://pkg.go.dev/github.com/kubernetes/kube-state-metrics)
[![govulncheck](https://github.com/kubernetes/kube-state-metrics/actions/workflows/govulncheck.yml/badge.svg)](https://github.com/kubernetes/kube-state-metrics/actions/workflows/govulncheck.yml)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/8696/badge)](https://www.bestpractices.dev/projects/8696)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/kubernetes/kube-state-metrics/badge)](https://api.securityscorecards.dev/projects/github.com/kubernetes/kube-state-metrics)

kube-state-metrics (KSM) is a simple service that listens to the Kubernetes API
server and generates metrics about the state of the objects. (See examples in
the Metrics section below.) It is not focused on the health of the individual
Kubernetes components, but rather on the health of the various objects inside,
such as deployments, nodes and pods.

kube-state-metrics is about generating metrics from Kubernetes API objects
without modification. This ensures that features provided by kube-state-metrics
have the same grade of stability as the Kubernetes API objects themselves. In
turn, this means that kube-state-metrics in certain situations may not show the
exact same values as kubectl, as kubectl applies certain heuristics to display
comprehensible messages. kube-state-metrics exposes raw data unmodified from the
Kubernetes API, this way users have all the data they require and perform
heuristics as they see fit.

The metrics are exported on the HTTP endpoint `/metrics` on the listening port
(default 8080). They are served as plaintext. They are designed to be consumed
either by Prometheus itself or by a scraper that is compatible with scraping a
Prometheus client endpoint. You can also open `/metrics` in a browser to see
the raw metrics. Note that the metrics exposed on the `/metrics` endpoint
reflect the current state of the Kubernetes cluster. When Kubernetes objects
are deleted they are no longer visible on the `/metrics` endpoint.

> [!NOTE]
> This README is generated from a [template](./README.md.tpl). Please make your changes there and run `make generate-template`.

## Table of Contents

* [Versioning](#versioning)
  * [Kubernetes Version](#kubernetes-version)
  * [Compatibility matrix](#compatibility-matrix)
  * [Resource group version compatibility](#resource-group-version-compatibility)
  * [Container Image](#container-image)
* [Metrics Documentation](#metrics-documentation)
  * [ECMAScript regular expression support for allow and deny lists](#ecmascript-regular-expression-support-for-allow-and-deny-lists)
  * [Conflict resolution in label names](#conflict-resolution-in-label-names)
* [Kube-state-metrics self metrics](#kube-state-metrics-self-metrics)
* [Resource recommendation](#resource-recommendation)
* [Latency](#latency)
* [A note on costing](#a-note-on-costing)
* [kube-state-metrics vs. metrics-server](#kube-state-metrics-vs-metrics-server)
* [Scaling kube-state-metrics](#scaling-kube-state-metrics)
  * [Resource recommendation](#resource-recommendation)
  * [Horizontal sharding](#horizontal-sharding)
    * [Automated sharding](#automated-sharding)
  * [Daemonset sharding for pod metrics](#daemonset-sharding-for-pod-metrics)
* [Setup](#setup)
  * [Building the Docker container](#building-the-docker-container)
* [Usage](#usage)
  * [Kubernetes Deployment](#kubernetes-deployment)
  * [Limited privileges environment](#limited-privileges-environment)
  * [Helm Chart](#helm-chart)
  * [Development](#development)
  * [Developer Contributions](#developer-contributions)
  * [Community](#community)

### Versioning

#### Kubernetes Version

kube-state-metrics uses [`client-go`](https://github.com/kubernetes/client-go) to talk with
Kubernetes clusters. The supported Kubernetes cluster version is determined by
[`client-go`](https://github.com/kubernetes/client-go#compatibility-matrix).
All additional compatibility is only best effort, or happens to still/already be supported.

#### Compatibility matrix

At most, 5 kube-state-metrics and 5 [kubernetes releases](https://github.com/kubernetes/kubernetes/releases) will be recorded below.
Generally, it is recommended to use the latest release of kube-state-metrics. If you run a very recent version of Kubernetes, you might want to use an unreleased version to have the full range of supported resources. If you run an older version of Kubernetes, you might need to run an older version in order to have full support for all resources. Be aware, that the maintainers will only support the latest release. Older versions might be supported by interested users of the community.

| kube-state-metrics | Kubernetes client-go Version |
|--------------------|:----------------------------:|
{{ define "compat-matrix" -}}
{{- range . -}}
| **{{ .version }}**{{ strings.Repeat (conv.ToInt (math.Sub 15 (len .version))) " " }}| v{{ .kubernetes }}                        |
{{ end -}}
{{- end -}}
{{ template "compat-matrix" (datasource "config").compat }}
#### Resource group version compatibility

Resources in Kubernetes can evolve, i.e., the group version for a resource may change from alpha to beta and finally GA
in different Kubernetes versions. For now, kube-state-metrics will only use the oldest API available in the latest
release.

#### Container Image

The latest container image can be found at:
{{ define "get-latest-release" -}}
{{ (index . (math.Sub (len .) 2)).version -}}
{{ end }}
* `registry.k8s.io/kube-state-metrics/kube-state-metrics:{{ template "get-latest-release" (datasource "config").compat }}` (arch: `amd64`, `arm`, `arm64`, `ppc64le` and `s390x`)
* [Multi-architecture images](https://explore.ggcr.dev/?image=registry.k8s.io%2Fkube-state-metrics%2Fkube-state-metrics:{{ template "get-latest-release" (datasource "config").compat -}})

### Metrics Documentation

Any resources and metrics based on alpha Kubernetes APIs are excluded from any stability guarantee,
which may be changed at any given release.

See the [`docs`](docs) directory for more information on the exposed metrics.

#### Conflict resolution in label names

The `*_labels` family of metrics exposes Kubernetes labels as Prometheus labels.
As [Kubernetes](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set)
is more liberal than
[Prometheus](https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels)
in terms of allowed characters in label names,
we automatically convert unsupported characters to underscores.
For example, `app.kubernetes.io/name` becomes `label_app_kubernetes_io_name`.

This conversion can create conflicts when multiple Kubernetes labels like
`foo-bar` and `foo_bar` would be converted to the same Prometheus label `label_foo_bar`.

Kube-state-metrics automatically adds a suffix `_conflictN` to resolve this conflict,
so it converts the above labels to
`label_foo_bar_conflict1` and `label_foo_bar_conflict2`.

If you'd like to have more control over how this conflict is resolved,
you might want to consider addressing this issue on a different level of the stack,
e.g. by standardizing Kubernetes labels using an
[Admission Webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
that ensures that there are no possible conflicts.

#### ECMAScript regular expression support for allow and deny lists

Starting from [#2616](https://github.com/kubernetes/kube-state-metrics/pull/2616/files), kube-state-metrics supports ECMAScript's `regexp` for allow and deny lists. This was incorporated as a workaround for the limitations of the `regexp` package in Go, which does not support lookarounds due to their non-linear time complexity. Please note that while lookarounds are now supported for allow and deny lists, regular expressions' evaluation time is capped at a minute to prevent performance issues.

### Kube-state-metrics self metrics

kube-state-metrics exposes its own general process metrics under `--telemetry-host` and `--telemetry-port` (default 8081).

kube-state-metrics also exposes list and watch success and error metrics. These can be used to calculate the error rate of list or watch resources.
If you encounter those errors in the metrics, it is most likely a configuration or permission issue, and the next thing to investigate would be looking
at the logs of kube-state-metrics.

Example of the above mentioned metrics:

```prometheus
kube_state_metrics_list_total{resource="*v1.Node",result="success"} 1
kube_state_metrics_list_total{resource="*v1.Node",result="error"} 52
kube_state_metrics_watch_total{resource="*v1beta1.Ingress",result="success"} 1
```

kube-state-metrics also exposes some http request metrics, examples of those are:

```prometheus
http_request_duration_seconds_bucket{handler="metrics",method="get",le="2.5"} 30
http_request_duration_seconds_bucket{handler="metrics",method="get",le="5"} 30
http_request_duration_seconds_bucket{handler="metrics",method="get",le="10"} 30
http_request_duration_seconds_bucket{handler="metrics",method="get",le="+Inf"} 30
http_request_duration_seconds_sum{handler="metrics",method="get"} 0.021113919999999998
http_request_duration_seconds_count{handler="metrics",method="get"} 30
```

kube-state-metrics also exposes build and configuration metrics:

```prometheus
kube_state_metrics_build_info{branch="main",goversion="go1.15.3",revision="6c9d775d",version="v2.0.0-beta"} 1
kube_state_metrics_shard_ordinal{shard_ordinal="0"} 0
kube_state_metrics_total_shards 1
```

`kube_state_metrics_build_info` is used to expose version and other build information. For more usage about the info pattern,
please check this [blog post](https://www.robustperception.io/exposing-the-software-version-to-prometheus).
Sharding metrics expose `--shard` and `--total-shards` flags and can be used to validate
run-time configuration, see [`/examples/prometheus-alerting-rules`](./examples/prometheus-alerting-rules).

kube-state-metrics also exposes metrics about it config file and the Custom Resource State config file:

```prometheus
kube_state_metrics_config_hash{filename="crs.yml",type="customresourceconfig"} 2.38272279311849e+14
kube_state_metrics_config_hash{filename="config.yml",type="config"} 2.65285922340846e+14
kube_state_metrics_last_config_reload_success_timestamp_seconds{filename="crs.yml",type="customresourceconfig"} 1.6704882592037103e+09
kube_state_metrics_last_config_reload_success_timestamp_seconds{filename="config.yml",type="config"} 1.6704882592035313e+09
kube_state_metrics_last_config_reload_successful{filename="crs.yml",type="customresourceconfig"} 1
kube_state_metrics_last_config_reload_successful{filename="config.yml",type="config"} 1
```

### Scaling kube-state-metrics

#### Resource recommendation

Resource usage for kube-state-metrics changes with the Kubernetes objects (Pods/Nodes/Deployments/Secrets etc.) size of the cluster.
To some extent, the Kubernetes objects in a cluster are in direct proportion to the node number of the cluster.

As a general rule, you should allocate:

* 250MiB memory
* 0.1 cores

Note that if CPU limits are set too low, kube-state-metrics' internal queues will not be able to be worked off quickly enough, resulting in increased memory consumption as the queue length grows. If you experience problems resulting from high memory allocation or CPU throttling, try increasing the CPU limits.

### Latency

In a 100 node cluster scaling test the latency numbers were as follows:

```text
"Perc50": 259615384 ns,
"Perc90": 475000000 ns,
"Perc99": 906666666 ns.
```

### A note on costing

By default, kube-state-metrics exposes several metrics for events across your cluster. If you have a large number of frequently-updating resources on your cluster, you may find that a lot of data is ingested into these metrics. This can incur high costs on some cloud providers. Please take a moment to [configure what metrics you'd like to expose](docs/developer/cli-arguments.md), as well as consult the documentation for your Kubernetes environment in order to avoid unexpectedly high costs.

### kube-state-metrics vs. metrics-server

The [metrics-server](https://github.com/kubernetes-incubator/metrics-server)
is a project that has been inspired by
[Heapster](https://github.com/kubernetes-retired/heapster) and is implemented
to serve the goals of core metrics pipelines in [Kubernetes monitoring
architecture](https://github.com/kubernetes/design-proposals-archive/blob/main/instrumentation/monitoring_architecture.md).
It is a cluster level component which periodically scrapes metrics from all
Kubernetes nodes served by Kubelet through Metrics API. The metrics are
aggregated, stored in memory and served in [Metrics API
format](https://git.k8s.io/metrics/pkg/apis/metrics/v1alpha1/types.go). The
metrics-server stores the latest values only and is not responsible for
forwarding metrics to third-party destinations.

kube-state-metrics is focused on generating completely new metrics from
Kubernetes' object state (e.g. metrics based on deployments, replica sets,
etc.). It holds an entire snapshot of Kubernetes state in memory and
continuously generates new metrics based off of it. And just like the
metrics-server it too is not responsible for exporting its metrics anywhere.

Having kube-state-metrics as a separate project also enables access to these
metrics from monitoring systems such as Prometheus.

### Horizontal sharding

In order to shard kube-state-metrics horizontally, some automated sharding capabilities have been implemented. It is configured with the following flags:

* `--shard` (zero indexed)
* `--total-shards`

Sharding is done by taking an md5 sum of the Kubernetes Object's UID and performing a modulo operation on it with the total number of shards. Each shard decides whether the object is handled by the respective instance of kube-state-metrics or not. Note that this means all instances of kube-state-metrics, even if sharded, will have the network traffic and the resource consumption for unmarshaling objects for all objects, not just the ones they are responsible for. To optimize this further, the Kubernetes API would need to support sharded list/watch capabilities. In the optimal case, memory consumption for each shard will be 1/n compared to an unsharded setup. Typically, kube-state-metrics needs to be memory and latency optimized in order for it to return its metrics rather quickly to Prometheus. One way to reduce the latency between kube-state-metrics and the kube-apiserver is to run KSM with the `--use-apiserver-cache` flag. In addition to reducing the latency, this option will also lead to a reduction in the load on etcd.

Sharding should be used carefully and additional monitoring should be set up in order to ensure that sharding is set up and functioning as expected (eg. instances for each shard out of the total shards are configured).

#### Automated sharding

Automatic sharding allows each shard to discover its nominal position when deployed in a StatefulSet which is useful for automatically configuring sharding. This is an experimental feature and may be broken or removed without notice.

To enable automated sharding, kube-state-metrics must be run by a `StatefulSet` and the pod name and namespace must be handed to the kube-state-metrics process via the `--pod` and `--pod-namespace` flags. Example manifests demonstrating the autosharding functionality can be found in [`/examples/autosharding`](./examples/autosharding).

This way of deploying shards is useful when you want to manage KSM shards through a single Kubernetes resource (a single `StatefulSet` in this case) instead of having one `Deployment` per shard. The advantage can be especially significant when deploying a high number of shards.

The downside of using an auto-sharded setup comes from the rollout strategy supported by `StatefulSet`s. When managed by a `StatefulSet`, pods are replaced one at a time with each pod first getting terminated and then recreated. Besides such rollouts being slower, they will also lead to short downtime for each shard. If a Prometheus scrape happens during a rollout, it can miss some of the metrics exported by kube-state-metrics.

### Daemonset sharding for pod metrics

For pod metrics, they can be sharded per node with the following flag:

* `--node=$(NODE_NAME)`

Each kube-state-metrics pod uses FieldSelector (spec.nodeName) to watch/list pod metrics only on the same node.

A daemonset kube-state-metrics example:

```yaml
apiVersion: apps/v1
kind: DaemonSet
spec:
  template:
    spec:
      containers:
      - image: registry.k8s.io/kube-state-metrics/kube-state-metrics:IMAGE_TAG
        name: kube-state-metrics
        args:
        - --resource=pods
        - --node=$(NODE_NAME)
        env:
        - name: NODE_NAME
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: spec.nodeName
```

To track metrics for unassigned pods, you need to add an additional deployment and set `--track-unscheduled-pods`, as shown in the following example:

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - image: registry.k8s.io/kube-state-metrics/kube-state-metrics:IMAGE_TAG
        name: kube-state-metrics
        args:
        - --resources=pods
        - --track-unscheduled-pods
```

Other metrics can be sharded via [Horizontal sharding](#horizontal-sharding).

### Setup

Install this project to your `$GOPATH` using `go get`:

```bash
go get k8s.io/kube-state-metrics/v2
```

#### Building the Docker container

Simply run the following command in this root folder, which will create a
self-contained, statically-linked binary and build a Docker image:

```bash
make container
```

### Usage

Simply build and run kube-state-metrics inside a Kubernetes pod which has a
service account token that has read-only access to the Kubernetes cluster.

#### For users of prometheus-operator/kube-prometheus stack

The ([`kube-prometheus`](https://github.com/prometheus-operator/kube-prometheus/)) stack installs kube-state-metrics as one of its [components](https://github.com/prometheus-operator/kube-prometheus#kube-prometheus); you do not need to install kube-state-metrics if you're using the kube-prometheus stack.

If you want to revise the default configuration for kube-prometheus, for example to enable non-default metrics, have a look at [Customizing Kube-Prometheus](https://github.com/prometheus-operator/kube-prometheus/blob/main/docs/customizing.md).

#### Kubernetes Deployment

To deploy this project, you can simply run `kubectl apply -f examples/standard` and a Kubernetes service and deployment will be created. (Note: Adjust the apiVersion of some resource if your kubernetes cluster's version is not 1.8+, check the yaml file for more information).

To have Prometheus discover kube-state-metrics instances it is advised to create a specific Prometheus scrape config for kube-state-metrics that picks up both metrics endpoints. Annotation based discovery is discouraged as only one of the endpoints would be able to be selected, plus kube-state-metrics in most cases has special authentication and authorization requirements as it essentially grants read access through the metrics endpoint to most information available to it.

**Note:** Google Kubernetes Engine (GKE) Users - GKE has strict role permissions that will prevent the kube-state-metrics roles and role bindings from being created. To work around this, you can give your GCP identity the cluster-admin role by running the following one-liner:

```bash
kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud info --format='value(config.account)')
```

Note that your GCP identity is case sensitive but `gcloud info` as of Google Cloud SDK 221.0.0 is not. This means that if your IAM member contains capital letters, the above one-liner may not work for you. If you have 403 forbidden responses after running the above command and `kubectl apply -f examples/standard`, check the IAM member associated with your account at <https://console.cloud.google.com/iam-admin/iam?project=PROJECT_ID>. If it contains capital letters, you may need to set the --user flag in the command above to the case-sensitive role listed at <https://console.cloud.google.com/iam-admin/iam?project=PROJECT_ID>.

After running the above, if you see `Clusterrolebinding "cluster-admin-binding" created`, then you are able to continue with the setup of this service.

#### Healthcheck Endpoints

The following healthcheck endpoints are available (`self` refers to the telemetry port, while `main` refers to the exposition port):

* `/healthz` (exposed on `main`): Returns a 200 status code if the application is running. We recommend to use this for the startup probe.
* `/livez` (exposed on `main`): Returns a 200 status code if the application is not affected by an outage of the Kubernetes API Server. We recommend to using this for the liveness probe.
* `/readyz` (exposed on `self`): Returns a 200 status code if the application is ready to accept requests and expose metrics. We recommend using this for the readiness probe.

Note that it is discouraged to use the telemetry metrics endpoint for any probe when proxying the exposition data.

#### Limited privileges environment

If you want to run kube-state-metrics in an environment where you don't have cluster-reader role, you can:

* create a serviceaccount

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-state-metrics
  namespace: your-namespace-where-kube-state-metrics-will-deployed
```

* give it `view` privileges on specific namespaces (using roleBinding) (*note: you can add this roleBinding to all the NS you want your serviceaccount to access*)

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kube-state-metrics
  namespace: project1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: view
subjects:
  - kind: ServiceAccount
    name: kube-state-metrics
    namespace: your-namespace-where-kube-state-metrics-will-deployed
```

* then specify a set of namespaces (using the `--namespaces` option) and a set of kubernetes objects (using the `--resources`) that your serviceaccount has access to in the `kube-state-metrics` deployment configuration

```yaml
spec:
  template:
    spec:
      containers:
      - name: kube-state-metrics
        args:
          - '--resources=pods'
          - '--namespaces=project1'
```

For the full list of arguments available, see the documentation in [docs/developer/cli-arguments.md](./docs/developer/cli-arguments.md)

#### Helm Chart

Starting from the kube-state-metrics chart `v2.13.3` (kube-state-metrics image `v1.9.8`), the official [Helm chart](https://artifacthub.io/packages/helm/prometheus-community/kube-state-metrics/) is maintained in [prometheus-community/helm-charts](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-state-metrics). Starting from kube-state-metrics chart `v3.0.0` only kube-state-metrics images of `v2.0.0 +` are supported.

#### Development

When developing, test a metric dump against your local Kubernetes cluster by running:

> Users can override the apiserver address in KUBE-CONFIG file with `--apiserver` command line.

```bash
go install
kube-state-metrics --port=8080 --telemetry-port=8081 --kubeconfig=<KUBE-CONFIG> --apiserver=<APISERVER>
```

Then curl the metrics endpoint

```bash
curl localhost:8080/metrics
```

To run the e2e tests locally see the documentation in [tests/README.md](./tests/README.md).

#### Developer Contributions

When developing, there are certain code patterns to follow to better your contributing experience and likelihood of e2e and other ci tests to pass. To learn more about them, see the documentation in [docs/developer/guide.md](./docs/developer/guide.md).

#### Community

This project is sponsored by [SIG Instrumentation](https://github.com/kubernetes/community/tree/master/sig-instrumentation).

There is also a channel for [#kube-state-metrics](https://kubernetes.slack.com/archives/CJJ529RUY) on Kubernetes' Slack.

You can also join the SIG Instrumentation [mailing list](https://groups.google.com/forum/#!forum/kubernetes-sig-instrumentation).
This will typically add invites for the following meetings to your calendar, in which topics around kube-state-metrics can be discussed.

* Regular SIG Meeting: [Thursdays at 9:30 PT (Pacific Time)](https://zoom.us/j/5342565819?pwd=RlVsK21NVnR1dmE3SWZQSXhveHZPdz09) (biweekly). [Convert to your timezone](http://www.thetimezoneconverter.com/?t=9:30&tz=PT%20%28Pacific%20Time%29).
* Regular Triage Meeting: [Thursdays at 9:30 PT (Pacific Time)](https://zoom.us/j/5342565819?pwd=RlVsK21NVnR1dmE3SWZQSXhveHZPdz09) (biweekly - alternating with regular meeting). [Convert to your timezone](http://www.thetimezoneconverter.com/?t=9:30&tz=PT%20%28Pacific%20Time%29).
