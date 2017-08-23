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
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/tools/cache"
)

var (
	descIngressInfo = prometheus.NewDesc(
		"kube_ingress_info",
		"The info of ingress.",
		[]string{"namespace", "ingress"}, nil,
	)
	descIngressMetadataGeneration = prometheus.NewDesc(
		"kube_ingress_metadata_generation",
		"Sequence number representing a specific generation of the desired state.",
		[]string{"namespace", "ingress"}, nil,
	)
	descIngressLoadBalancer = prometheus.NewDesc(
		"kube_ingress_loadbalancer",
		"kube ingress loadbalancer.",
		[]string{"namespace", "ingress", "ip", "hostname"}, nil,
	)
)

type IngressLister func() (v1beta1.IngressList, error)

func (l IngressLister) List() (v1beta1.IngressList, error) {
	return l()
}

func RegisterIngressCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface, namespace string) {
	client := kubeClient.ExtensionsV1beta1().RESTClient()
	rslw := cache.NewListWatchFromClient(client, "ingresses", namespace, nil)
	rsinf := cache.NewSharedInformer(rslw, &v1beta1.Ingress{}, resyncPeriod)

	ingressLister := IngressLister(func() (ingresses v1beta1.IngressList, err error) {
		for _, c := range rsinf.GetStore().List() {
			ingresses.Items = append(ingresses.Items, *(c.(*v1beta1.Ingress)))
		}
		return ingresses, nil
	})

	registry.MustRegister(&ingressCollector{store: ingressLister})
	go rsinf.Run(context.Background().Done())
}

type ingressStore interface {
	List() (ingresses v1beta1.IngressList, err error)
}

// replicasetCollector collects metrics about all replicasets in the cluster.
type ingressCollector struct {
	store ingressStore
}

// Describe implements the prometheus.Collector interface.
func (dc *ingressCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descIngressInfo
	ch <- descIngressMetadataGeneration
	ch <- descIngressLoadBalancer
}

// Collect implements the prometheus.Collector interface.
func (dc *ingressCollector) Collect(ch chan<- prometheus.Metric) {
	dpls, err := dc.store.List()
	if err != nil {
		glog.Errorf("listing ingresses failed: %s", err)
		return
	}
	for _, d := range dpls.Items {
		dc.collectIngress(ch, d)
	}
}

func (dc *ingressCollector) collectIngress(ch chan<- prometheus.Metric, d v1beta1.Ingress) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{d.Namespace, d.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addGauge(descIngressMetadataGeneration, float64(d.ObjectMeta.Generation))
	addGauge(descIngressInfo, 1)
	if d.Status.LoadBalancer.Size() > 0 {
		for _, lb := range d.Status.LoadBalancer.Ingress {
			addGauge(descIngressLoadBalancer, 1, lb.IP, lb.Hostname)
		}
	}
}
