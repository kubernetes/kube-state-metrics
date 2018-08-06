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
	descLimitRangeLabelsDefaultLabels = []string{"limitrange", "namespace"}
	descLimitRange                    = prometheus.NewDesc(
		"kube_limitrange",
		"Information about limit range.",
		append(descLimitRangeLabelsDefaultLabels, "resource", "type", "constraint"),
		nil,
	)

	descLimitRangeCreated = prometheus.NewDesc(
		"kube_limitrange_created",
		"Unix creation timestamp",
		descLimitRangeLabelsDefaultLabels,
		nil,
	)
)

type LimitRangeLister func() (v1.LimitRangeList, error)

func (l LimitRangeLister) List() (v1.LimitRangeList, error) {
	return l()
}

func RegisterLimitRangeCollector(registry prometheus.Registerer, informerFactories []informers.SharedInformerFactory, opts *options.Options) {

	infs := SharedInformerList{}
	for _, f := range informerFactories {
		infs = append(infs, f.Core().V1().LimitRanges().Informer().(cache.SharedInformer))
	}

	limitRangeLister := LimitRangeLister(func() (ranges v1.LimitRangeList, err error) {
		for _, rqinf := range infs {
			for _, rq := range rqinf.GetStore().List() {
				ranges.Items = append(ranges.Items, *(rq.(*v1.LimitRange)))
			}
		}
		return ranges, nil
	})

	registry.MustRegister(&limitRangeCollector{store: limitRangeLister, opts: opts})
	infs.Run(context.Background().Done())
}

type limitRangeStore interface {
	List() (v1.LimitRangeList, error)
}

// limitRangeCollector collects metrics about all limit ranges in the cluster.
type limitRangeCollector struct {
	store limitRangeStore
	opts  *options.Options
}

// Describe implements the prometheus.Collector interface.
func (lrc *limitRangeCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descLimitRange
	ch <- descLimitRangeCreated
}

// Collect implements the prometheus.Collector interface.
func (lrc *limitRangeCollector) Collect(ch chan<- prometheus.Metric) {
	limitRangeCollector, err := lrc.store.List()
	if err != nil {
		ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "limitrange"}).Inc()
		glog.Errorf("listing limit ranges failed: %s", err)
		return
	}
	ScrapeErrorTotalMetric.With(prometheus.Labels{"resource": "limitrange"}).Add(0)

	ResourcesPerScrapeMetric.With(prometheus.Labels{"resource": "limitrange"}).Observe(float64(len(limitRangeCollector.Items)))
	for _, rq := range limitRangeCollector.Items {
		lrc.collectLimitRange(ch, rq)
	}

	glog.V(4).Infof("collected %d limitranges", len(limitRangeCollector.Items))
}

func (lrc *limitRangeCollector) collectLimitRange(ch chan<- prometheus.Metric, rq v1.LimitRange) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{rq.Name, rq.Namespace}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	if !rq.CreationTimestamp.IsZero() {
		addGauge(descLimitRangeCreated, float64(rq.CreationTimestamp.Unix()))
	}

	rawLimitRanges := rq.Spec.Limits
	for _, rawLimitRange := range rawLimitRanges {
		for resource, min := range rawLimitRange.Min {
			addGauge(descLimitRange, float64(min.MilliValue())/1000, string(resource), string(rawLimitRange.Type), "min")
		}

		for resource, max := range rawLimitRange.Max {
			addGauge(descLimitRange, float64(max.MilliValue())/1000, string(resource), string(rawLimitRange.Type), "max")
		}

		for resource, df := range rawLimitRange.Default {
			addGauge(descLimitRange, float64(df.MilliValue())/1000, string(resource), string(rawLimitRange.Type), "default")
		}

		for resource, dfR := range rawLimitRange.DefaultRequest {
			addGauge(descLimitRange, float64(dfR.MilliValue())/1000, string(resource), string(rawLimitRange.Type), "defaultRequest")
		}

		for resource, mLR := range rawLimitRange.MaxLimitRequestRatio {
			addGauge(descLimitRange, float64(mLR.MilliValue())/1000, string(resource), string(rawLimitRange.Type), "maxLimitRequestRatio")
		}

	}

}
