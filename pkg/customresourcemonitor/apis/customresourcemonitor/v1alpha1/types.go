/*
Copyright The Kubernetes Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

const (
	// CustomResourceMonitorKind is Kind of CustomResourceMonitor
	CustomResourceMonitorKind = "CustomResourceMonitor"
	// CustomResourceMonitorName is plural Name of CustomResourceMonitor
	CustomResourceMonitorName = "customresourcemonitors"
)

// CustomResourceMonitor defines monitoring for a set of custom resources.
// +genclient
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CustomResourceMonitor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of desired custom resource selection for target discovery by kube-state-metrics.
	customresourcestate.Metrics `json:",inline"`
}

// CustomResourceMonitorList is a list of CustomResourceMonitors.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CustomResourceMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of CustomResourceMonitor
	Items []*CustomResourceMonitor `json:"items"`
}
