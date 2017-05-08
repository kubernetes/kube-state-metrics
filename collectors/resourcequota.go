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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	descResourceQuota = prometheus.NewDesc(
		"kube_resourcequota",
		"Information about resource quota.",
		[]string{
			"resourcequota",
			"namespace",
			"resource",
			"type",
		}, nil,
	)
)

type ResourceQuotaLister func() (v1.ResourceQuotaList, error)

func (l ResourceQuotaLister) List() (v1.ResourceQuotaList, error) {
	return l()
}

func RegisterResourceQuotaCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface) {
	client := kubeClient.CoreV1().RESTClient()
	rqlw := cache.NewListWatchFromClient(client, "resourcequotas", api.NamespaceAll, nil)
	rqinf := cache.NewSharedInformer(rqlw, &v1.ResourceQuota{}, resyncPeriod)

	resourceQuotaLister := ResourceQuotaLister(func() (quotas v1.ResourceQuotaList, err error) {
		for _, rq := range rqinf.GetStore().List() {
			quotas.Items = append(quotas.Items, *(rq.(*v1.ResourceQuota)))
		}
		return quotas, nil
	})

	registry.MustRegister(&resourceQuotaCollector{store: resourceQuotaLister})
	go rqinf.Run(context.Background().Done())
}

type resourceQuotaStore interface {
	List() (v1.ResourceQuotaList, error)
}

// resourceQuotaCollector collects metrics about all resource quotas in the cluster.
type resourceQuotaCollector struct {
	store resourceQuotaStore
}

// Describe implements the prometheus.Collector interface.
func (rqc *resourceQuotaCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descResourceQuota
}

// Collect implements the prometheus.Collector interface.
func (rqc *resourceQuotaCollector) Collect(ch chan<- prometheus.Metric) {
	resourceQuota, err := rqc.store.List()
	if err != nil {
		glog.Errorf("listing resource quotas failed: %s", err)
		return
	}

	for _, rq := range resourceQuota.Items {
		rqc.collectResourceQuota(ch, rq)
	}
}

func (rqc *resourceQuotaCollector) collectResourceQuota(ch chan<- prometheus.Metric, rq v1.ResourceQuota) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{rq.Name, rq.Namespace}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	for res, qty := range rq.Status.Hard {
		addGauge(descResourceQuota, float64(qty.MilliValue())/1000, string(res), "hard")
	}
	for res, qty := range rq.Status.Used {
		addGauge(descResourceQuota, float64(qty.MilliValue())/1000, string(res), "used")
	}

}
