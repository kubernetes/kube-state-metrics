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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	descComponentStatus = prometheus.NewDesc(
		"kube_component_status",
		"kube component status.",
		[]string{"name", "status"}, nil,
	)
)

type ComponentStatusLister func() ([]v1.ComponentStatus, error)

func (l ComponentStatusLister) List() ([]v1.ComponentStatus, error) {
	return l()
}

func RegisterComponentStatusCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface) {
	client := kubeClient.CoreV1().RESTClient()
	slw := cache.NewListWatchFromClient(client, "componentstatuses", api.NamespaceAll, nil)
	sinf := cache.NewSharedInformer(slw, &v1.ComponentStatus{}, resyncPeriod)

	componentStatusLister := ComponentStatusLister(func() (componentStatus []v1.ComponentStatus, err error) {
		for _, m := range sinf.GetStore().List() {
			componentStatus = append(componentStatus, *m.(*v1.ComponentStatus))
		}
		return componentStatus, nil
	})

	registry.MustRegister(&componentStatusCollector{store: componentStatusLister})
	go sinf.Run(context.Background().Done())
}

type componentStatusStore interface {
	List() (componentStatus []v1.ComponentStatus, err error)
}

// componentStatusCollector collects metrics about all component in the cluster.
type componentStatusCollector struct {
	store componentStatusStore
}

// Describe implements the prometheus.Collector interface.
func (csc *componentStatusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descComponentStatus
}

// Collect implements the prometheus.Collector interface.
func (csc *componentStatusCollector) Collect(ch chan<- prometheus.Metric) {
	componentStatus, err := csc.store.List()
	if err != nil {
		glog.Errorf("listing component status failed: %s", err)
		return
	}
	for _, s := range componentStatus {
		csc.collectComponentStatus(ch, s)
	}
}

func (csc *componentStatusCollector) collectComponentStatus(ch chan<- prometheus.Metric, s v1.ComponentStatus) {
	addConstMetric := func(desc *prometheus.Desc, t prometheus.ValueType, v float64, lv ...string) {
		lv = append([]string{s.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, t, v, lv...)
	}
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	for _, p := range s.Conditions {
		if p.Type == v1.ComponentHealthy {
			addGauge(descComponentStatus, boolFloat64(p.Status == v1.ConditionTrue), string(v1.ConditionTrue))
			addGauge(descComponentStatus, boolFloat64(p.Status == v1.ConditionFalse), string(v1.ConditionFalse))
			addGauge(descComponentStatus, boolFloat64(p.Status == v1.ConditionUnknown), string(v1.ConditionUnknown))
			break
		}
	}
}
