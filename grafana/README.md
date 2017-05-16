# Grafana

This directory contains useful out-of-box dashboards.

# Constraints

Dashboards contain [summary](../grafana/dashboards/summary) and [single](../grafana/dashboards/single) dashboards.
* summary: Aggregated dashboards for a cluster.
* single: dashboards for a single resource(e.g. `node` or `pod`).

**Notice**
* summary dashboards use `alias` as the `cluster` template variable.

# Dashboards

* summary
  * [nodes](../grafana/dashboards/summary/nodes.json)
    * nodes unschedulable: nodes whose node `Unschedulable` status is `true`.
    * nodes notready: nodes whose node `NodeReady` status is not `true`.
    * ready node number: number of nodes whose node `NodeReady` status is `true`.
* single

