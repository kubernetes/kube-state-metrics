# Kube-State-Metrics - Validate stable metrics Proposal

---

Author: CatherineF-dev@

Date: 1. Jan 2024

---

## Glossary

* STABLE metrics: it's same to kubernetes/kubernetes BETA metrics. It does
 not allow breaking changes which break metrics queries. For example, changing
 label name is not allowed.

* Experimental metrics: it's same to kubernetes/kubernetes ALPHA metrics. It
allows breaking changes. For example, it allows changing label name.

## Problem Statement

Broken stable metrics bring overhead for down-stream users to migrate metrics
queries.

## Goal

The goal of this proposal is guaranteeing these for stable metrics:

1. metric name not changed

2. metric type not changed

3. old metric labels is a subset of new metric labels

## Status Quo

Kubernetes/kubernetes has implemented stable metrics framework. It can not be
used in KSM repo directly, because kubernetes/kubernetes metrics are defined
using prometheus libraries while KSM metrics are defined using custom functions.

## Proposal - validate stable metrics using static analysis

1. Add a new funciton NewFamilyGeneratorWithStabilityV2 which has labels in its
 parameter. It's easier for static analysis in step 2.

2. Adapt stable metrics framework <https://github.com/kubernetes/kubernetes/tree/master/test/instrumentation>
into kube-state-metrics repo.

2.1 Find stable metrics definitions. It finds all function calls with name NewFamilyGeneratorWithStabilityV2 where fourth paraemeter (StabilityLevel) is  stable

2.2 Extract metric name, labels, help, stability level from 2.1

```
func createPodInitContainerStatusRestartsTotalFamilyGenerator() generator.FamilyGenerator {
 return *generator.NewFamilyGeneratorWithStabilityV2(
  "kube_pod_init_container_status_restarts_total",
  "The number of restarts for the init container.",
  metric.Counter, basemetrics.STABLE,
  "",
  []string{"container"}, # labels
    ...
}
```

2.3 Export all stable metrics, with format

```
- name: kube_pod_init_container_status_restarts_total
  help: The number of restarts for the init container.
  type: Counter
  stabilityLevel: STABLE
  labels:
  - container
```

2.4 Compare output in 2.3 with expected results tests/stablemetrics/testdata/test-stable-metrics-list.yaml

## Alternatives

### Validate exposed metrics in runtime

Generated metrics are not complete. Some metrics are exposed when certain conditions
are met.

## FAQ / Follow up improvements
