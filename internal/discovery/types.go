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
	Plural   string
	stopChan chan struct{}
}

// String returns a string representation of the DiscoveredResource.
func (d DiscoveredResource) String() string {
	return fmt.Sprintf("%s/%s, Kind=%s, Plural=%s", d.Group, d.Version, d.Kind, d.Plural)
}

// extractor defines the interface for extracting DiscoveredResources
type extractor interface {
	// SourceID returns a unique identifier for the source object.
	// For CRDs: "crd:<name>"
	SourceID(obj interface{}) string
	// ExtractGVKs extracts discovered resources from the object.
	// Return nil to skip, empty array to signal deletion of all resources for the source.
	ExtractGVKs(obj interface{}) []*DiscoveredResource
}

// CRDiscoverer provides discovery and lifecycle management for custom resources.
type CRDiscoverer struct {
	// mu protects all fields below.
	mu sync.RWMutex
	// resourcesBySource maps source objects to their discovered resources.
	// Keys e.g. "crd:<name>"
	resourcesBySource map[string][]*DiscoveredResource
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
		resourcesBySource: make(map[string][]*DiscoveredResource),
		AddEvents:         addEvents,
		UpdateEvents:      updateEvents,
		DeleteEvents:      deleteEvents,
		CacheCount:        cacheCount,
	}
}

// UpdateSource replaces all resources for a source with new resources.
// If resources is nil, this is a noop.
// If resources is empty, all resources for the source are removed.
func (r *CRDiscoverer) UpdateSource(sourceID string, resources []*DiscoveredResource) {
	if resources == nil {
		return // Skip if nil resources
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	_, existing := r.resourcesBySource[sourceID]

	// Close stop channels for old resources
	if oldResources, ok := r.resourcesBySource[sourceID]; ok {
		for _, old := range oldResources {
			if old.stopChan != nil {
				close(old.stopChan)
			}
		}
	}

	// Create stop channels for new resources
	for _, res := range resources {
		res.stopChan = make(chan struct{})
	}

	if len(resources) == 0 {
		delete(r.resourcesBySource, sourceID) // empty slice signals deletion
	} else {
		r.resourcesBySource[sourceID] = resources
	}

	r.wasUpdated = true

	if !existing {
		r.AddEvents.Inc()
	} else {
		r.UpdateEvents.Inc()
	}

	r.updateCacheCountLocked()
}

// DeleteSource removes all resources for a source and closes their stop channels.
func (r *CRDiscoverer) DeleteSource(sourceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldResources, ok := r.resourcesBySource[sourceID]
	if !ok {
		return
	}

	// Close stop channels
	for _, res := range oldResources {
		if res.stopChan != nil {
			close(res.stopChan)
		}
	}

	delete(r.resourcesBySource, sourceID)
	r.wasUpdated = true

	r.DeleteEvents.Inc()
	r.updateCacheCountLocked()
}

// GetStopChan returns the stop channel for the given GVK.
func (r *CRDiscoverer) GetStopChan(gvk schema.GroupVersionKind) (chan struct{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, resources := range r.resourcesBySource {
		for _, res := range resources {
			if res.GroupVersionKind == gvk {
				return res.stopChan, true
			}
		}
	}
	return nil, false
}

// Resolve resolves a GVK pattern to matching DiscoveredResources.
// Group is required and cannot be a wildcard.
// Supports "*" for Version and/or Kind.
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
	for _, resources := range r.resourcesBySource {
		for _, res := range resources {
			if res.Group != g {
				continue
			}
			if hasVersion && res.Version != v {
				continue
			}
			if hasKind && res.Kind != k {
				continue
			}
			results = append(results, DiscoveredResource{
				GroupVersionKind: res.GroupVersionKind,
				Plural:           res.Plural,
			})
			// exit if exact match
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
	for _, resources := range r.resourcesBySource {
		count += len(resources)
	}
	r.CacheCount.Set(float64(count))
}
