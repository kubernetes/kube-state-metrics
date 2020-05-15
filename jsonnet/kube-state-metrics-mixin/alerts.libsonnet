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
              message: 'kube-state-metrics is experiencing errors at an elevated rate in list operations. This is likely causing it to not be able to expose metrics about Kubernetes objects correctly or at all.',
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
              message: 'kube-state-metrics is experiencing errors at an elevated rate in watch operations. This is likely causing it to not be able to expose metrics about Kubernetes objects correctly or at all.',
            },
          },
        ],
      },
    ],
  },
}
