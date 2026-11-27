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
	"runtime"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

var _ cache.TransformingStore = &MetricsStore{}

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
	generateMetricsFunc func(any) []metric.FamilyInterface
	// headers contains the header (TYPE and HELP) of each metric family. It is
	// later on zipped with with their corresponding metric families in
	// MetricStore.WriteAll().
	headers []string

	headersOpenMetrics []string
	headersTextPlain   []string
	metricNames        []string

	// objectLimit is the maximum number of objects kept in the store, 0 means
	// no limit. objectCount tracks the number of objects currently stored and
	// is only maintained when a limit is set. It is only accessed by Add,
	// Delete and Replace, which the reflector calls from a single goroutine,
	// so it needs no synchronization.
	objectLimit int64
	objectCount int64
	// onObjectDropped is called for every add or update of an object that is
	// not stored because objectLimit was reached.
	onObjectDropped func()
}

// Option configures a MetricsStore.
type Option func(*MetricsStore)

// WithObjectLimit limits the number of objects the store keeps. Once the limit
// is reached, new objects are not stored until existing ones are deleted;
// updates to objects already in the store are still applied. onDropped, if not
// nil, is called for every add or update of an object that is not stored. A
// limit of 0 disables it.
func WithObjectLimit(limit int64, onDropped func()) Option {
	return func(s *MetricsStore) {
		s.objectLimit = limit
		s.onObjectDropped = onDropped
	}
}

