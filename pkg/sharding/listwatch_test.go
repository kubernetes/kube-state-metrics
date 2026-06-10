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
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apisharding "k8s.io/apimachinery/pkg/sharding"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func TestSharding(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap1",
			Namespace: "ns1",
			UID:       types.UID("test_uid"),
		},
	}

	s1 := &sharding{
		shard:       0,
		totalShards: 2,
	}
	s2 := &sharding{
		shard:       1,
		totalShards: 2,
	}

	if !s1.keep(cm) && !s2.keep(cm) {
		t.Fatal("One shard must pick up the object.")
	}

	if !s1.keep(cm) {
		t.Fatal("Shard one should pick up the object.")
	}

	if s2.keep(cm) {
		t.Fatal("Shard two should not pick up the object.")
	}
}

func TestShardedListWatchPassesInitialEventsEndBookmarkToEveryShard(t *testing.T) {
	bookmark := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "1",
			Annotations: map[string]string{
				metav1.InitialEventsAnnotationKey: "true",
			},
		},
	}

	source := watch.NewBroadcaster(1, watch.WaitIfChannelFull)
	t.Cleanup(source.Shutdown)

	var shardedWatches []watch.Interface
	for shard := int32(0); shard < 4; shard++ {
		lw := &cache.ListWatch{
			WatchFunc: func(metav1.ListOptions) (watch.Interface, error) {
				return source.Watch()
			},
		}

		shardedWatch, err := cache.ToWatcherWithContext(NewShardedListWatch(shard, 4, lw)).WatchWithContext(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("failed to create sharded watch: %v", err)
		}
		t.Cleanup(shardedWatch.Stop)
		shardedWatches = append(shardedWatches, shardedWatch)
	}

	if err := source.Action(watch.Bookmark, bookmark); err != nil {
		t.Fatalf("failed to broadcast initial events end bookmark: %v", err)
	}

	for shard, shardedWatch := range shardedWatches {
		t.Run(strconv.Itoa(shard), func(t *testing.T) {
			select {
			case event := <-shardedWatch.ResultChan():
				if event.Type != watch.Bookmark {
					t.Fatalf("got event type %q, want %q", event.Type, watch.Bookmark)
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for initial events end bookmark")
			}
		})
	}
}

func TestShardRangeBoundary(t *testing.T) {
	for _, tc := range []struct {
		i           int
		totalShards int
		want        string
	}{
		{0, 2, "0x0000000000000000"},
		{1, 2, "0x8000000000000000"},
		{2, 2, "0x10000000000000000"},
		{1, 4, "0x4000000000000000"},
		{3, 4, "0xc000000000000000"},
		{1, 3, "0x5555555555555555"},
		{3, 3, "0x10000000000000000"},
	} {
		if got := shardRangeBoundary(tc.i, tc.totalShards); got != tc.want {
			t.Errorf("shardRangeBoundary(%d, %d) = %q, want %q", tc.i, tc.totalShards, got, tc.want)
		}
	}
}

func TestServerSideShardedListWatchSetsShardSelector(t *testing.T) {
	want := "shardRange(object.metadata.uid, '0x0000000000000000', '0x8000000000000000')"

	var listOptions, watchOptions metav1.ListOptions
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			listOptions = options
			return &v1.ConfigMapList{}, nil
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			watchOptions = options
			return watch.NewFake(), nil
		},
	}

	sharded := NewServerSideShardedListWatch(0, 2, lw)
	if _, err := sharded.List(metav1.ListOptions{}); err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if listOptions.ShardSelector != want {
		t.Errorf("list shard selector = %q, want %q", listOptions.ShardSelector, want)
	}

	w, err := sharded.Watch(metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed to watch: %v", err)
	}
	w.Stop()
	if watchOptions.ShardSelector != want {
		t.Errorf("watch shard selector = %q, want %q", watchOptions.ShardSelector, want)
	}
}

func testConfigMaps(n int) []v1.ConfigMap {
	items := make([]v1.ConfigMap, n)
	for i := range items {
		// Sequential UIDs cluster in the FNV-1a hash space because the strings
		// only differ in their final characters, so derive pseudo-random UIDs
		// the way real clusters get random UUIDs.
		uid := apisharding.HashField(fmt.Sprintf("test_uid_%d", i))
		items[i] = v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("configmap%d", i),
				Namespace: "ns1",
				UID:       types.UID(uid),
			},
		}
	}
	return items
}

func TestServerSideShardedListWatchTrustsShardedResponses(t *testing.T) {
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			// A response carrying shard info was already filtered by the
			// apiserver, items must be returned as-is even if they would not
			// match the shard's range.
			return &v1.ConfigMapList{
				ListMeta: metav1.ListMeta{
					ResourceVersion: "10",
					ShardInfo:       &metav1.ShardInfo{Selector: options.ShardSelector},
				},
				Items: testConfigMaps(10),
			}, nil
		},
	}

	for shard := int32(0); shard < 2; shard++ {
		list, err := NewServerSideShardedListWatch(shard, 2, lw).List(metav1.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}
		items, err := meta.ExtractList(list)
		if err != nil {
			t.Fatalf("failed to extract list: %v", err)
		}
		if len(items) != 10 {
			t.Errorf("shard %d: got %d items, want all 10 items of the sharded response", shard, len(items))
		}
	}
}

