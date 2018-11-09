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
	"k8s.io/client-go/kubernetes"
)

var (
	descConfigMapInfo = prometheus.NewDesc(
		"kube_configmap_info",
		"Information about configmap.",
		[]string{"namespace", "configmap"}, nil,
	)

	descConfigMapCreated = prometheus.NewDesc(
		"kube_configmap_created",
		"Unix creation timestamp",
		[]string{"namespace", "configmap"}, nil,
	)

	descConfigMapMetadataResourceVersion = prometheus.NewDesc(
		"kube_configmap_metadata_resource_version",
		"Resource version representing a specific version of the configmap.",
		[]string{"namespace", "configmap", "resource_version"}, nil,
	)
)

type ConfigMapLister func() ([]v1.ConfigMap, error)

func (l ConfigMapLister) List() ([]v1.ConfigMap, error) {
	return l()
}

func RegisterConfigMapCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface, namespaces []string) {
	client := kubeClient.CoreV1().RESTClient()
	glog.Infof("collect configmap with %s", client.APIVersion())

	cminfs := NewSharedInformerList(client, "configmaps", namespaces, &v1.ConfigMap{})

	configMapLister := ConfigMapLister(func() (configMaps []v1.ConfigMap, err error) {
		for _, cminf := range *cminfs {
			for _, m := range cminf.GetStore().List() {
				configMaps = append(configMaps, *m.(*v1.ConfigMap))
			}
		}
		return configMaps, nil
	})

	registry.MustRegister(&configMapCollector{store: configMapLister})
	cminfs.Run(context.Background().Done())
}

type configMapStore interface {
	List() (configMaps []v1.ConfigMap, err error)
}

// configMapCollector collects metrics about all configMaps in the cluster.
type configMapCollector struct {
	store configMapStore
}

// Describe implements the prometheus.Collector interface.
func (sc *configMapCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descConfigMapInfo
	ch <- descConfigMapCreated
	ch <- descConfigMapMetadataResourceVersion
}

// Collect implements the prometheus.Collector interface.
func (cmc *configMapCollector) Collect(ch chan<- prometheus.Metric) {
	configMaps, err := cmc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "configmap"}).Inc()
		glog.Errorf("listing configmaps failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "configmap"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "configmap"}).Observe(float64(len(configMaps)))
	for _, s := range configMaps {
		cmc.collectConfigMap(ch, s)
	}

	glog.V(4).Infof("collected %d configmaps", len(configMaps))
}

func (cmc *configMapCollector) collectConfigMap(ch chan<- prometheus.Metric, s v1.ConfigMap) {
	addConstMetric := func(desc *prometheus.Desc, t prometheus.ValueType, v float64, lv ...string) {
		lv = append([]string{s.Namespace, s.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, t, v, lv...)
	}
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addGauge(descConfigMapInfo, 1)

	if !s.CreationTimestamp.IsZero() {
		addGauge(descConfigMapCreated, float64(s.CreationTimestamp.Unix()))
	}

	addGauge(descConfigMapMetadataResourceVersion, 1, string(s.ObjectMeta.ResourceVersion))
}
