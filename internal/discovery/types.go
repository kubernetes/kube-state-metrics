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

type groupVersionKindPlural struct {
	schema.GroupVersionKind
	Plural string
}

func (g groupVersionKindPlural) String() string {
	return fmt.Sprintf("%s/%s, Kind=%s, Plural=%s", g.Group, g.Version, g.Kind, g.Plural)
}

type kindPlural struct {
	Kind   string
	Plural string
}

// CRDiscoverer provides a cache of the collected GVKs, along with helper utilities.
type CRDiscoverer struct {
	// CRDsAddEventsCounter tracks the number of times that the CRD informer triggered the "add" event.
	CRDsAddEventsCounter prometheus.Counter
	// CRDsUpdateEventsCounter tracks the number of times that the CRD informer triggered the "update" event.
	CRDsUpdateEventsCounter prometheus.Counter
	// CRDsDeleteEventsCounter tracks the number of times that the CRD informer triggered the "remove" event.
	CRDsDeleteEventsCounter prometheus.Counter
	// CRDsCacheCountGauge tracks the net amount of CRDs affecting the cache at this point.
	CRDsCacheCountGauge prometheus.Gauge
	// Map is a cache of the collected GVKs.
	Map map[string]map[string][]kindPlural
	// GVKToReflectorStopChanMap is a map of GVKs to channels that can be used to stop their corresponding reflector.
	GVKToReflectorStopChanMap map[string]chan struct{}
	// warnedMissingGVKs tracks fully-specified GVKs that were already logged as
	// absent from the cache. ResolveGVKToGVKPs runs on every discovery tick, so
	// this avoids re-logging the same missing CRD until it is resolved.
	warnedMissingGVKs map[string]struct{}
	// m is a mutex to protect the cache.
	m sync.RWMutex
	// ShouldUpdate is a flag that indicates whether the cache was updated.
	WasUpdated bool
}

// SafeRead executes the given function while holding a read lock.
func (r *CRDiscoverer) SafeRead(f func()) {
	r.m.RLock()
	defer r.m.RUnlock()
	f()
}

// SafeWrite executes the given function while holding a write lock.
func (r *CRDiscoverer) SafeWrite(f func()) {
	r.m.Lock()
	defer r.m.Unlock()
	f()
}

// markMissingGVKWarned records that the given GVK was found missing and reports
// whether this is the first time, so the caller logs only once per missing
// episode. The cache mutex protects warnedMissingGVKs.
func (r *CRDiscoverer) markMissingGVKWarned(gvk schema.GroupVersionKind) bool {
	firstTime := false
	r.SafeWrite(func() {
		if r.warnedMissingGVKs == nil {
			r.warnedMissingGVKs = map[string]struct{}{}
		}
		key := gvk.String()
		if _, ok := r.warnedMissingGVKs[key]; !ok {
			r.warnedMissingGVKs[key] = struct{}{}
			firstTime = true
		}
	})
	return firstTime
}

// clearMissingGVKWarning forgets a previously recorded "missing" warning for the
// given GVK so that a future disappearance is logged again.
func (r *CRDiscoverer) clearMissingGVKWarning(gvk schema.GroupVersionKind) {
	r.SafeWrite(func() {
		delete(r.warnedMissingGVKs, gvk.String())
	})
}

// AppendToMap appends the given GVKs to the cache.
func (r *CRDiscoverer) AppendToMap(gvkps ...groupVersionKindPlural) {
	if r.Map == nil {
		r.Map = map[string]map[string][]kindPlural{}
	}
	if r.GVKToReflectorStopChanMap == nil {
		r.GVKToReflectorStopChanMap = map[string]chan struct{}{}
	}
	for _, gvkp := range gvkps {
		if _, ok := r.Map[gvkp.Group]; !ok {
			r.Map[gvkp.Group] = map[string][]kindPlural{}
		}
		if _, ok := r.Map[gvkp.Group][gvkp.Version]; !ok {
			r.Map[gvkp.Group][gvkp.Version] = []kindPlural{}
		}
		alreadyExists := false
		for _, existing := range r.Map[gvkp.Group][gvkp.Version] {
			if existing.Kind == gvkp.Kind {
				alreadyExists = true
				break
			}
		}
		if !alreadyExists {
			r.Map[gvkp.Group][gvkp.Version] = append(r.Map[gvkp.Group][gvkp.Version], kindPlural{Kind: gvkp.Kind, Plural: gvkp.Plural})
		}
		if _, exists := r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()]; !exists {
			r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()] = make(chan struct{})
		}
	}
}

// GetStopChanForGVK returns the stop channel for the given GVK under the read lock.
func (r *CRDiscoverer) GetStopChanForGVK(gvk string) chan struct{} {
	var ch chan struct{}
	r.SafeRead(func() {
		ch = r.GVKToReflectorStopChanMap[gvk]
	})
	return ch
}

// RemoveFromMap removes the given GVKs from the cache.
func (r *CRDiscoverer) RemoveFromMap(gvkps ...groupVersionKindPlural) {
	for _, gvkp := range gvkps {
		if _, ok := r.Map[gvkp.Group]; !ok {
			continue
		}
		if _, ok := r.Map[gvkp.Group][gvkp.Version]; !ok {
			continue
		}
		for i, el := range r.Map[gvkp.Group][gvkp.Version] {
			if el.Kind == gvkp.Kind {
				if _, ok := r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()]; ok {
					close(r.GVKToReflectorStopChanMap[gvkp.GroupVersionKind.String()])
					delete(r.GVKToReflectorStopChanMap, gvkp.GroupVersionKind.String())
				}
				if len(r.Map[gvkp.Group][gvkp.Version]) == 1 {
					delete(r.Map[gvkp.Group], gvkp.Version)
					if len(r.Map[gvkp.Group]) == 0 {
						delete(r.Map, gvkp.Group)
					}
					break
				}
				r.Map[gvkp.Group][gvkp.Version] = append(r.Map[gvkp.Group][gvkp.Version][:i], r.Map[gvkp.Group][gvkp.Version][i+1:]...)
				break
			}
		}
	}
}
