# Documentation

This documentation is intended to be a complete reflection of the current state of the exposed metrics of kube-state-metrics.

Any contribution to improving this documentation or adding sample usages will be appreciated.

Stages about metrics are grouped into three categoriges：

| Stage | Description |
| ----------- | ----------- |
| EXPERIMENTAL | Metrics which are normally corresponds to Kubernetes API object alpha status or spec fields and can be changed at any time. |
| STABLE       | Metrics which should be very few backwards-incompatible changes outside of major version updates. |
| DEPRECATED   | Metrics which will be removed once the deprecation timeline is met. |
    
Per group of metrics there is one file for each metrics. See each file for specific documentation about the exposed metrics:

* [CronJob Metrics](cronjob-metrics.md)
* [DaemonSet Metrics](daemonset-metrics.md)
* [Deployment Metrics](deployment-metrics.md)
* [Job Metrics](job-metrics.md)
* [LimitRange Metrics](limitrange-metrics.md)
* [Node Metrics](node-metrics.md)
* [PersistentVolume Metrics](persistentvolume-metrics.md)
* [PersistentVolumeClaim Metrics](persistentvolumeclaim-metrics.md)
* [Pod Metrics](pod-metrics.md)
* [ReplicaSet Metrics](replicaset-metrics.md)
* [ReplicationController Metrics](replicationcontroller-metrics.md)
* [ResourceQuota Metrics](resourcequota-metrics.md)
* [Service Metrics](service-metrics.md)
* [StatefulSet Metrics](statefulset-metrics.md)
* [Namespace Metrics](namespace-metrics.md)
* [Horizontal Pod Autoscaler Metrics](horizontalpodautoscaler-metrics.md)
* [Endpoint Metrics](endpoint-metrics.md)
* [Secret Metrics](secret-metrics.md)
* [ConfigMap Metrics](configmap-metrics.md)