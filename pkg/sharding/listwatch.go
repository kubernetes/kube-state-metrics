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
	"hash/fnv"

	jump "github.com/dgryski/go-jump"
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
		a, err := meta.Accessor(in.Object)
		if err != nil {
			// TODO(brancz): needs logging
			return in, true
		}

		return in, s.sharding.keep(a)
	}), nil
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
