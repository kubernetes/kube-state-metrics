/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	descServiceLabelsName          = "kube_service_labels"
	descServiceLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descServiceLabelsDefaultLabels = []string{"namespace", "service"}

	descServiceInfo = prometheus.NewDesc(
		"kube_service_info",
		"Information about service.",
		append(descServiceLabelsDefaultLabels, "cluster_ip"),
		nil,
	)

	descServiceCreated = prometheus.NewDesc(
		"kube_service_created",
		"Unix creation timestamp",
		descServiceLabelsDefaultLabels,
		nil,
	)

	descServiceSpecType = prometheus.NewDesc(
		"kube_service_spec_type",
		"Type about service.",
		append(descServiceLabelsDefaultLabels, "type"),
		nil,
	)

	descServiceLabels = prometheus.NewDesc(
		descServiceLabelsName,
		descServiceLabelsHelp,
		descServiceLabelsDefaultLabels,
		nil,
	)
)

type ServiceLister func() ([]v1.Service, error)

func (l ServiceLister) List() ([]v1.Service, error) {
	return l()
}

func RegisterServiceCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Core().V1().Services().Informer().(cache.SharedInformer))
	}

	serviceLister := ServiceLister(func() (services []v1.Service, err error) {
		for _, sinf := range infs {
			for _, m := range sinf.GetStore().List() {
				services = append(services, *m.(*v1.Service))
			}
		}
		return services, nil
	})

	registry.MustRegister(&serviceCollector{store: serviceLister, opts: opts})
	infs.Run(context.Background().Done())
}

type serviceStore interface {
	List() (services []v1.Service, err error)
}

// serviceCollector collects metrics about all services in the cluster.
type serviceCollector struct {
	store serviceStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (pc *serviceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descServiceInfo
	ch <- descServiceLabels
	ch <- descServiceCreated
	ch <- descServiceSpecType
}

// Collect implements the prometheus.Collector interface.
func (sc *serviceCollector) Collect(ch chan<- prometheus.Metric) {
	services, err := sc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "service"}).Inc()
		glog.Errorf("listing services failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "service"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "service"}).Observe(float64(len(services)))
	for _, s := range services {
		sc.collectService(ch, s)
	}
	glog.V(4).Infof("collected %d services", len(services))
}

func serviceLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descServiceLabelsName,
		descServiceLabelsHelp,
		append(descServiceLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func (sc *serviceCollector) collectService(ch chan<- prometheus.Metric, s v1.Service) {
	addConstMetric := func(desc *prometheus.Desc, t prometheus.ValueType, v float64, lv ...string) {
		lv = append([]string{s.Namespace, s.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, t, v, lv...)
	}
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addGauge(descServiceSpecType, 1, string(s.Spec.Type))

	addGauge(descServiceInfo, 1, s.Spec.ClusterIP)
	if !s.CreationTimestamp.IsZero() {
		addGauge(descServiceCreated, float64(s.CreationTimestamp.Unix()))
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(s.Labels)
	addGauge(serviceLabelsDesc(labelKeys), 1, labelValues...)
}
