{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'kube-state-metrics',
        rules: [
          {
            alert: 'KubeStateMetricsListErrors',
            expr: |||
              (sum(rate(kube_state_metrics_list_total{%(kubeStateMetricsSelector)s,result="error"}[5m]))
                /
              sum(rate(kube_state_metrics_list_total{%(kubeStateMetricsSelector)s}[5m])))
              > 0.01
            ||| % $._config,
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'kube-state-metrics is experiencing errors in list operations.',
              description: 'kube-state-metrics is experiencing errors at an elevated rate in list operations. This is likely causing it to not be able to expose metrics about Kubernetes objects correctly or at all.',
            },
          },
          {
            alert: 'KubeStateMetricsWatchErrors',
            expr: |||
              (sum(rate(kube_state_metrics_watch_total{%(kubeStateMetricsSelector)s,result="error"}[5m]))
                /
              sum(rate(kube_state_metrics_watch_total{%(kubeStateMetricsSelector)s}[5m])))
              > 0.01
            ||| % $._config,
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'kube-state-metrics is experiencing errors in watch operations.',
              description: 'kube-state-metrics is experiencing errors at an elevated rate in watch operations. This is likely causing it to not be able to expose metrics about Kubernetes objects correctly or at all.',
            },
          },
          {
            alert: 'KubeStateMetricsShardingMismatch',
            //
            expr: |||
              stdvar (kube_state_metrics_total_shards{%(kubeStateMetricsSelector)s}) != 0
            ||| % $._config,
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'kube-state-metrics sharding is misconfigured.',
              description: 'kube-state-metrics pods are running with different --total-shards configuration, some Kubernetes objects may be exposed multiple times or not exposed at all.',
            },
          },
          {
            alert: 'KubeStateMetricsShardsMissing',
            // Each shard ordinal is assigned a binary position (2^ordinal) and we compute a sum of those.
            // This sum is compared to the expected number (2^total_shards - 1).
            // Result of zero all shards are being scraped, anything else indicates an issue.
            // A handy side effect of this computation is the result indicates what ordinals are missing.
            // Eg. a result of "5" decimal, which translates to binary "101", means shards #0 and #2 are not available.
            expr: |||
              2^max(kube_state_metrics_total_shards{%(kubeStateMetricsSelector)s}) - 1
                -
              sum( 2 ^ max by (shard_ordinal) (kube_state_metrics_shard_ordinal{%(kubeStateMetricsSelector)s}) )
              != 0
            ||| % $._config,
            'for': '15m',
            labels: {
              severity: 'critical',
            },
            annotations: {
              summary: 'kube-state-metrics shards are missing.',
              description: 'kube-state-metrics shards are missing, some Kubernetes objects are not being exposed.',
            },
          },
        ],
      },
    ],
  },
}
