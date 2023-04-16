package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

const (
	CustomResourceMonitorKind = "CustomResourceMonitor"
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

// PodMonitorList is a list of PodMonitors.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CustomResourceMonitorList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of CustomResourceMonitor
	Items []*CustomResourceMonitor `json:"items"`
}
