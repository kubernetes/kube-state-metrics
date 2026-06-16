/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

package sharding

import (
	"fmt"
	"hash/fnv"
	"sync"

	jump "github.com/dgryski/go-jump"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type shardedListWatch struct {
	sharding *sharding
	lw       cache.ListerWatcher
}

// shardedWatch filters events from an upstream watch while allowing Stop to
// interrupt both receiving and forwarding. This is intentionally local to the
// sharding implementation: once its consumer stops, an in-flight event may be
// discarded so the forwarding goroutine can terminate promptly. This avoids
// the blocked-send leak in watch.Filter documented in:
// https://github.com/kubernetes/kubernetes/issues/113254.
type shardedWatch struct {
	incoming watch.Interface
	result   chan watch.Event
	filter   watch.FilterFunc
	stopCh   chan struct{}
	doneCh   chan struct{}
	stopOnce sync.Once
}

var _ watch.Interface = &shardedWatch{}

func newShardedWatch(incoming watch.Interface, filter watch.FilterFunc) *shardedWatch {
	w := &shardedWatch{
		incoming: incoming,
		result:   make(chan watch.Event),
		filter:   filter,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
	go w.run()
	return w
}

// ResultChan returns the filtered event stream.
func (w *shardedWatch) ResultChan() <-chan watch.Event {
	return w.result
}

// Stop stops the upstream watch and unblocks the forwarding goroutine.
func (w *shardedWatch) Stop() {
	w.stopOnce.Do(func() {
		// Close stopCh first so forwarding can stop independently of upstream
		// watcher shutdown.
		close(w.stopCh)
		w.incoming.Stop()
	})
}

func (w *shardedWatch) run() {
	defer close(w.doneCh)
	defer close(w.result)
	defer w.Stop()

	incoming := w.incoming.ResultChan()
	for {
		select {
		case <-w.stopCh:
			return
		case event, ok := <-incoming:
			if !ok {
				return
			}

			filtered, keep := w.filter(event)
			if !keep {
				continue
			}

			select {
			case <-w.stopCh:
				return
			case w.result <- filtered:
			}
		}
	}
}

// NewShardedListWatch returns a new shardedListWatch via the cache.ListerWatcher interface.
// In the case of no sharding needed, it returns the provided cache.ListerWatcher
func NewShardedListWatch(shard int32, totalShards int, lw cache.ListerWatcher) cache.ListerWatcher {
	// This is an "optimization" as this configuration means no sharding is to
	// be performed.
	if shard == 0 && totalShards == 1 {
		return lw
	}

	return &shardedListWatch{sharding: &sharding{shard: shard, totalShards: totalShards}, lw: lw}
}

func (s *shardedListWatch) List(options metav1.ListOptions) (runtime.Object, error) {
	list, err := s.lw.List(options)
	if err != nil {
		return nil, err
	}
	// Retained shard items outlive the source list. Allocate non-pointer items so
	// they do not keep the source list's entire Items backing array reachable.
	items, err := meta.ExtractListWithAlloc(list)
	if err != nil {
		return nil, err
	}
	metaObj, err := meta.ListAccessor(list)
	if err != nil {
		return nil, err
	}
	res := &metav1.List{
		Items: []runtime.RawExtension{},
	}
	for _, item := range items {
		a, err := meta.Accessor(item)
		if err != nil {
			return nil, err
		}
		if s.sharding.keep(a) {
			res.Items = append(res.Items, runtime.RawExtension{Object: item})
		}
	}
	res.ResourceVersion = metaObj.GetResourceVersion()

	return res, nil
}

func (s *shardedListWatch) Watch(options metav1.ListOptions) (watch.Interface, error) {
	w, err := s.lw.Watch(options)
	if err != nil {
		return nil, err
	}

	return newShardedWatch(w, s.filterWatchEvent), nil
}

// filterWatchEvent shards resource state changes, passes control events through,
// and rejects unknown events so new mutation types cannot bypass sharding.
func (s *shardedListWatch) filterWatchEvent(in watch.Event) (out watch.Event, keep bool) {
	switch in.Type {
	case watch.Added, watch.Modified, watch.Deleted:
		a, err := meta.Accessor(in.Object)
		if err != nil {
			return internalErrorEvent(fmt.Errorf("sharded list watch failed to access object metadata for event type %q: %w", in.Type, err)), true
		}

		return in, s.sharding.keep(a)
	case watch.Bookmark, watch.Error:
		return in, true
	default:
		return internalErrorEvent(fmt.Errorf("sharded list watch failed to recognize event type %q", in.Type)), true
	}
}

func internalErrorEvent(err error) watch.Event {
	return watch.Event{
		Type:   watch.Error,
		Object: &apierrors.NewInternalError(err).ErrStatus,
	}
}

// IsWatchListSemanticsUnSupported delegates to the underlying ListerWatcher if it implements this interface.
func (s *shardedListWatch) IsWatchListSemanticsUnSupported() bool {
	type unsupported interface {
		IsWatchListSemanticsUnSupported() bool
	}
	if u, ok := s.lw.(unsupported); ok {
		return u.IsWatchListSemanticsUnSupported()
	}
	return false
}

type sharding struct {
	shard       int32
	totalShards int
}

func (s *sharding) keep(o metav1.Object) bool {
	h := fnv.New64a()
	h.Write([]byte(o.GetUID()))
	return jump.Hash(h.Sum64(), s.totalShards) == s.shard
}
