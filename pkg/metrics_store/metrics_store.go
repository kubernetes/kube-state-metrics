package metricsstore

import (
	"sync"

	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/apimachinery/pkg/api/meta"
)

// MetricsStore implements the k8s.io/kubernetes/client-go/tools/cache.Store
// interface. Instead of storing entire Kubernetes objects, it stores metrics
// generated based on them.
type MetricsStore struct {
	mutex   sync.RWMutex
	metrics map[string][]*metrics.Metric

	generateMetricsFunc func(interface{}) []*metrics.Metric
}

// NewMetricsStore returns a new MetricsStore
func NewMetricsStore(generateFunc func(interface{}) []*metrics.Metric) *MetricsStore {
	return &MetricsStore{
		generateMetricsFunc: generateFunc,
		metrics:             map[string][]*metrics.Metric{},
	}
}

// Implementing k8s.io/kubernetes/client-go/tools/cache.Store interface

// TODO: Proper comments on all functions below.
func (s *MetricsStore) Add(obj interface{}) error {
	o, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.metrics[o.GetName()] = s.generateMetricsFunc(obj)

	return nil
}

func (s *MetricsStore) Update(obj interface{}) error {
	// For now, just call Add, in the future one could check if the resource version changed?
	return s.Add(obj)
}

func (s *MetricsStore) Delete(obj interface{}) error {

	o, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.metrics, o.GetName())

	return nil
}

func (s *MetricsStore) List() []interface{} {
	return nil
}

func (s *MetricsStore) ListKeys() []string {
	return nil
}

func (s *MetricsStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

func (s *MetricsStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	return nil, false, nil
}

// Replace will delete the contents of the store, using instead the
// given list. Store takes ownership of the list, you should not reference
// it after calling this function.
// TODO: Comment necessary?
// TODO: What is 'name' for?
func (s *MetricsStore) Replace(list []interface{}, name string) error {
	s.mutex.Lock()
	s.metrics = map[string][]*metrics.Metric{}
	s.mutex.Unlock()

	for _, o := range list {
		err := s.Add(o)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *MetricsStore) Resync() error {
	return nil
}

func (s *MetricsStore) GetAll() []*metrics.Metric {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	m := make([]*metrics.Metric, 0, len(s.metrics))

	for _, metrics := range s.metrics {
		m = append(m, metrics...)
	}

	return m
}
