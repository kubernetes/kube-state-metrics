{
  _config+:: {
    // Select the metrics coming from the kube state metrics.
    kubeStateMetricsSelector: 'job="kube-state-metrics"',
  },
}
