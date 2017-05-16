# Grafana

This directory contains useful out-of-box dashboards.

# Constraints

Dashboards contain [summary](../grafana/dashboards/summary) and [single](../grafana/dashboards/single) dashboards.
* summary: Aggregated dashboards for a cluster(`alias`).
* single: single dashboards for a single resource(e.g. `node` or `pod`).

summary dashboards use `alias` as a template variable. `alias` can be set as a cluster name
thus we can make a summary for all resources about a cluster.

# Dashboards

* summary
  * [nodes](../grafana/dashboards/summary/nodes.json)
    * nodes unschedulable: nodes whose node `Unschedulable` status is `true`.
    * nodes notready: nodes whose node `NodeReady` status is not `true`.
    * ready node number: number of nodes whose node `NodeReady` status is `true`.
* single

# Caution

After importing all necessary dashboards, users should adapt data source to the right one.
