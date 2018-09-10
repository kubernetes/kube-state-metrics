/*
Copyright 2018 The Kubernetes Authors All rights reserved.
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
	"k8s.io/api/core/v1"
	v1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsInformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/pkg/options"

	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	descCustomResourceDefinitionLabelsName          = "kube_customresourcedefinition_labels"
	descCustomResourceDefinitionLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descCustomResourceDefinitionLabelsDefaultLabels = []string{"customresourcedefinition"}

	descCustomResourceDefinitionCreated = prometheus.NewDesc(
		"kube_customresourcedefinition_created",
		"Unix creation timestamp.",
		descCustomResourceDefinitionLabelsDefaultLabels,
		nil,
	)
	descCustomResourceDefinitionLabels = prometheus.NewDesc(
		descCustomResourceDefinitionLabelsName,
		descCustomResourceDefinitionLabelsHelp,
		descCustomResourceDefinitionLabelsDefaultLabels,
		nil,
	)
	descCustomResourceDefinitionSpecGroupVersion = prometheus.NewDesc(
		"kube_customresourcedefinition_spec_groupversion",
		"Information about the customresourcedefinition group and version.",
		append(descNodeLabelsDefaultLabels, "group", "version"),
		nil,
	)
	descCustomResourceDefinitionSpecScope = prometheus.NewDesc(
		"kube_customresourcedefinition_spec_scope",
		"kubernetes customresourcedefinition spec scope.",
		append(descCustomResourceDefinitionLabelsDefaultLabels, "Scope"),
		nil,
	)
	descCustomResourceDefinitionStatusCondition = prometheus.NewDesc(
		"kube_customresourcedefinition_status_condition",
		"The condition of a customresourcedefinition.",
		append(descNodeLabelsDefaultLabels, "condition", "status"),
		nil,
	)
)

// CustomResourceDefinitionLister define CustomResourceDefinitionLister type
type CustomResourceDefinitionLister func() ([]v1beta1.CustomResourceDefinition, error)

// List return customresourcedefinition list
func (l CustomResourceDefinitionLister) List() ([]v1beta1.CustomResourceDefinition, error) {
	return l()
}

// RegisterCustomResourceDefinitionCollector registry namespace collector
func RegisterCustomResourceDefinitionCollector(registry prometheus.Registerer, informerFactories map[string][]interface{}, opts *options.Options) {
	crdsinfs := SharedInformerList{}
	gvr := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1beta1",
		Resource: "customresourcedefinitions",
	}
	for _, f := range informerFactories["crd"] {
		crdinformer, err := f.(apiextensionsInformers.SharedInformerFactory).ForResource(gvr)
		if err != nil {
			glog.Errorf("create customresourcedefinition GenericInformer failed: %s", err)
			continue
		}
		crdsinfs = append(crdsinfs, crdinformer.Informer().(cache.SharedInformer))
	}
	crdLister := CustomResourceDefinitionLister(func() (crds []v1beta1.CustomResourceDefinition, err error) {
		for _, crdinf := range crdsinfs {
			for _, crd := range crdinf.GetStore().List() {
				crds = append(crds, *(crd.(*v1beta1.CustomResourceDefinition)))
			}
		}
		return crds, nil
	})

	registry.MustRegister(&crdCollector{store: crdLister, opts: opts})
	crdsinfs.Run(context.Background().Done())
}

type crdStore interface {
	List() ([]v1beta1.CustomResourceDefinition, error)
}

// namespaceCollector collects metrics about all namespace in the cluster.
type crdCollector struct {
	store crdStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (crdc *crdCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descCustomResourceDefinitionCreated
	ch <- descCustomResourceDefinitionLabels
	ch <- descCustomResourceDefinitionSpecGroupVersion
	ch <- descCustomResourceDefinitionSpecScope
	ch <- descCustomResourceDefinitionStatusCondition
}

// Collect implements the prometheus.Collector interface.
func (crdc *crdCollector) Collect(ch chan<- prometheus.Metric) {
	crdls, err := crdc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "customresourcedefinition"}).Inc()
		glog.Errorf("listing customresourcedefinition failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "customresourcedefinition"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "customresourcedefinition"}).Observe(float64(len(crdls)))
	for _, crd := range crdls {
		crdc.collectCustomResourceDefinition(ch, crd)
	}

	glog.V(4).Infof("collected %d customresourcedefinitions", len(crdls))
}

func (crdc *crdCollector) collectCustomResourceDefinition(ch chan<- prometheus.Metric, crd v1beta1.CustomResourceDefinition) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{crd.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	if !crd.CreationTimestamp.IsZero() {
		addGauge(descCustomResourceDefinitionCreated, float64(crd.CreationTimestamp.Unix()))
	}

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(crd.Labels)
	addGauge(customresourcedefinitionLabelsDesc(labelKeys), 1, labelValues...)

	addGauge(descCustomResourceDefinitionSpecGroupVersion, 1, crd.Spec.Group, crd.Spec.Version)

	addGauge(descCustomResourceDefinitionSpecScope, boolFloat64(crd.Spec.Scope == v1beta1.ClusterScoped), string(v1beta1.ClusterScoped))
	addGauge(descCustomResourceDefinitionSpecScope, boolFloat64(crd.Spec.Scope == v1beta1.NamespaceScoped), string(v1beta1.NamespaceScoped))

	// Collect crd conditions and while default to false.
	for _, c := range crd.Status.Conditions {
		addConditionMetrics(ch, descCustomResourceDefinitionStatusCondition, v1.ConditionStatus(c.Status), crd.Name, string(c.Type))
	}

}

func customresourcedefinitionLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descCustomResourceDefinitionLabelsName,
		descCustomResourceDefinitionLabelsHelp,
		append(descCustomResourceDefinitionLabelsDefaultLabels, labelKeys...),
		nil,
	)
}