func TestServerSideShardedListWatchFallsBackToClientSideFiltering(t *testing.T) {
	const totalShards = 4
	lw := &cache.ListWatch{
		ListFunc: func(metav1.ListOptions) (runtime.Object, error) {
			// No shard info, i.e. the apiserver ignored the shard selector.
			return &v1.ConfigMapList{
				ListMeta: metav1.ListMeta{ResourceVersion: "10"},
				Items:    testConfigMaps(100),
			}, nil
		},
	}

	seen := map[types.UID]int{}
	for shard := int32(0); shard < totalShards; shard++ {
		list, err := NewServerSideShardedListWatch(shard, totalShards, lw).List(metav1.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}
		listMeta, err := meta.ListAccessor(list)
		if err != nil {
			t.Fatalf("failed to access list meta: %v", err)
		}
		if listMeta.GetResourceVersion() != "10" {
			t.Errorf("shard %d: resource version = %q, want %q", shard, listMeta.GetResourceVersion(), "10")
		}
		items, err := meta.ExtractList(list)
		if err != nil {
			t.Fatalf("failed to extract list: %v", err)
		}
		if len(items) == 100 {
			t.Errorf("shard %d: kept all 100 items, want a filtered subset", shard)
		}
		for _, item := range items {
			a, err := meta.Accessor(item)
			if err != nil {
				t.Fatalf("failed to access item: %v", err)
			}
			seen[a.GetUID()]++
		}
	}

	if len(seen) != 100 {
		t.Errorf("shards kept %d distinct items in total, want all 100", len(seen))
	}
	for uid, count := range seen {
		if count != 1 {
			t.Errorf("item %q was kept by %d shards, want exactly 1", uid, count)
		}
	}
}

func TestServerSideShardedListWatchFiltersWatchEventsClientSide(t *testing.T) {
	const totalShards = 4
	// The queue must hold all broadcast events per watcher, because the shards
	// are drained sequentially below and a full queue would block the
	// broadcaster before the bookmark reaches the earlier shards.
	source := watch.NewBroadcaster(16, watch.WaitIfChannelFull)
	t.Cleanup(source.Shutdown)

	var shardedWatches []watch.Interface
	for shard := int32(0); shard < totalShards; shard++ {
		lw := &cache.ListWatch{
			WatchFunc: func(metav1.ListOptions) (watch.Interface, error) {
				return source.Watch()
			},
		}

		shardedWatch, err := cache.ToWatcherWithContext(NewServerSideShardedListWatch(shard, totalShards, lw)).WatchWithContext(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Fatalf("failed to create sharded watch: %v", err)
		}
		t.Cleanup(shardedWatch.Stop)
		shardedWatches = append(shardedWatches, shardedWatch)
	}

	configMaps := testConfigMaps(10)
	for i := range configMaps {
		if err := source.Action(watch.Added, &configMaps[i]); err != nil {
			t.Fatalf("failed to broadcast event: %v", err)
		}
	}
	// The bookmark passes every shard's filter and serves as an end-of-stream
	// marker for the assertions below.
	if err := source.Action(watch.Bookmark, &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "10"}}); err != nil {
		t.Fatalf("failed to broadcast bookmark: %v", err)
	}

	seen := map[types.UID]int{}
	for shard, shardedWatch := range shardedWatches {
		for {
			var event watch.Event
			select {
			case event = <-shardedWatch.ResultChan():
			case <-time.After(time.Second):
				t.Fatalf("shard %d: timed out waiting for bookmark", shard)
			}
			if event.Type == watch.Bookmark {
				break
			}
			a, err := meta.Accessor(event.Object)
			if err != nil {
				t.Fatalf("failed to access event object: %v", err)
			}
			seen[a.GetUID()]++
		}
	}

	if len(seen) != len(configMaps) {
		t.Errorf("shards received %d distinct events in total, want %d", len(seen), len(configMaps))
	}
	for uid, count := range seen {
		if count != 1 {
			t.Errorf("event for %q was received by %d shards, want exactly 1", uid, count)
		}
	}
}

func TestServerSideShardedListWatchMatchesServerSideHashing(t *testing.T) {
	// The client-side fallback filter must agree with the hash the apiserver
	// uses for shardRange selectors (64-bit FNV-1a over the UID).
	selector := apisharding.NewSelector(apisharding.ShardRangeRequirement{
		Key:   shardUIDFieldPath,
		Start: shardRangeBoundary(0, 2),
		End:   shardRangeBoundary(1, 2),
	})
	for _, cm := range testConfigMaps(100) {
		matched, err := selector.Matches(&cm)
		if err != nil {
			t.Fatalf("failed to match %q: %v", cm.UID, err)
		}
		want := apisharding.HashField(string(cm.UID)) < "8000000000000000"
		if matched != want {
			t.Errorf("selector match for %q = %v, want %v", cm.UID, matched, want)
		}
	}
}
