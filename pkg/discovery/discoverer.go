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

package discovery

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/rest"

	"k8s.io/kube-state-metrics/v2/internal/discovery"
)

// Discoverer is an interface that provides a cache of the collected GVKs, along with helper utilities.
type Discoverer interface {
	SafeRead(f func())
	SafeWrite(f func())
	StartDiscovery(ctx context.Context, config *rest.Config) error
}

// ensure interface matches the struct that implements it
var _ Discoverer = &discovery.CRDiscoverer{}

// NewDiscoverer creates a new Discoverer.
func NewDiscoverer(crdsAddEventsCounter, crdsUpdateEventsCounter, crdsDeleteEventsCounter prometheus.Counter, crdsCacheCountGauge prometheus.Gauge) Discoverer {
	return discovery.NewCRDiscoverer(crdsAddEventsCounter, crdsUpdateEventsCounter, crdsDeleteEventsCounter, crdsCacheCountGauge)
}
