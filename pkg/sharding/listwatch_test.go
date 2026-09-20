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
	"testing"
)

func TestSharding(t *testing.T) {
	s1 := &sharding{
		shard:       0,
		totalShards: 2,
	}
	s2 := &sharding{
		shard:       1,
		totalShards: 2,
	}

	if s1.selector() != "shardRange(object.metadata.uid, '0x0000000000000000', '0x8000000000000000')" {
		t.Fatal("Shard selector of shard one must equal to shardRange(object.metadata.uid, '0x0000000000000000', '0x8000000000000000').")
	}
	if s2.selector() != "shardRange(object.metadata.uid, '0x8000000000000000', '0x10000000000000000')" {
		t.Fatal("Shard selector of shard one must equal to shardRange(object.metadata.uid, '0x8000000000000000', '0x10000000000000000').")
	}
}
