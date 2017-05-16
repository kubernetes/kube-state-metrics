# Grafana

This directory contains useful out-of-box dashboards.

# Constraints

Dashboards contain [cluster](../grafana/dashboards/cluster) and [single](../grafana/dashboards/single) dashboards.
* cluster: aggregated dashboards for a cluster.
* single: dashboards for a single resource(e.g. `node` or `pod`).

**Notice**
* cluster dashboards use a `cluster` template variable. It can be set at target level as a custom label or set as an `external_labels` label in case Prometheus federation is used.

# Dashboards

* cluster
  * [nodes](../grafana/dashboards/cluster/nodes.json)
    * nodes unschedulable: nodes whose node `Unschedulable` status is `true`.
    * nodes notready: nodes whose node `NodeReady` status is not `true`.
    * ready node number: number of nodes whose node `NodeReady` status is `true`.
* single

