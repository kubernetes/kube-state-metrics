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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

// MetricsStore implements the k8s.io/client-go/tools/cache.Store
// interface. Instead of storing entire Kubernetes objects, it stores metrics
// generated based on those objects.
type MetricsStore struct {
	// metrics is a map indexed by Kubernetes object id, containing a slice of
	// metric families, containing a slice of metrics. We need to keep metrics
	// grouped by metric families in order to zip families with their help text in
	// MetricsStore.WriteAll().
	metrics sync.Map

	// generateMetricsFunc generates metrics based on a given Kubernetes object
	// and returns them grouped by metric family.
	generateMetricsFunc func(interface{}) []metric.FamilyInterface
	// headers contains the header (TYPE and HELP) of each metric family. It is
	// later on zipped with with their corresponding metric families in
	// MetricStore.WriteAll().
	headers []string

	predicates []Predicate
}

type Predicate func(obj interface{}) (bool, error)

type Opt func(*MetricsStore)

// NewMetricsStore returns a new MetricsStore
func NewMetricsStore(headers []string, generateFunc func(interface{}) []metric.FamilyInterface, opts ...Opt) *MetricsStore {
	s := &MetricsStore{
		generateMetricsFunc: generateFunc,
		headers:             headers,
		metrics:             sync.Map{},
		predicates:          []Predicate{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func WithNamespacesPredicate(namespaces []string) Opt {
	return func(s *MetricsStore) {
		s.predicates = append(s.predicates, func(obj interface{}) (bool, error) {
			o, err := meta.Accessor(obj)
			if err != nil {
				return false, err
			}

			if o.GetNamespace() == metav1.NamespaceNone {
				// cluster-scoped objects are excluded from this predicate
				return true, nil
			}

			for _, namespace := range namespaces {
				if o.GetNamespace() == namespace {
					return true, nil
				}
			}
			return false, nil
		})
	}
}

// Implementing k8s.io/client-go/tools/cache.Store interface

// Add inserts adds to the MetricsStore by calling the metrics generator functions and
// adding the generated metrics to the metrics map that underlies the MetricStore.
func (s *MetricsStore) Add(obj interface{}) error {
	for _, p := range s.predicates {
		ok, err := p(obj)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	o, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

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

// Replace will delete the contents of the store, using instead the
// given list.
func (s *MetricsStore) Replace(list []interface{}, _ string) error {
	s.metrics.Clear()

	for _, o := range list {
		err := s.Add(o)
		if err != nil {
			return err
		}
	}

	return nil
}

// Resync implements the Resync method of the store interface.
func (s *MetricsStore) Resync() error {
	return nil
}
