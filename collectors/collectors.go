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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var (
	resyncPeriod = 5 * time.Minute

	ScrapeErrorTotalMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ksm_scrape_error_total",
			Help: "Total scrape errors encountered when scraping a resource",
		},
		[]string{"resource"},
	)

	ResourcesPerScrapeMetric = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "ksm_resources_per_scrape",
			Help: "Number of resources returned per scrape",
		},
		[]string{"resource"},
	)
)

type SharedInformerList []cache.SharedInformer

func NewSharedInformerList(client rest.Interface, resource string, namespaces []string, objType runtime.Object) *SharedInformerList {
	sinfs := SharedInformerList{}
	for _, namespace := range namespaces {
		slw := cache.NewListWatchFromClient(client, resource, namespace, fields.Everything())
		sinfs = append(sinfs, cache.NewSharedInformer(slw, objType, resyncPeriod))
	}
	return &sinfs
}

func (sil SharedInformerList) Run(stopCh <-chan struct{}) {
	for _, sinf := range sil {
		go sinf.Run(stopCh)
	}
}
