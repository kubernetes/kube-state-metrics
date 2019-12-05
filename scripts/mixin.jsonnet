((import 'kube-state-metrics-mixin/mixin.libsonnet') {
   _config+:: {
     // Selectors are inserted between {} in Prometheus queries.
     kubeStateMetricsSelector: 'job="kube-state-metrics"',
   },
 }).prometheusAlerts
