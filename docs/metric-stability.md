# Metric Stability and Lifecycle

This document describes what users can rely on for each metric stability level, and how a metric moves from introduction to removal.

## Stability levels

| Level        | In code              | Guarantee                                                                  |
| ------------ | -------------------- | -------------------------------------------------------------------------- |
| EXPERIMENTAL | `basemetrics.ALPHA`  | None. The metric can change or be removed in any release.                  |
| STABLE       | `basemetrics.STABLE` | Covered by the compatibility guarantees below.                             |
| DEPRECATED   | `deprecatedVersion`  | Keeps its previous guarantees until it is removed, see [Deprecation](#deprecation). |

A deprecated metric keeps its stability level. The docs show it as for example `STABLE (Deprecated from 2.14.0)` and the help text is prefixed with `(Deprecated since v2.14.0)`.

### Mapping to Kubernetes

kube-state-metrics follows the [Kubernetes metrics stability framework](https://kubernetes.io/docs/concepts/cluster-administration/system-metrics/#metric-lifecycle), but the levels do not line up one to one. EXPERIMENTAL is equivalent to Kubernetes ALPHA. STABLE allows new labels to be added, so its guarantees are closest to Kubernetes BETA, even though it uses `basemetrics.STABLE` in code.

## Compatibility guarantees for STABLE metrics

The following changes to a STABLE metric are breaking:

* Renaming or removing the metric.
* Changing the metric type.
* Removing or renaming a label.
* Changing the meaning or format of a label value or of the metric value, for example a unit change.
* Changing which series are exposed for the same object, for example dropping zero valued series.

The following changes are not breaking:

* Adding a new label.
* Adding a new value to a label with an open set of values, for example a new reason.
* Changing the help text.
* Bug fixes where the previous output did not match the documented behavior.

A breaking change to a STABLE metric always goes through [deprecation](#deprecation) first. This also applies to major releases. A major release is the point where deprecated metrics can be removed in bulk, not a way to skip the deprecation window.

Following a newer naming convention is not on its own a reason to break a STABLE metric. Users build dashboards and alerts on these names, and a rename costs every one of them a migration.

## Lifecycle

### Introduction

* New metrics start as EXPERIMENTAL.
* The pull request explains the use case and the expected cardinality.
* Metrics that are expensive or have high cardinality are opt-in via `--metric-opt-in-list`.
* The metric is documented in [docs/metrics](metrics) with its labels and stability level.

### Promotion

An EXPERIMENTAL metric can be promoted to STABLE when:

* The Kubernetes API fields it reads from are GA. Metrics based on alpha or beta fields stay EXPERIMENTAL.
* It was part of two consecutive minor releases without changes to its name, type or labels.

Promotion is its own pull request that changes the level to `basemetrics.STABLE`, updates the docs and adds a `[CHANGE]` entry to the changelog.

### Deprecation

A STABLE metric, or a label on a STABLE metric, that is deprecated in release `Major.Minor` can be removed no earlier than release `Major.Minor+2`. Releases are counted in order across major versions, so a metric deprecated in `v2.20` can be removed in `v3.0` if `v2.21` was released in between, and in `v3.1` otherwise. To deprecate a metric:

* Set `deprecatedVersion` on the metric generator.
* Update the stability column in the docs.
* If there is a replacement, add it in the same release so old and new are exposed side by side for the whole window.
* Add a `[CHANGE]` entry to the changelog that names the replacement and the earliest removal release.

EXPERIMENTAL metrics do not need a deprecation window, but a changelog entry is still required when they change or are removed.

### Removal

A deprecated metric is removed once its window has passed. The removal is a separate pull request with a `[CHANGE]` entry in the changelog.
