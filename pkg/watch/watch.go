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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type ListWatchMetrics struct {
	WatchTotal *prometheus.CounterVec
	ListTotal  *prometheus.CounterVec
}

// NewListWatchMetrics takes in a prometheus registry and initializes
// and registers the kube_state_metrics_list_total and
// kube_state_metrics_watch_total metrics. It returns those registered metrics.
func NewListWatchMetrics(r *prometheus.Registry) *ListWatchMetrics {
	var m ListWatchMetrics
	m.WatchTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kube_state_metrics_watch_total",
			Help: "Number of total resource watches in kube-state-metrics",
		},
		[]string{"result", "resource"},
	)

	m.ListTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kube_state_metrics_list_total",
			Help: "Number of total resource list in kube-state-metrics",
		},
		[]string{"result", "resource"},
	)
	if r != nil {
		r.MustRegister(
			m.ListTotal,
			m.WatchTotal,
		)
	}
	return &m
}

type InstrumentedListerWatcher struct {
	lw       cache.ListerWatcher
	metrics  *ListWatchMetrics
	resource string
}

// NewInstrumentedListerWatcher returns a new InstrumentedListerWatcher.
func NewInstrumentedListerWatcher(lw cache.ListerWatcher, metrics *ListWatchMetrics, resource string) cache.ListerWatcher {
	return &InstrumentedListerWatcher{
		lw:       lw,
		metrics:  metrics,
		resource: resource,
	}
}

// List is a wrapper func around the cache.ListerWatcher.List func. It increases the success/error
// / counters based on the outcome of the List operation it instruments.
func (i *InstrumentedListerWatcher) List(options metav1.ListOptions) (res runtime.Object, err error) {
	res, err = i.lw.List(options)
	if err != nil {
		i.metrics.ListTotal.WithLabelValues("error", i.resource).Inc()
		return
	}

	i.metrics.ListTotal.WithLabelValues("success", i.resource).Inc()
	return
}

// Watch is a wrapper func around the cache.ListerWatcher.Watch func. It increases the success/error
// counters based on the outcome of the Watch operation it instruments.
func (i *InstrumentedListerWatcher) Watch(options metav1.ListOptions) (res watch.Interface, err error) {
	res, err = i.lw.Watch(options)
	if err != nil {
		i.metrics.WatchTotal.WithLabelValues("error", i.resource).Inc()
		return
	}

	i.metrics.WatchTotal.WithLabelValues("success", i.resource).Inc()
	return
}
