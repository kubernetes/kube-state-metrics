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

package main

import (
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/1.5/pkg/api/v1"
)

var (
	descPodNumOfHealthcheckFailures = prometheus.NewDesc(
		"kube_pod_healthcheck_num_of_failures",
		"Number of healthcheck failures for a given pod.",
		[]string{"namespace", "pod"}, nil,
	)
	descSecondsSinceLastHealthcheckFailure = prometheus.NewDesc(
		"kube_pod_healthcheck_seconds_since_last_failure",
		"Number of seconds since of last healthcheck failure.",
		[]string{"namespace", "pod"}, nil,
	)
)

type eventStore interface {
	List() (events []v1.Event, err error)
}

// eventCollector collects metrics about selected events in the cluster.
type eventCollector struct {
	store eventStore
}

// Describe implements the prometheus.Collector interface.
func (pc *eventCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descPodNumOfHealthcheckFailures
	ch <- descSecondsSinceLastHealthcheckFailure
}

// Collect implements the prometheus.Collector interface.
func (pc *eventCollector) Collect(ch chan<- prometheus.Metric) {
	events, err := pc.store.List()
	if err != nil {
		glog.Errorf("listing events failed: %s", err)
		return
	}
	for _, ev := range events {
		pc.collectEvent(ch, ev)
	}
}

func (pc *eventCollector) collectEvent(ch chan<- prometheus.Metric, ev v1.Event) {
	addConstMetric := func(desc *prometheus.Desc, t prometheus.ValueType, v float64, lv ...string) {
		lv = append([]string{ev.Namespace, ev.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, t, v, lv...)
	}
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	if ev.InvolvedObject.Kind == "Pod" {
		if (ev.Reason == "Unhealthy") && (ev.Count > 0) {
			addGauge(descPodNumOfHealthcheckFailures, float64(ev.Count))
			addGauge(descSecondsSinceLastHealthcheckFailure,
				float64(int(time.Since(ev.LastTimestamp.Time).Seconds())))
		}
	}
}
