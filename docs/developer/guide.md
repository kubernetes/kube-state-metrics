# Developer Guide 

This developer guide documentation is intended to assist all contributors in various code contributions.
Any contribution to improving this documentation will be appreciated.

## Table of Contents

- [Add New Kubernetes Resource Metric Collector](#add-new-kubernetes-resource-metric-collector)

### Add New Kubernetes Resource Metric Collector

The following steps are needed to introduce a new resource and its respective resource metrics.

- Reference your new resource(s) to the [docs/README.md](https://github.com/kubernetes/kube-state-metrics/blob/master/docs/README.md#exposed-metrics).
- Reference your new resource(s) in the [docs/cli-arguments.md](https://github.com/kubernetes/kube-state-metrics/blob/master/docs/cli-arguments.md#available-options) as part of the `--resources` flag.
- Create a new `<name-of-resource>.md` in the [docs](https://github.com/kubernetes/kube-state-metrics/tree/master/docs) directory to provide documentation on the resource(s) and metrics you implemented. Follow the formatting of all other resources.
- Add the resource(s) you are representing to the [examples/standard/cluster-role.yaml](https://github.com/kubernetes/kube-state-metrics/blob/master/examples/standard/cluster-role.yaml) under the appropriate `apiGroup` using the `verbs`: `list` and `watch`.
- Reference and add build functions for the new resource(s) in [internal/store/builder.go](https://github.com/kubernetes/kube-state-metrics/blob/master/internal/store/builder.go).
- Reference the new resource in [pkg/options/resource.go](https://github.com/kubernetes/kube-state-metrics/blob/master/pkg/options/resource.go).
- Add a sample Kubernetes manifest to be used by tests in the [tests/manifests/](https://github.com/kubernetes/kube-state-metrics/tree/master/tests/manifests) directory.
- Lastly, and most importantly, actually implement your new resource(s) and its test binary in [internal/store](https://github.com/kubernetes/kube-state-metrics/tree/master/internal/store). Follow the formatting and structure of other resources.
