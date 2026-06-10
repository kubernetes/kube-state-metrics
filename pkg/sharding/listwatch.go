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
	"math/bits"
	"sync"

	jump "github.com/dgryski/go-jump"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apisharding "k8s.io/apimachinery/pkg/sharding"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type shardedListWatch struct {
	sharding *sharding
	lw       cache.ListerWatcher
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
	items, err := meta.ExtractList(list)
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

	return watch.Filter(w, func(in watch.Event) (out watch.Event, keep bool) {
		// Bookmarks are stream control events. They carry a resource version but
		// no UID, so filtering them would route every bookmark to a single shard.
		if in.Type == watch.Bookmark {
			return in, true
		}

		a, err := meta.Accessor(in.Object)
		if err != nil {
			// TODO(brancz): needs logging
			return in, true
		}

		return in, s.sharding.keep(a)
	}), nil
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

// shardUIDFieldPath is the CEL-style field path hashed by shardRange selectors.
const shardUIDFieldPath = "object.metadata.uid"

// NewServerSideShardedListWatch returns a cache.ListerWatcher that asks the
// apiserver to filter list and watch responses down to the hash range owned by
// this shard (KEP-5866, `ShardedListAndWatch` feature gate, alpha since
// Kubernetes 1.36), so each shard only receives and decodes its own subset of
// objects. Because the apiserver silently ignores the shard selector when the
// feature is unavailable, every response is verified: lists fall back to
// client-side filtering when the server did not echo shard info, and watch
// events are always re-checked against the same selector (a no-op when the
// server already filtered them).
//
// Note that the hash space is partitioned into contiguous ranges, which
// assigns objects to shards differently than the jump hash used by
// NewShardedListWatch.
//
// In the case of no sharding needed, it returns the provided cache.ListerWatcher.
func NewServerSideShardedListWatch(shard int32, totalShards int, lw cache.ListerWatcher) cache.ListerWatcher {
	// This is an "optimization" as this configuration means no sharding is to
	// be performed.
	if shard == 0 && totalShards == 1 {
		return lw
	}

	selector := apisharding.NewSelector(apisharding.ShardRangeRequirement{
		Key:   shardUIDFieldPath,
		Start: shardRangeBoundary(int(shard), totalShards),
		End:   shardRangeBoundary(int(shard)+1, totalShards),
	})

	return &serverSideShardedListWatch{selector: selector, lw: lw}
}

// shardRangeBoundary returns floor(i * 2^64 / totalShards) as a 0x-prefixed
// hex string, i.e. the i-th boundary when partitioning the 64-bit FNV-1a hash
// space into totalShards contiguous ranges. For i == totalShards the result is
// 2^64, the exclusive upper bound of the hash space.
func shardRangeBoundary(i, totalShards int) string {
	if i >= totalShards {
		return "0x10000000000000000"
	}
	quotient, _ := bits.Div64(uint64(i), 0, uint64(totalShards))
	return fmt.Sprintf("0x%016x", quotient)
}

type serverSideShardedListWatch struct {
	selector       apisharding.Selector
	lw             cache.ListerWatcher
	fallbackLogged sync.Once
}

func (s *serverSideShardedListWatch) List(options metav1.ListOptions) (runtime.Object, error) {
	options.ShardSelector = s.selector.String()
	list, err := s.lw.List(options)
	if err != nil {
		return nil, err
	}
	listMeta, err := meta.ListAccessor(list)
	if err != nil {
		return nil, err
	}
	if sharded, ok := listMeta.(metav1.ShardedListInterface); ok && sharded.GetShardInfo() != nil {
		// The apiserver applied the shard selector and echoed it back, the
		// list already only contains this shard's objects.
		return list, nil
	}

	s.logFallback()
	items, err := meta.ExtractList(list)
	if err != nil {
		return nil, err
	}
	kept := make([]runtime.Object, 0, len(items))
	for _, item := range items {
		matched, err := s.selector.Matches(item)
		if err != nil {
			return nil, err
		}
		if matched {
			kept = append(kept, item)
		}
	}
	if err := meta.SetList(list, kept); err != nil {
		return nil, err
	}

	return list, nil
}

func (s *serverSideShardedListWatch) Watch(options metav1.ListOptions) (watch.Interface, error) {
	options.ShardSelector = s.selector.String()
	w, err := s.lw.Watch(options)
	if err != nil {
		return nil, err
	}

	// Watch responses carry no equivalent of the list's shard info, so events
	// are always re-checked client-side. When the apiserver applied the shard
	// selector every event matches and this filter is a no-op.
	return watch.Filter(w, func(in watch.Event) (out watch.Event, keep bool) {
		// Bookmarks are stream control events. They carry a resource version but
		// no UID, so filtering them would route every bookmark to a single shard.
		if in.Type == watch.Bookmark {
			return in, true
		}

		matched, err := s.selector.Matches(in.Object)
		if err != nil {
			// Objects without metadata, e.g. the *metav1.Status carried by
			// error events, must reach the reflector to be handled there.
			return in, true
		}

		return in, matched
	}), nil
}

// IsWatchListSemanticsUnSupported delegates to the underlying ListerWatcher if it implements this interface.
func (s *serverSideShardedListWatch) IsWatchListSemanticsUnSupported() bool {
	type unsupported interface {
		IsWatchListSemanticsUnSupported() bool
	}
	if u, ok := s.lw.(unsupported); ok {
		return u.IsWatchListSemanticsUnSupported()
	}
	return false
}

func (s *serverSideShardedListWatch) logFallback() {
	s.fallbackLogged.Do(func() {
		klog.InfoS("The apiserver did not apply the shard selector, falling back to client-side shard filtering. Server-side sharding requires Kubernetes 1.36+ with the ShardedListAndWatch feature gate enabled on the apiserver", "shardSelector", s.selector.String())
	})
}
