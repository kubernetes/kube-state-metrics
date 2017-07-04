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
	descNamespace = prometheus.NewDesc(
		"kube_namespace_status_phase",
		"kubernetes namespace status phase.",
		[]string{
			"name",
			"create_time",
			"status",
		}, nil,
	)
)

// NamespaceLister ...
type NamespaceLister func() ([]v1.Namespace, error)

// List ...
func (l NamespaceLister) List() ([]v1.Namespace, error) {
	return l()
}

// RegisterNamespaceCollector registry namespace collector
func RegisterNamespaceCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface) {
	client := kubeClient.CoreV1().RESTClient()

	nslw := cache.NewListWatchFromClient(client, "namespaces", api.NamespaceAll, nil)
	nsinf := cache.NewSharedInformer(nslw, &v1.Namespace{}, resyncPeriod)

	namespaceLister := NamespaceLister(func() (namespaces []v1.Namespace, err error) {
		for _, ns := range nsinf.GetStore().List() {
			namespaces = append(namespaces, *(ns.(*v1.Namespace)))
		}
		return namespaces, nil
	})

	registry.MustRegister(&namespaceCollector{store: namespaceLister})
	go nsinf.Run(context.Background().Done())
}

type namespaceStore interface {
	List() ([]v1.Namespace, error)
}

// limitRangeCollector collects metrics about all limit ranges in the cluster.
type namespaceCollector struct {
	store namespaceStore
}

// Describe implements the prometheus.Collector interface.
func (nsc *namespaceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descNamespace
}

// Collect implements the prometheus.Collector interface.
func (nsc *namespaceCollector) Collect(ch chan<- prometheus.Metric) {
	nsls, err := nsc.store.List()
	if err != nil {
		glog.Errorf("listing limit ranges failed: %s", err)
		return
	}

	for _, rq := range nsls {
		nsc.collectNamespace(ch, rq)
	}
}

func (lrc *namespaceCollector) collectNamespace(ch chan<- prometheus.Metric, rq v1.Namespace) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{rq.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	status := string(rq.Status.Phase)
	addGauge(descNamespace, 1, rq.CreationTimestamp.String(), status)

}
