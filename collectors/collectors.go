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
)

var (
	resyncPeriod = 5 * time.Minute

	kubeStateMetricsSubsystem = "ksm"

	ScrapeErrorTotalMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: kubeStateMetricsSubsystem,
			Name:      "scrape_error_total",
			Help:      "Total scrape errors encountered when scraping a resource",
		},
		[]string{"resource"},
	)

	ResourcesPerScrapeMetric = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Subsystem: kubeStateMetricsSubsystem,
			Name:      "resources_per_scrape",
			Help:      "Number of resources returned per scrape",
		},
		[]string{"resource"},
	)
)
