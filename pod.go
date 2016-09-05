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
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
)

var (
	descContainerRestarts = prometheus.NewDesc(
		"pod_container_restarts",
		"The number of container restarts per container.",
		[]string{"namespace", "pod", "container"}, nil,
	)
)

type podStore interface {
	List(selector labels.Selector) (pods []*api.Pod, err error)
}

// podCollector collects metrics about all pods in the cluster.
type podCollector struct {
	store podStore
}

// Describe implements the prometheus.Collector interface.
func (pc *podCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descContainerRestarts
}

// Collect implements the prometheus.Collector interface.
func (pc *podCollector) Collect(ch chan<- prometheus.Metric) {
	pods, err := pc.store.List(labels.Everything())
	if err != nil {
		glog.Errorf("listing pods failed: %s", err)
		return
	}
	for _, p := range pods {
		for _, m := range pc.collectPod(p) {
			ch <- m
		}
	}
}

func (pc *podCollector) collectPod(p *api.Pod) (res []prometheus.Metric) {
	for _, cs := range p.Status.ContainerStatuses {
		res = append(res, prometheus.MustNewConstMetric(
			descContainerRestarts, prometheus.CounterValue, float64(cs.RestartCount),
			p.Namespace, p.Name, cs.Name,
		))
	}

	return
}
