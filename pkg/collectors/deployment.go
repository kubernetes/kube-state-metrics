/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package collectors

import (
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	descDeploymentLabelsName          = "kube_deployment_labels"
	descDeploymentLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descDeploymentLabelsDefaultLabels = []string{"namespace", "deployment"}

	descDeploymentCreated = prometheus.NewDesc(
		"kube_deployment_created",
		"Unix creation timestamp",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentStatusReplicas = prometheus.NewDesc(
		"kube_deployment_status_replicas",
		"The number of replicas per deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)
	descDeploymentStatusReplicasAvailable = prometheus.NewDesc(
		"kube_deployment_status_replicas_available",
		"The number of available replicas per deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)
	descDeploymentStatusReplicasUnavailable = prometheus.NewDesc(
		"kube_deployment_status_replicas_unavailable",
		"The number of unavailable replicas per deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)
	descDeploymentStatusReplicasUpdated = prometheus.NewDesc(
		"kube_deployment_status_replicas_updated",
		"The number of updated replicas per deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentStatusObservedGeneration = prometheus.NewDesc(
		"kube_deployment_status_observed_generation",
		"The generation observed by the deployment controller.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentSpecReplicas = prometheus.NewDesc(
		"kube_deployment_spec_replicas",
		"Number of desired pods for a deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentSpecPaused = prometheus.NewDesc(
		"kube_deployment_spec_paused",
		"Whether the deployment is paused and will not be processed by the deployment controller.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentStrategyRollingUpdateMaxUnavailable = prometheus.NewDesc(
		"kube_deployment_spec_strategy_rollingupdate_max_unavailable",
		"Maximum number of unavailable replicas during a rolling update of a deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentStrategyRollingUpdateMaxSurge = prometheus.NewDesc(
		"kube_deployment_spec_strategy_rollingupdate_max_surge",
		"Maximum number of replicas that can be scheduled above the desired number of replicas during a rolling update of a deployment.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentMetadataGeneration = prometheus.NewDesc(
		"kube_deployment_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		descDeploymentLabelsDefaultLabels,
		nil,
	)

	descDeploymentLabels = prometheus.NewDesc(
		descDeploymentLabelsName,
		descDeploymentLabelsHelp,
		descDeploymentLabelsDefaultLabels, nil,
	)
)

type DeploymentLister func() ([]v1beta1.Deployment, error)

func (l DeploymentLister) List() ([]v1beta1.Deployment, error) {
	return l()
}

func RegisterDeploymentCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Extensions().V1beta1().Deployments().Informer().(cache.SharedInformer))
	}

	dplLister := DeploymentLister(func() (deployments []v1beta1.Deployment, err error) {
		for _, dinf := range infs {
			for _, c := range dinf.GetStore().List() {
				deployments = append(deployments, *(c.(*v1beta1.Deployment)))
			}
		}
		return deployments, nil
	})

	registry.MustRegister(&deploymentCollector{store: dplLister, opts: opts})
	infs.Run(context.Background().Done())
}

type deploymentStore interface {
	List() (deployments []v1beta1.Deployment, err error)
}

// deploymentCollector collects metrics about all deployments in the cluster.
type deploymentCollector struct {
	store deploymentStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (dc *deploymentCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descDeploymentCreated
	ch <- descDeploymentStatusReplicas
	ch <- descDeploymentStatusReplicasAvailable
	ch <- descDeploymentStatusReplicasUnavailable
	ch <- descDeploymentStatusReplicasUpdated
	ch <- descDeploymentStatusObservedGeneration
	ch <- descDeploymentSpecPaused
	ch <- descDeploymentStrategyRollingUpdateMaxUnavailable
	ch <- descDeploymentStrategyRollingUpdateMaxSurge
	ch <- descDeploymentSpecReplicas
	ch <- descDeploymentMetadataGeneration
	ch <- descDeploymentLabels
}

// Collect implements the prometheus.Collector interface.
func (dc *deploymentCollector) Collect(ch chan<- prometheus.Metric) {
	ds, err := dc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "deployment"}).Inc()
		glog.Errorf("listing deployments failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "deployment"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "deployment"}).Observe(float64(len(ds)))
	for _, d := range ds {
		dc.collectDeployment(ch, d)
	}

	glog.V(4).Infof("collected %d deployments", len(ds))
}

func deploymentLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descDeploymentLabelsName,
		descDeploymentLabelsHelp,
		append(descDeploymentLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func (dc *deploymentCollector) collectDeployment(ch chan<- prometheus.Metric, d v1beta1.Deployment) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{d.Namespace, d.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(d.Labels)
	addGauge(deploymentLabelsDesc(labelKeys), 1, labelValues...)
	if !d.CreationTimestamp.IsZero() {
		addGauge(descDeploymentCreated, float64(d.CreationTimestamp.Unix()))
	}
	addGauge(descDeploymentStatusReplicas, float64(d.Status.Replicas))
	addGauge(descDeploymentStatusReplicasAvailable, float64(d.Status.AvailableReplicas))
	addGauge(descDeploymentStatusReplicasUnavailable, float64(d.Status.UnavailableReplicas))
	addGauge(descDeploymentStatusReplicasUpdated, float64(d.Status.UpdatedReplicas))
	addGauge(descDeploymentStatusObservedGeneration, float64(d.Status.ObservedGeneration))
	addGauge(descDeploymentSpecPaused, boolFloat64(d.Spec.Paused))
	addGauge(descDeploymentSpecReplicas, float64(*d.Spec.Replicas))
	addGauge(descDeploymentMetadataGeneration, float64(d.ObjectMeta.Generation))

	if d.Spec.Strategy.RollingUpdate == nil {
		return
	}

	maxUnavailable, err := intstr.GetValueFromIntOrPercent(d.Spec.Strategy.RollingUpdate.MaxUnavailable, int(*d.Spec.Replicas), true)
	if err != nil {
		glog.Errorf("Error converting RollingUpdate MaxUnavailable to int: %s", err)
	} else {
		addGauge(descDeploymentStrategyRollingUpdateMaxUnavailable, float64(maxUnavailable))
	}

	maxSurge, err := intstr.GetValueFromIntOrPercent(d.Spec.Strategy.RollingUpdate.MaxSurge, int(*d.Spec.Replicas), true)
	if err != nil {
		glog.Errorf("Error converting RollingUpdate MaxSurge to int: %s", err)
	} else {
		addGauge(descDeploymentStrategyRollingUpdateMaxSurge, float64(maxSurge))
	}

}
