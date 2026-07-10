/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package metricsstore

import (
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

// MetricsStore implements the k8s.io/client-go/tools/cache.Store
// interface. Instead of storing entire Kubernetes objects, it stores metrics
// generated based on those objects.
type MetricsStore struct {
	// metrics points to a sync.Map indexed by Kubernetes object id, containing a slice of
	// metric families, containing a slice of metrics. It's a pointer so cloned stores can
	// safely share the same backing storage without copying or mutating it.
	metrics *sync.Map

	// lastResourceVersion points to a string containing the last resource version seen.
	// It's a pointer so cloned stores can safely share the same resource version.
	lastResourceVersion *string
	// lastResourceVersionMu points to a mutex protecting lastResourceVersion.
	lastResourceVersionMu *sync.RWMutex

	// generateMetricsFunc generates metrics based on a given Kubernetes object
	// and returns them grouped by metric family.
	generateMetricsFunc func(interface{}) []metric.FamilyInterface
	// headers contains the header (TYPE and HELP) of each metric family. It is
	// later on zipped with with their corresponding metric families in
	// MetricStore.WriteAll().
	headers []string

	headersOpenMetrics []string
	headersTextPlain   []string
	metricNames        []string
}

// NewMetricsStore returns a new MetricsStore
func NewMetricsStore(headers []string, generateFunc func(interface{}) []metric.FamilyInterface) *MetricsStore {
	rv := ""
	headersOpenMetrics, headersTextPlain, metricNames := precomputeHeaders(headers)
	return &MetricsStore{
		generateMetricsFunc:   generateFunc,
		headers:               headers,
		headersOpenMetrics:    headersOpenMetrics,
		headersTextPlain:      headersTextPlain,
		metricNames:           metricNames,
		metrics:               &sync.Map{},
		lastResourceVersion:   &rv,
		lastResourceVersionMu: &sync.RWMutex{},
	}
}

func precomputeHeaders(headers []string) (headersOpenMetrics, headersTextPlain, metricNames []string) {
	headersOpenMetrics = make([]string, len(headers))
	headersTextPlain = make([]string, len(headers))
	metricNames = make([]string, len(headers))
	for i, h := range headers {
		mName, rHeaderOM := parseHeaderStatic(h, false)
		_, rHeaderText := parseHeaderStatic(h, true)
		headersOpenMetrics[i] = rHeaderOM
		headersTextPlain[i] = rHeaderText
		metricNames[i] = mName
	}
	return headersOpenMetrics, headersTextPlain, metricNames
}

// Implementing k8s.io/client-go/tools/cache.Store interface

// Add inserts adds to the MetricsStore by calling the metrics generator functions and
// adding the generated metrics to the metrics map that underlies the MetricStore.
func (s *MetricsStore) Add(obj interface{}) error {
	o, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	s.setLastResourceVersion(o.GetResourceVersion())

	families := s.generateMetricsFunc(obj)
	familyStrings := make([][]byte, len(families))

	for i, f := range families {
		familyStrings[i] = f.ByteSlice()
	}

	s.metrics.Store(o.GetUID(), familyStrings)

	return nil
}

// Update updates the existing entry in the MetricsStore.
func (s *MetricsStore) Update(obj interface{}) error {
	// TODO: For now, just call Add, in the future one could check if the resource version changed?
	return s.Add(obj)
}

// Delete deletes an existing entry in the MetricsStore.
func (s *MetricsStore) Delete(obj interface{}) error {

	o, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	s.setLastResourceVersion(o.GetResourceVersion())

	s.metrics.Delete(o.GetUID())

	return nil
}

// List implements the List method of the store interface.
func (s *MetricsStore) List() []interface{} {
	return nil
}

// ListKeys implements the ListKeys method of the store interface.
func (s *MetricsStore) ListKeys() []string {
	return nil
}

// Get implements the Get method of the store interface.
func (s *MetricsStore) Get(_ interface{}) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

// GetByKey implements the GetByKey method of the store interface.
func (s *MetricsStore) GetByKey(_ string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

// Replace will delete the contents of the store, using instead the given list,
// and records the provided resourceVersion as the last sync resource version.
func (s *MetricsStore) Replace(list []interface{}, resourceVersion string) error {
	s.metrics.Clear()

	for _, o := range list {
		err := s.Add(o)
		if err != nil {
			return err
		}
	}

	s.setLastResourceVersion(resourceVersion)

	return nil
}

// Resync implements the Resync method of the store interface.
func (s *MetricsStore) Resync() error {
	return nil
}

// Bookmark implements the Bookmark method of the store interface.
func (s *MetricsStore) Bookmark(resourceVersion string) {
	s.setLastResourceVersion(resourceVersion)
}

// LastStoreSyncResourceVersion implements the LastStoreSyncResourceVersion method of the store interface.
func (s *MetricsStore) LastStoreSyncResourceVersion() string {
	s.lastResourceVersionMu.RLock()
	defer s.lastResourceVersionMu.RUnlock()
	return *s.lastResourceVersion
}

func (s *MetricsStore) setLastResourceVersion(rv string) {
	s.lastResourceVersionMu.Lock()
	defer s.lastResourceVersionMu.Unlock()
	*s.lastResourceVersion = rv
}
