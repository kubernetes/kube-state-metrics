package app

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

type FilteredRegistry struct {
	*prometheus.Registry
	metricsFilter generator.CompositeFamilyGeneratorFilter
}

func NewFilteredGatherer(r *prometheus.Registry, metricsFilter generator.CompositeFamilyGeneratorFilter) *FilteredRegistry {
	return &FilteredRegistry{Registry: r, metricsFilter: metricsFilter}
}

func (f *FilteredRegistry) Gather() ([]*dto.MetricFamily, error) {
	families, err := f.Registry.Gather()
	if err != nil {
		return nil, err
	}

	var filtered []*dto.MetricFamily
	for _, family := range families {
		gen := generator.FamilyGenerator{Name: *family.Name}
		if f.metricsFilter.Test(gen) {
			filtered = append(filtered, family)
		}
	}

	return filtered, nil
}
