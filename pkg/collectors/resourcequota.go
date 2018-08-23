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
	"k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	descResourceQuotaLabelsDefaultLabels = []string{"resourcequota", "namespace"}

	descResourceQuotaCreated = prometheus.NewDesc(
		"kube_resourcequota_created",
		"Unix creation timestamp",
		descResourceQuotaLabelsDefaultLabels,
		nil,
	)
	descResourceQuota = prometheus.NewDesc(
		"kube_resourcequota",
		"Information about resource quota.",
		append(descResourceQuotaLabelsDefaultLabels,
			"resource",
			"type",
		), nil,
	)
)

type ResourceQuotaLister func() (v1.ResourceQuotaList, error)

func (l ResourceQuotaLister) List() (v1.ResourceQuotaList, error) {
	return l()
}

func RegisterResourceQuotaCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Core().V1().ResourceQuotas().Informer().(cache.SharedInformer))
	}

	resourceQuotaLister := ResourceQuotaLister(func() (quotas v1.ResourceQuotaList, err error) {
		for _, rqinf := range infs {
			for _, rq := range rqinf.GetStore().List() {
				quotas.Items = append(quotas.Items, *(rq.(*v1.ResourceQuota)))
			}
		}
		return quotas, nil
	})

	registry.MustRegister(&resourceQuotaCollector{store: resourceQuotaLister, opts: opts})
	infs.Run(context.Background().Done())
}

type resourceQuotaStore interface {
	List() (v1.ResourceQuotaList, error)
}

// resourceQuotaCollector collects metrics about all resource quotas in the cluster.
type resourceQuotaCollector struct {
	store resourceQuotaStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (rqc *resourceQuotaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descResourceQuotaCreated
	ch <- descResourceQuota
}

// Collect implements the prometheus.Collector interface.
func (rqc *resourceQuotaCollector) Collect(ch chan<- prometheus.Metric) {
	resourceQuota, err := rqc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "resourcequota"}).Inc()
		glog.Errorf("listing resource quotas failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "resourcequota"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "resourcequota"}).Observe(float64(len(resourceQuota.Items)))
	for _, rq := range resourceQuota.Items {
		rqc.collectResourceQuota(ch, rq)
	}

	glog.V(4).Infof("collected %d resourcequotas", len(resourceQuota.Items))
}

func (rqc *resourceQuotaCollector) collectResourceQuota(ch chan<- prometheus.Metric, rq v1.ResourceQuota) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{rq.Name, rq.Namespace}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	if !rq.CreationTimestamp.IsZero() {
		addGauge(descResourceQuotaCreated, float64(rq.CreationTimestamp.Unix()))
	}
	for res, qty := range rq.Status.Hard {
		addGauge(descResourceQuota, float64(qty.MilliValue())/1000, string(res), "hard")
	}
	for res, qty := range rq.Status.Used {
		addGauge(descResourceQuota, float64(qty.MilliValue())/1000, string(res), "used")
	}

}
