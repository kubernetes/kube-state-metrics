# Developer Guide

This developer guide documentation is intended to assist all contributors in various code contributions.
Any contribution to improving this documentation will be appreciated.

## Table of Contents

* [Add New Kubernetes Resource Metric Collector](#add-new-kubernetes-resource-metric-collector)
* [Add New Metrics](#add-new-metrics)

### Add New Kubernetes Resource Metric Collector

The following steps are needed to introduce a new resource and its respective resource metrics.

* Reference your new resource(s) to the [docs/README.md](./../README.md#exposed-metrics).
* Reference your new resource(s) in the [docs/developer/cli-arguments.md](./cli-arguments.md#available-options) as part of the `--resources` flag.
* Create a new `<name-of-resource>.md` in the [docs](./../docs) directory to provide documentation on the resource(s) and metrics you implemented. Follow the formatting of all other resources.
* Add the resource(s) you are representing to the [jsonnet/kube-state-metrics/kube-state-metrics.libsonnet](./../../jsonnet/kube-state-metrics/kube-state-metrics.libsonnet) under the appropriate `apiGroup` using the `verbs`: `list` and `watch`.
* Run `make examples/standard`, this should re-generate [examples/standard/cluster-role.yaml](./../../examples/standard/cluster-role.yaml) with the resource(s) added to [jsonnet/kube-state-metrics/kube-state-metrics.libsonnet](./../../jsonnet/kube-state-metrics/kube-state-metrics.libsonnet).
* Reference and add build functions for the new resource(s) in [internal/store/builder.go](./../../internal/store/builder.go).
* Reference the new resource in [pkg/options/resource.go](./../../pkg/options/resource.go).
* Add a sample Kubernetes manifest to be used by tests in the [tests/manifests/](./../../tests/manifests) directory.
* Lastly, and most importantly, actually implement your new resource(s) and its test binary in [internal/store](./../../internal/store). Follow the formatting and structure of other resources.

### Add New Metrics

* Make metrics experimental first when introducing them, refer [#1910](https://github.com/kubernetes/kube-state-metrics/pull/1910) for more information.

| Metric stability level |                    |
|------------------------|--------------------|
| EXPERIMENTAL           | basemetrics.ALPHA  |
| STABLE                 | basemetrics.STABLE |
