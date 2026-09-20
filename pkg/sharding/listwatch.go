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
	"math"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
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
	options.ShardSelector = s.sharding.selector()
	return s.lw.List(options)
}

func (s *shardedListWatch) Watch(options metav1.ListOptions) (watch.Interface, error) {
	options.ShardSelector = s.sharding.selector()
	return s.lw.Watch(options)
}

type sharding struct {
	shard       int32
	totalShards int
}

func (s *sharding) selector() string {
	step := uint64(math.MaxUint64)/uint64(s.totalShards) + 1
	start := step * uint64(s.shard)
	// end overflows uint64
	if s.shard+1 == int32(s.totalShards) {
		return fmt.Sprintf("shardRange(object.metadata.uid, '0x%016x', '0x10000000000000000')", start)
	}
	end := start + step
	return fmt.Sprintf("shardRange(object.metadata.uid, '0x%016x', '0x%16x')", start, end)
}
