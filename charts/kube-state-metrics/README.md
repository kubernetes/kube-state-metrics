# kube-state-metrics Helm Chart

Installs the [kube-state-metrics agent](https://github.com/kubernetes/kube-state-metrics).

## Get Repo Info

```console
helm repo add kube-state-metrics https://kubernetes.github.io/kube-state-metrics
helm repo update
```

_See [helm repo](https://helm.sh/docs/helm/helm_repo/) for command documentation._

## Install Chart

```console
# Helm 3
$ helm install [RELEASE_NAME] kube-state-metrics/kube-state-metrics [flags]
# Helm 2

$ helm install --name [RELEASE_NAME] kube-state-metrics/kube-state-metrics [flags]
```

_See [configuration](#configuration) below._

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._

## Uninstall Chart

```console
# Helm 3
$ helm uninstall [RELEASE_NAME]

# Helm 2
# helm delete --purge [RELEASE_NAME]
```

This removes all the Kubernetes components associated with the chart and deletes the release.

_See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation._

## Upgrading Chart

```console
# Helm 3 or 2
$ helm upgrade [RELEASE_NAME] kube-state-metrics/kube-state-metrics [flags]
```

_See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade/) for command documentation._

### From stable/kube-state-metrics

You can upgrade in-place:

1. [get repo info](#get-repo-info)
1. [upgrade](#upgrading-chart) your existing release name using the new chart repo

## Configuration

See [Customizing the Chart Before Installing](https://helm.sh/docs/intro/using_helm/#customizing-the-chart-before-installing). To see all configurable options with detailed comments:

```console
helm show values kube-state-metrics/kube-state-metrics
```

You may also `helm show values` on this chart's [dependencies](#dependencies) for additional options.
