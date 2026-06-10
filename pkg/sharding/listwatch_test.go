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
	"strconv"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestShardedListWatchFiltersOnlyResourceStateEvents(t *testing.T) {
	obj := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap1",
			Namespace: "ns1",
			UID:       types.UID("test_uid"),
		},
	}

	tests := []struct {
		name        string
		eventType   watch.EventType
		shouldShard bool
	}{
		{name: "added", eventType: watch.Added, shouldShard: true},
		{name: "modified", eventType: watch.Modified, shouldShard: true},
		{name: "deleted", eventType: watch.Deleted, shouldShard: true},
		{name: "bookmark", eventType: watch.Bookmark},
		{name: "error", eventType: watch.Error},
		{name: "unknown", eventType: watch.EventType("UNKNOWN")},
	}

	for _, shardIndex := range []int32{0, 1} {
		shardedListWatch := &shardedListWatch{
			sharding: &sharding{
				shard:       shardIndex,
				totalShards: 2,
			},
		}

		for _, test := range tests {
			t.Run(test.name+"/shard-"+strconv.Itoa(int(shardIndex)), func(t *testing.T) {
				// Use a metadata-bearing payload for every event so control events
				// pass because of their type, not because their usual payload lacks a UID.
				in := watch.Event{Type: test.eventType, Object: obj}
				out, gotKeep := shardedListWatch.filterWatchEvent(in)
				wantKeep := !test.shouldShard || shardedListWatch.sharding.keep(obj)

				if gotKeep != wantKeep {
					t.Fatalf("got keep %t, want %t", gotKeep, wantKeep)
				}
				if out.Type != in.Type || out.Object != in.Object {
					t.Fatalf("filter changed event from %#v to %#v", in, out)
				}
			})
		}
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
