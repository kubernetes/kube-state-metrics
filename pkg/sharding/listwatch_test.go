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
	"k8s.io/apimachinery/pkg/runtime"
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

func TestShardedListWatchListDoesNotRetainSourceItemsArray(t *testing.T) {
	const (
		totalShards     = 2
		resourceVersion = "42"
	)

	source := &v1.ConfigMapList{
		ListMeta: metav1.ListMeta{ResourceVersion: resourceVersion},
		Items: []v1.ConfigMap{
			configMapForShard(0, totalShards),
			configMapForShard(1, totalShards),
		},
	}
	lw := &cache.ListWatch{
		ListFunc: func(metav1.ListOptions) (runtime.Object, error) {
			return source, nil
		},
	}
	shardedLister := cache.ToListerWithContext(NewShardedListWatch(0, totalShards, lw))

	result, err := shardedLister.ListWithContext(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list shard: %v", err)
	}
	list, ok := result.(*metav1.List)
	if !ok {
		t.Fatalf("got result type %T, want *metav1.List", result)
	}
	if list.ResourceVersion != resourceVersion {
		t.Errorf("got resource version %q, want %q", list.ResourceVersion, resourceVersion)
	}
	if len(list.Items) != 1 {
		t.Fatalf("got %d retained items, want 1", len(list.Items))
	}
	retained, ok := list.Items[0].Object.(*v1.ConfigMap)
	if !ok {
		t.Fatalf("got retained item type %T, want *v1.ConfigMap", list.Items[0].Object)
	}
	if retained.UID != source.Items[0].UID {
		t.Errorf("got retained item UID %q, want %q", retained.UID, source.Items[0].UID)
	}
	for i := range source.Items {
		if retained == &source.Items[i] {
			t.Fatalf("retained item aliases source Items[%d]", i)
		}
	}
}

func configMapForShard(shard int32, totalShards int) v1.ConfigMap {
	filter := &sharding{shard: shard, totalShards: totalShards}
	for i := 0; ; i++ {
		configMap := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "configmap-" + strconv.Itoa(i),
				UID:  types.UID(strconv.Itoa(i)),
			},
		}
		if filter.keep(&configMap) {
			return configMap
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
