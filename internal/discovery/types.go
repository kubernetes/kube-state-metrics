/*
Copyright 2023 The Kubernetes Authors All rights reserved.
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
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DiscoveredResource represents a discovered custom resource type.
type DiscoveredResource struct {
	schema.GroupVersionKind
	Plural string
}

// String returns a string representation of the DiscoveredResource.
func (d DiscoveredResource) String() string {
	return fmt.Sprintf("%s/%s, Kind=%s, Plural=%s", d.Group, d.Version, d.Kind, d.Plural)
}

// resourceEntry is the internal cache entry: a DiscoveredResource plus the
// stop channel used to stop its reflector.
type resourceEntry struct {
	DiscoveredResource
	stopChan chan struct{}
}

// extractor extracts DiscoveredResources from source objects (e.g. CRDs).
type extractor interface {
	// SourceID returns a unique identifier for the source object,
	// e.g. "crd:<name>". Returns "" if obj cannot be identified.
	SourceID(obj interface{}) string
	// ExtractGVKs returns the resources discovered from obj.
	// Return nil to skip; return an empty slice to signal deletion of all
	// resources for the source.
	ExtractGVKs(obj interface{}) []DiscoveredResource
}

// CRDiscoverer provides discovery and lifecycle management for custom resources.
type CRDiscoverer struct {
	// mu protects all fields below.
	mu sync.RWMutex
	// resourcesBySource maps source objects to their discovered resources.
	// Keys e.g. "crd:<name>"
	resourcesBySource map[string][]resourceEntry
	// wasUpdated indicates whether the cache was updated since last check.
	wasUpdated bool

	// Metrics for discovery events.
	// AddEvents counts add operations.
	AddEvents prometheus.Counter
	// UpdateEvents counts update operations.
	UpdateEvents prometheus.Counter
	// DeleteEvents counts source deletions.
	DeleteEvents prometheus.Counter
	// CacheCount tracks the current number of discovered resources.
	CacheCount prometheus.Gauge
}

// NewCRDiscoverer creates a new CRDiscoverer instance.
func NewCRDiscoverer(
	addEvents prometheus.Counter,
	updateEvents prometheus.Counter,
	deleteEvents prometheus.Counter,
	cacheCount prometheus.Gauge,
) *CRDiscoverer {
	return &CRDiscoverer{
		resourcesBySource: make(map[string][]resourceEntry),
		AddEvents:         addEvents,
		UpdateEvents:      updateEvents,
		DeleteEvents:      deleteEvents,
		CacheCount:        cacheCount,
	}
}

// UpdateSource replaces all resources for a source.
//   - sourceID == "": noop.
//   - resources == nil: noop (signals "no change").
//   - len(resources) == 0: equivalent to DeleteSource(sourceID).
func (r *CRDiscoverer) UpdateSource(sourceID string, resources []DiscoveredResource) {
	if sourceID == "" {
		return
	}
	if resources == nil {
		return
	}

	if len(resources) == 0 {
		r.DeleteSource(sourceID)
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	_, existing := r.resourcesBySource[sourceID]

	// Stop reflectors for the previous resources before replacing them.
	r.closeStopChans(r.resourcesBySource[sourceID])

	newEntries := make([]resourceEntry, 0, len(resources))
	for _, res := range resources {
		newEntries = append(newEntries, resourceEntry{
			DiscoveredResource: res,
			stopChan:           make(chan struct{}),
		})
	}

	r.resourcesBySource[sourceID] = newEntries
	r.wasUpdated = true

	if existing {
		r.UpdateEvents.Inc()
	} else {
		r.AddEvents.Inc()
	}

	r.updateCacheCountLocked()
}

// DeleteSource removes all resources for a source and stops their reflectors.
func (r *CRDiscoverer) DeleteSource(sourceID string) {
	if sourceID == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleteSourceLocked(sourceID)
}

// deleteSourceLocked removes a source and updates metrics. Caller must hold r.mu.
func (r *CRDiscoverer) deleteSourceLocked(sourceID string) {
	entries, ok := r.resourcesBySource[sourceID]
	if !ok {
		return
	}

	r.closeStopChans(entries)
	delete(r.resourcesBySource, sourceID)
	r.wasUpdated = true

	r.DeleteEvents.Inc()
	r.updateCacheCountLocked()
}

// closeStopChans closes every non-nil stop channel and nils it out so a
// subsequent close is a no-op rather than a panic.
func (r *CRDiscoverer) closeStopChans(entries []resourceEntry) {
	for i := range entries {
		if entries[i].stopChan != nil {
			close(entries[i].stopChan)
			entries[i].stopChan = nil
		}
	}
}

// GetStopChan returns the stop channel for the given GVK. A (group, version,
// kind) tuple is unique across all sources (Kubernetes enforces this at the
// API discovery layer), so the first match is the only match.
func (r *CRDiscoverer) GetStopChan(gvk schema.GroupVersionKind) (chan struct{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, entries := range r.resourcesBySource {
		for _, entry := range entries {
			if entry.GroupVersionKind == gvk {
				return entry.stopChan, true
			}
		}
	}
	return nil, false
}

// Resolve resolves a GVK pattern to matching DiscoveredResources.
// Group is required and cannot be a wildcard. Version and Kind may be "*".
func (r *CRDiscoverer) Resolve(gvk schema.GroupVersionKind) ([]DiscoveredResource, error) {
	g := gvk.Group
	v := gvk.Version
	k := gvk.Kind

	if g == "" || g == "*" {
		return nil, fmt.Errorf("group is required in the defined GVK %v", gvk)
	}

	hasVersion := v != "" && v != "*"
	hasKind := k != "" && k != "*"

	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []DiscoveredResource
	for _, entries := range r.resourcesBySource {
		for _, entry := range entries {
			if entry.Group != g {
				continue
			}
			if hasVersion && entry.Version != v {
				continue
			}
			if hasKind && entry.Kind != k {
				continue
			}
			results = append(results, entry.DiscoveredResource)
			// A fully-specified GVK has at most one match.
			if hasVersion && hasKind {
				return results, nil
			}
		}
	}
	return results, nil
}

// CheckAndResetUpdated checks if the cache was updated and resets the flag.
func (r *CRDiscoverer) CheckAndResetUpdated() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	updated := r.wasUpdated
	r.wasUpdated = false
	return updated
}

// updateCacheCountLocked updates the cache count gauge. Must be called with mu held.
func (r *CRDiscoverer) updateCacheCountLocked() {
	count := 0
	for _, entries := range r.resourcesBySource {
		count += len(entries)
	}
	r.CacheCount.Set(float64(count))
}
