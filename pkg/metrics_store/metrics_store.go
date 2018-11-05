package metricsstore

import (
	"io"
	"sync"

	"k8s.io/kube-state-metrics/pkg/metrics"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
)

var (
	helpPrefix = []byte("# HELP ")
)

// MetricsStore implements the k8s.io/kubernetes/client-go/tools/cache.Store
// interface. Instead of storing entire Kubernetes objects, it stores metrics
// generated based on them.
type MetricsStore struct {
	// Protects metrics
	mutex sync.RWMutex
	// metrics is a map indexed by Kubernetes object id, containing a slice of
	// metric families, containing a slice of metrics. We need to keep metrics
	// grouped by metric families in order to zip families with their help text in
	// MetricsStore.WriteAll().
	metrics map[types.UID][][]*metrics.Metric
	// helpTexts is later on zipped with with their corresponding metric
	// families in MetricStore.WriteAll().
	helpTexts []string

	generateMetricsFunc func(interface{}) [][]*metrics.Metric
}

// NewMetricsStore returns a new MetricsStore
func NewMetricsStore(helpTexts []string, generateFunc func(interface{}) [][]*metrics.Metric) *MetricsStore {
	return &MetricsStore{
		generateMetricsFunc: generateFunc,
		helpTexts:           helpTexts,
		metrics:             map[types.UID][][]*metrics.Metric{},
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

	s.metrics[o.GetUID()] = s.generateMetricsFunc(obj)

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

	delete(s.metrics, o.GetUID())

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
	s.metrics = map[types.UID][][]*metrics.Metric{}
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

// WriteAll writes all metrics of the store into the given writer, zipped with the
// help text of each metric family.
func (s *MetricsStore) WriteAll(w io.Writer) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for i, help := range s.helpTexts {
		w.Write(append(helpPrefix, []byte(help)...))
		w.Write([]byte{'\n'})
		for _, metricsPerObject := range s.metrics {
			for _, metric := range metricsPerObject[i] {
				w.Write([]byte(*metric))
			}
		}
	}
}
