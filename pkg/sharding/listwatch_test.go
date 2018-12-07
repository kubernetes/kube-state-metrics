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

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	if !(s1.keep(cm) || s2.keep(cm)) {
		t.Fatal("One shard must pick up the object.")
	}

	if !s1.keep(cm) {
		t.Fatal("Shard one should pick up the object.")
	}

	if s2.keep(cm) {
		t.Fatal("Shard two should not pick up the object.")
	}
}
