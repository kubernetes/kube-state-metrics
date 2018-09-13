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
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	descSecretLabelsName          = "kube_secret_labels"
	descSecretLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descSecretLabelsDefaultLabels = []string{"namespace", "secret"}

	descSecretInfo = prometheus.NewDesc(
		"kube_secret_info",
		"Information about secret.",
		descSecretLabelsDefaultLabels,
		nil,
	)

	descSecretType = prometheus.NewDesc(
		"kube_secret_type",
		"Type about secret.",
		append(descSecretLabelsDefaultLabels, "type"),
		nil,
	)

	descSecretLabels = prometheus.NewDesc(
		descSecretLabelsName,
		descSecretLabelsHelp,
		descSecretLabelsDefaultLabels,
		nil,
	)

	descSecretCreated = prometheus.NewDesc(
		"kube_secret_created",
		"Unix creation timestamp",
		descSecretLabelsDefaultLabels,
		nil,
	)

	descSecretMetadataResourceVersion = prometheus.NewDesc(
		"kube_secret_metadata_resource_version",
		"Resource version representing a specific version of secret.",
		append(descSecretLabelsDefaultLabels, "resource_version"),
		nil,
	)
)

type SecretLister func() ([]v1.Secret, error)

func (l SecretLister) List() ([]v1.Secret, error) {
	return l()
}

func RegisterSecretCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Core().V1().Secrets().Informer().(cache.SharedInformer))
	}

	secretLister := SecretLister(func() (secrets []v1.Secret, err error) {
		for _, sinf := range infs {
			for _, m := range sinf.GetStore().List() {
				secrets = append(secrets, *m.(*v1.Secret))
			}
		}
		return secrets, nil
	})

	registry.MustRegister(&secretCollector{store: secretLister, opts: opts})
	infs.Run(context.Background().Done())
}

type secretStore interface {
	List() (secrets []v1.Secret, err error)
}

// secretCollector collects metrics about all secrets in the cluster.
type secretCollector struct {
	store secretStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (sc *secretCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descSecretInfo
	ch <- descSecretCreated
	ch <- descSecretLabels
	ch <- descSecretMetadataResourceVersion
	ch <- descSecretType
}

// Collect implements the prometheus.Collector interface.
func (sc *secretCollector) Collect(ch chan<- prometheus.Metric) {
	secrets, err := sc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "secret"}).Inc()
		glog.Errorf("listing secrets failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "secret"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "secret"}).Observe(float64(len(secrets)))
	for _, s := range secrets {
		sc.collectSecret(ch, s)
	}

	glog.V(4).Infof("collected %d secrets", len(secrets))
}

func secretLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descSecretLabelsName,
		descSecretLabelsHelp,
		append(descSecretLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func (sc *secretCollector) collectSecret(ch chan<- prometheus.Metric, s v1.Secret) {
	addConstMetric := func(desc *prometheus.Desc, t prometheus.ValueType, v float64, lv ...string) {
		lv = append([]string{s.Namespace, s.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, t, v, lv...)
	}
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addGauge(descSecretInfo, 1)

	addGauge(descSecretType, 1, string(s.Type))
	if !s.CreationTimestamp.IsZero() {
		addGauge(descSecretCreated, float64(s.CreationTimestamp.Unix()))
	}
	labelKeys, labelValues := kubeLabelsToPrometheusLabels(s.Labels)
	addGauge(secretLabelsDesc(labelKeys), 1, labelValues...)

	addGauge(descSecretMetadataResourceVersion, 1, string(s.ObjectMeta.ResourceVersion))
}
