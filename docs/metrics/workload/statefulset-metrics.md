# Stateful Set Metrics

| Metric name                                             | Metric type | Description                                                                                                                             | Labels/tags                                                                                                                                                                                                         | Status       |
| ------------------------------------------------------- | ----------- | --------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------ |
| kube_statefulset_annotations                            | Gauge       | Kubernetes annotations converted to Prometheus labels controlled via [--metric-annotations-allowlist](../../developer/cli-arguments.md) | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt; <br> `annotation_STATEFULSET_ANNOTATION`=&lt;STATEFULSET_ANNOTATION&gt;                                                       | EXPERIMENTAL |
| kube_statefulset_status_replicas                        | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;                                                                                                                               | STABLE       |
| kube_statefulset_status_replicas_current                | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;                                                                                                                               | STABLE       |
| kube_statefulset_status_replicas_ready                  | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;                                                                                                                               | STABLE       |
| kube_statefulset_status_replicas_available              | Gauge       | | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;                                                                                                                               | STABLE       |
| kube_statefulset_status_replicas_updated                | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;                                                                                                                               | STABLE       |
| kube_statefulset_status_observed_generation             | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;                                                                                                                               | STABLE       |
| kube_statefulset_replicas                               | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;                                                                                                                               | STABLE       |
| kube_statefulset_ordinals_start                         | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;                                                                                                                               | STABLE       |
| kube_statefulset_metadata_generation                    | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;                                                                                                                               | STABLE       |
| kube_statefulset_persistentvolumeclaim_retention_policy | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt; <br> `when_deleted`=&lt;statefulset-when-deleted-pvc-policy&gt; <br> `when_scaled`=&lt;statefulset-when-scaled-pvc-policy&gt; | EXPERIMENTAL |
| kube_statefulset_created                                | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;                                                                                                                               | STABLE       |
| kube_statefulset_labels                                 | Gauge       | Kubernetes labels converted to Prometheus labels controlled via [--metric-labels-allowlist](../../developer/cli-arguments.md)           | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt; <br> `label_STATEFULSET_LABEL`=&lt;STATEFULSET_LABEL&gt;                                                                      | STABLE       |
| kube_statefulset_status_current_revision                | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt; <br> `revision`=&lt;statefulset-current-revision&gt;                                                                          | STABLE       |
| kube_statefulset_status_update_revision                 | Gauge       |                                                                                                                                         | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt; <br> `revision`=&lt;statefulset-update-revision&gt;                                                                           | STABLE       |
| kube_statefulset_deletion_timestamp                     | Gauge       | Unix deletion timestamp                                                                                                                 | `statefulset`=&lt;statefulset-name&gt; <br> `namespace`=&lt;statefulset-namespace&gt;                                                                                                                               | EXPERIMENTAL |

## Common PromQL Queries

### StatefulSet Health Monitoring

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


### Troubleshooting Queries

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

## Major Alerting Rules

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
