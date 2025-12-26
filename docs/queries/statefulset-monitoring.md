# StatefulSet Monitoring Guide

This guide provides monitoring examples for StatefulSets using kube-state-metrics and Prometheus.

## Query Cookbook

### Health & Availability Queries

**Check StatefulSet rollout status:**

```promql
# Percentage of updated replicas
(kube_statefulset_status_replicas_updated / kube_statefulset_replicas) * 100
```

**Monitor unavailable replicas:**

```promql
# Number of unavailable replicas
kube_statefulset_replicas - kube_statefulset_status_replicas_available
```

### Diagnostic & Troubleshooting Queries

**Find StatefulSets with outdated replicas:**

```promql
# StatefulSets with replicas not yet updated
kube_statefulset_status_replicas_current != kube_statefulset_status_replicas_updated
```

**StatefulSets stuck during rollout:**

```promql
# StatefulSets where observed generation is behind metadata generation
kube_statefulset_status_observed_generation < kube_statefulset_metadata_generation
```

**StatefulSets with scaling issues:**

```promql
# StatefulSets where current replicas don't match desired
kube_statefulset_status_replicas_current != kube_statefulset_replicas
```

## Production Alert Rules

### Critical Alerts

**StatefulSet is completely down:**

```yaml
- alert: StatefulSetDown
  expr: kube_statefulset_status_replicas_available == 0 and kube_statefulset_replicas > 0
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "StatefulSet {{ $labels.statefulset }} is completely down"
    description: "StatefulSet {{ $labels.statefulset }} in namespace {{ $labels.namespace }} has no available replicas despite having {{ $labels.replicas }} desired replicas."
```

**StatefulSet rollout stuck:**

```yaml
- alert: StatefulSetRolloutStuck
  expr: kube_statefulset_status_observed_generation < kube_statefulset_metadata_generation
  for: 15m
  labels:
    severity: critical
  annotations:
    summary: "StatefulSet {{ $labels.statefulset }} rollout is stuck"
    description: "StatefulSet {{ $labels.statefulset }} in namespace {{ $labels.namespace }} has been stuck rolling out for more than 15 minutes."
```

### Warning Alerts

**StatefulSet has unavailable replicas:**

```yaml
- alert: StatefulSetReplicasUnavailable
  expr: (kube_statefulset_replicas - kube_statefulset_status_replicas_available) > 0
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "StatefulSet {{ $labels.statefulset }} has unavailable replicas"
    description: "StatefulSet {{ $labels.statefulset }} in namespace {{ $labels.namespace }} has {{ $value }} unavailable replicas."
```

**StatefulSet replica count mismatch:**

```yaml
- alert: StatefulSetReplicasMismatch
  expr: kube_statefulset_status_replicas_current != kube_statefulset_replicas
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "StatefulSet {{ $labels.statefulset }} replica count mismatch"
    description: "StatefulSet {{ $labels.statefulset }} in namespace {{ $labels.namespace }} has {{ $labels.status_replicas_current }} current replicas but {{ $labels.replicas }} are desired."
```
