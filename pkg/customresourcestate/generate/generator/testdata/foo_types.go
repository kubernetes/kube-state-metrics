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

//go:generate sh -c "go run ../../../../../ generate ./... > foo-config.yaml"

// +groupName=bar.example.com
package foo

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FooSpec is the spec of Foo.
type FooSpec struct {
	// This tests that defaulted fields are stripped for v1beta1,
	// but not for v1
	DefaultedString string `json:"defaultedString"`
}

// FooStatus is the status of Foo.
type FooStatus struct {
	// +Metrics:stateset:name="status_condition",help="The condition of a foo.",labelName="status",JSONPath=".status",list={"True","False","Unknown"},labelsFromPath={"type":".type"}
	// +Metrics:gauge:name="status_condition_last_transition_time",help="The condition last transition time of a foo.",valueFrom=.lastTransitionTime,labelsFromPath={"type":".type","status":".status"}
	Conditions Condition `json:"conditions,omitempty"`
}

// Foo is a test object.
// +Metrics:gvk:namePrefix="foo"
// +Metrics:labelFromPath:name="name",JSONPath=".metadata.name"
// +Metrics:gauge:name="created",JSONPath=".metadata.creationTimestamp",help="Unix creation timestamp."
// +Metrics:info:name="owner",JSONPath=".metadata.ownerReferences",help="Owner references.",labelsFromPath={owner_is_controller:".controller",owner_kind:".kind",owner_name:".name",owner_uid:".uid"}
// +Metrics:labelFromPath:name="cluster_name",JSONPath=.metadata.labels.cluster\.x-k8s\.io/cluster-name
type Foo struct {
	// TypeMeta comments should NOT appear in the CRD spec
	metav1.TypeMeta `json:",inline"`
	// ObjectMeta comments should NOT appear in the CRD spec
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec comments SHOULD appear in the CRD spec
	Spec FooSpec `json:"spec,omitempty"`
	// Status comments SHOULD appear in the CRD spec
	Status FooStatus `json:"status,omitempty"`
}

// Condition is a test condition.
type Condition struct {
	// Type of condition.
	Type string `json:"type"`
	// Status of condition.
	Status string `json:"status"`
	// LastTransitionTime of condition.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
}
