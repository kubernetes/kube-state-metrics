/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package sharding

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// LabelOrdinal is name of Prometheus metric label to use in conjunction with kube_state_metrics_shard_ordinal.
	LabelOrdinal = "shard_ordinal"
)

// Metrics stores the pointers of kube_state_metrics_shard_ordinal
// and kube_state_metrics_total_shards metrics.
type Metrics struct {
	Ordinal *prometheus.GaugeVec
	Total   prometheus.Gauge
}

// NewShardingMetrics takes in a prometheus registry and initializes
// and registers sharding configuration metrics. It returns those registered metrics.
func NewShardingMetrics(r prometheus.Registerer) *Metrics {
	return &Metrics{
		Ordinal: promauto.With(r).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "kube_state_metrics_shard_ordinal",
				Help: "Current sharding ordinal/index of this instance",
			}, []string{LabelOrdinal},
		),
		Total: promauto.With(r).NewGauge(
			prometheus.GaugeOpts{
				Name: "kube_state_metrics_total_shards",
				Help: "Number of total shards this instance is aware of",
			},
		),
	}
}
