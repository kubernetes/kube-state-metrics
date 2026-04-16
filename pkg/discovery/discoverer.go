package discovery

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/rest"
	"k8s.io/kube-state-metrics/v2/internal/discovery"
)

type Discoverer interface {
	SafeRead(f func())
	SafeWrite(f func())
	StartDiscovery(ctx context.Context, config *rest.Config) error
}

func NewDiscoverer(crdsAddEventsCounter, crdsUpdateEventsCounter, crdsDeleteEventsCounter prometheus.Counter, crdsCacheCountGauge prometheus.Gauge) Discoverer {
	return discovery.NewCRDiscoverer(crdsAddEventsCounter, crdsUpdateEventsCounter, crdsDeleteEventsCounter, crdsCacheCountGauge)
}
