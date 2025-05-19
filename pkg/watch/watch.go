/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

package watch

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// ListWatchMetrics stores the pointers of kube_state_metrics_[list|watch]_total metrics.
type ListWatchMetrics struct {
	WatchRequestsTotal *prometheus.CounterVec
	ListRequestsTotal  *prometheus.CounterVec
	ListObjectsLimit   *prometheus.GaugeVec
	ListObjectsCurrent *prometheus.GaugeVec
}

// NewListWatchMetrics takes in a prometheus registry and initializes
// and registers the kube_state_metrics_list_total and
// kube_state_metrics_watch_total metrics. It returns those registered metrics.
func NewListWatchMetrics(r prometheus.Registerer) *ListWatchMetrics {
	return &ListWatchMetrics{
		WatchRequestsTotal: promauto.With(r).NewCounterVec(
			prometheus.CounterOpts{
				Name: "kube_state_metrics_watch_total",
				Help: "Number of total resource watch calls in kube-state-metrics",
			},
			[]string{"result", "resource"},
		),
		ListRequestsTotal: promauto.With(r).NewCounterVec(
			prometheus.CounterOpts{
				Name: "kube_state_metrics_list_total",
				Help: "Number of total resource list calls in kube-state-metrics",
			},
			[]string{"result", "resource"},
		),
		ListObjectsCurrent: promauto.With(r).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kube_state_metrics_list_objects",
				Help: "Number of resources listed in kube-state-metrics",
			},
			[]string{"resource"},
		),
		ListObjectsLimit: promauto.With(r).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kube_state_metrics_list_objects_limit",
				Help: "Number of resource list limit in kube-state-metrics",
			},
			[]string{"resource"},
		),
	}
}

// InstrumentedListerWatcher provides the kube_state_metrics_watch_total metric
// with a cache.ListerWatcher obj and the related resource.
type InstrumentedListerWatcher struct {
	lw                cache.ListerWatcher
	metrics           *ListWatchMetrics
	resource          string
	useAPIServerCache bool
	limit             int64
}

// NewInstrumentedListerWatcher returns a new InstrumentedListerWatcher.
func NewInstrumentedListerWatcher(lw cache.ListerWatcher, metrics *ListWatchMetrics, resource string, useAPIServerCache bool, limit int64) cache.ListerWatcher {
	return &InstrumentedListerWatcher{
		lw:                lw,
		metrics:           metrics,
		resource:          resource,
		useAPIServerCache: useAPIServerCache,
		limit:             limit,
	}
}

// List is a wrapper func around the cache.ListerWatcher.List func. It increases the success/error
// / counters based on the outcome of the List operation it instruments.
// It supports setting object limits, this means if it is set it will only list and process
// n objects of the same resource type.
func (i *InstrumentedListerWatcher) List(options metav1.ListOptions) (runtime.Object, error) {

	if i.useAPIServerCache {
		options.ResourceVersion = "0"
	}

	if i.limit > 0 {
		options.Limit = i.limit
		i.metrics.ListObjectsLimit.WithLabelValues(i.resource).Set(float64(i.limit))
	}

	res, err := i.lw.List(options)

	if err != nil {
		i.metrics.ListRequestsTotal.WithLabelValues("error", i.resource).Inc()
		return nil, err
	}

	list, err := meta.ExtractList(res)
	if err != nil {
		return nil, err
	}
	i.metrics.ListRequestsTotal.WithLabelValues("success", i.resource).Inc()

	if i.limit > 0 {
		if int64(len(list)) > i.limit {
			meta.SetList(res, list[0:i.limit])
			i.metrics.ListObjectsCurrent.WithLabelValues(i.resource).Set(float64(i.limit))
		} else {
			i.metrics.ListObjectsCurrent.WithLabelValues(i.resource).Set(float64(len(list)))
		}
	}

	return res, nil

}

// Watch is a wrapper func around the cache.ListerWatcher.Watch func. It increases the success/error
// counters based on the outcome of the Watch operation it instruments.
func (i *InstrumentedListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	res, err := i.lw.Watch(options)
	if err != nil {
		i.metrics.WatchRequestsTotal.WithLabelValues("error", i.resource).Inc()
		return nil, err
	}

	i.metrics.WatchRequestsTotal.WithLabelValues("success", i.resource).Inc()
	return res, nil
}