// NewMetricsStore returns a new MetricsStore
func NewMetricsStore(headers []string, generateFunc func(any) []metric.FamilyInterface, opts ...Option) *MetricsStore {
	rv := ""
	headersOpenMetrics, headersTextPlain, metricNames := precomputeHeaders(headers)
	s := &MetricsStore{
		generateMetricsFunc:   generateFunc,
		headers:               headers,
		headersOpenMetrics:    headersOpenMetrics,
		headersTextPlain:      headersTextPlain,
		metricNames:           metricNames,
		metrics:               &sync.Map{},
		lastResourceVersion:   &rv,
		lastResourceVersionMu: &sync.RWMutex{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// precomputeHeaders parses each header once at store construction, returning the
// rewritten header for both exposition formats plus the metric name it declares.
// SanitizeHeaders runs on every scrape and would otherwise redo this work per
// header, per store, per request.
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
func (s *MetricsStore) Add(obj any) error {
	// Objects rendered by Transformer are only expected in Replace at the end
	// of a WatchList initial sync, which stores them itself; this keeps Add
	// working should a reflector hand one over directly. WatchList is disabled
	// when --object-limit is set, so no per-object bookkeeping is skipped here.
	if r, ok := obj.(*renderedObject); ok {
		s.metrics.Store(r.uid, r.wait())
		return nil
	}

	o, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	s.setLastResourceVersion(o.GetResourceVersion())

	// The object limit has to be enforced here and not only on List: objects
	// created after the initial List, and updates to objects the List dropped,
	// arrive through the watch.
	if !s.admit(o.GetUID()) {
		return nil
	}

	families := s.generateMetricsFunc(obj)

	s.metrics.Store(o.GetUID(), renderFamilies(families))

	return nil
}

// admit reports whether the object with the given UID may be stored under the
// object limit, and counts it if it is new. It calls onObjectDropped for an
// object that is not admitted.
func (s *MetricsStore) admit(uid types.UID) bool {
	if s.objectLimit <= 0 {
		return true
	}
	if _, exists := s.metrics.Load(uid); exists {
		return true
	}
	if s.objectCount >= s.objectLimit {
		if s.onObjectDropped != nil {
			s.onObjectDropped()
		}
		return false
	}
	s.objectCount++
	return true
}

// objectUID returns the UID an object is stored under.
func objectUID(obj any) (types.UID, error) {
	if r, ok := obj.(*renderedObject); ok {
		return r.uid, nil
	}
	o, err := meta.Accessor(obj)
	if err != nil {
		return "", err
	}
	return o.GetUID(), nil
}

// byteAppender is implemented by families that can render themselves into a
// caller-provided buffer, letting all families of one object share a single
// allocation.
type byteAppender interface {
	AppendBytes(b []byte) []byte
	SizeHint() int
}

// renderFamilies renders every family of an object into one contiguous buffer
// and returns per-family views into it. One allocation per object beats one per
// family, both for allocation count and for the size-class rounding that a few
// dozen small slices would otherwise waste.
func renderFamilies(families []metric.FamilyInterface) [][]byte {
	familyStrings := make([][]byte, len(families))
	ends := make([]int, len(families))

	// Size the shared buffer from the families themselves, so appending does not
	// grow it geometrically and overshoot; the result is retained for the
	// lifetime of the object, so slack is memory we would never reclaim.
	hint := 0
	for _, f := range families {
		if a, ok := f.(byteAppender); ok {
			hint += a.SizeHint()
		}
	}

	buf := make([]byte, 0, hint)
	for i, f := range families {
		if a, ok := f.(byteAppender); ok {
			buf = a.AppendBytes(buf)
		} else {
			buf = append(buf, f.ByteSlice()...)
		}
		ends[i] = len(buf)
	}

	// SizeHint over-estimates the value length, so trim the remaining slack
	// rather than retain it for every object in the store.
	if cap(buf)-len(buf) > len(buf)/8 {
		buf = append(make([]byte, 0, len(buf)), buf...)
	}

	start := 0
	for i, end := range ends {
		// Full slice expression: a later append must not write into the next family.
		familyStrings[i] = buf[start:end:end]
		start = end
	}

	return familyStrings
}

// Update updates the existing entry in the MetricsStore.
func (s *MetricsStore) Update(obj any) error {
	// TODO: For now, just call Add, in the future one could check if the resource version changed?
	return s.Add(obj)
}

// Delete deletes an existing entry in the MetricsStore.
func (s *MetricsStore) Delete(obj any) error {

	o, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	s.setLastResourceVersion(o.GetResourceVersion())

	if _, loaded := s.metrics.LoadAndDelete(o.GetUID()); loaded && s.objectLimit > 0 {
		s.objectCount--
	}

	return nil
}

// List implements the List method of the store interface.
func (s *MetricsStore) List() []any {
	return nil
}

// ListKeys implements the ListKeys method of the store interface.
func (s *MetricsStore) ListKeys() []string {
	return nil
}

// Get implements the Get method of the store interface.
func (s *MetricsStore) Get(_ any) (item any, exists bool, err error) {
	return nil, false, nil
}

// GetByKey implements the GetByKey method of the store interface.
func (s *MetricsStore) GetByKey(_ string) (item any, exists bool, err error) {
	return nil, false, nil
}

// Replace will delete the contents of the store, using instead the given list,
// and records the provided resourceVersion as the last sync resource version.
//
// The store is read concurrently by WriteAll, so it is never emptied: the
// objects missing from the list are deleted first, and only then is every
// listed object upserted. A scrape during a relist therefore sees each surviving
// object, rather than whatever subset had been re-added so far, and never sees
// a deleted object next to the objects that were created since.
func (s *MetricsStore) Replace(list []any, resourceVersion string) error {
	// listed holds the UIDs of the list, and whether the object was
	// admitted by this Replace, so that a UID listed twice is counted once.
	listed := make(map[types.UID]bool, len(list))
	for _, o := range list {
		uid, err := objectUID(o)
		if err != nil {
			return err
		}
		listed[uid] = false
	}

	var kept int64
	s.metrics.Range(func(key, _ any) bool {
		if _, ok := listed[key.(types.UID)]; ok {
			kept++
		} else {
			s.metrics.Delete(key)
		}
		return true
	})
	// The objects kept are the ones counted against the limit; the upsert
	// below admits new ones while there is room.
	if s.objectLimit > 0 {
		s.objectCount = kept
	}

	// Render every object that is not rendered yet on the shared workers,
	// and store them in list order. Objects from a WatchList initial sync
	// were handed to the workers by Transformer as they arrived; objects from
	// a List are full objects and are handed over here. Only a few objects
	// are in flight at a time, so that on a relist each object's old metrics
	// are replaced, and can be collected, soon after its new ones are
	// rendered, rather than once the whole list is.
	maxInFlight := 2 * runtime.GOMAXPROCS(0)
	var inFlight []*renderedObject
	storeRendered := func(keep int) {
		for len(inFlight) > keep {
			r := inFlight[0]
			inFlight[0] = nil
			inFlight = inFlight[1:]
			s.metrics.Store(r.uid, r.wait())
		}
	}
	for _, o := range list {
		r, ok := o.(*renderedObject)
		if !ok {
			uid, _ := objectUID(o) // checked above
			// An admitted object is only stored once it is rendered, so
			// admit cannot tell that it was counted already.
			if !listed[uid] {
				if !s.admit(uid) {
					continue
				}
				listed[uid] = true
			}
			r = s.renderAsync(uid, o)
		}
		inFlight = append(inFlight, r)
		storeRendered(maxInFlight)
	}
	storeRendered(0)

	s.setLastResourceVersion(resourceVersion)

	return nil
}

// Resync implements the Resync method of the store interface.
func (s *MetricsStore) Resync() error {
	return nil
}

// Transformer implements the cache.TransformingStore interface. During a
// WatchList initial sync, the reflector holds every object of the resource in
// a temporary store until the initial events end, and applies this transform
// to each object it puts there. Generating the metrics here lets the temporary
// store hold the metrics this store keeps anyway instead of whole objects,
// which lowers the memory peak of the initial sync.
//
// The rendered objects are not runtime.Objects, so client-go's WatchList data
// consistency check (KUBE_WATCHLIST_INCONSISTENCY_DETECTOR) cannot be used with
// this store.
//
// The objects are rendered concurrently, see renderAsync.
func (s *MetricsStore) Transformer() cache.TransformFunc {
	return func(obj any) (any, error) {
		o, err := meta.Accessor(obj)
		if err != nil {
			return obj, nil
		}
		return s.renderAsync(o.GetUID(), obj), nil
	}
}

// renderSlots bounds the number of objects rendered at the same time, across
// all stores. The reflectors of the different resources already sync
// concurrently, so a bound per store would oversubscribe the CPUs.
var renderSlots renderLimiter

// renderLimiter is a semaphore whose size follows GOMAXPROCS, which the
// runtime adjusts when the container's CPU limit changes.
type renderLimiter struct {
	mu      sync.Mutex
	cond    *sync.Cond
	running int
}

func (l *renderLimiter) acquire() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cond == nil {
		l.cond = sync.NewCond(&l.mu)
	}
	for l.running >= runtime.GOMAXPROCS(0) {
		l.cond.Wait()
	}
	l.running++
}

func (l *renderLimiter) release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.running--
	l.cond.Signal()
}

// renderAsync renders obj on a worker and returns at once. When every worker
// is busy it blocks until one is free, so a reflector feeding objects faster
// than they render is slowed down instead of accumulating unrendered objects.
// This keeps the memory peak of a WatchList initial sync close to that of
// rendering inline, while spreading the rendering of a large resource, which
// would otherwise run on its reflector's goroutine alone, over all CPUs.
//
// The store's generate function is therefore called concurrently, and must be
// safe for concurrent use.
func (s *MetricsStore) renderAsync(uid types.UID, obj any) *renderedObject {
	r := &renderedObject{uid: uid}
	renderSlots.acquire()
	r.wg.Go(func() {
		// Report a panic as the reflector goroutine the rendering moved
		// from would have.
		defer utilruntime.HandleCrash()
		defer renderSlots.release()
		r.families = renderFamilies(s.generateMetricsFunc(obj))
	})
	return r
}

// renderedObject stands in for an object in the reflector's temporary store,
// and for a listed object in Replace, while its metrics are rendered.
type renderedObject struct {
	uid types.UID
	wg  sync.WaitGroup
	// families is set once wg is done.
	families [][]byte
}

// wait blocks until the object is rendered and returns its families.
func (r *renderedObject) wait() [][]byte {
	r.wg.Wait()
	return r.families
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
