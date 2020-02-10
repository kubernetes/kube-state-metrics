/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package options

import (
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MetricSet represents a collection which has a unique set of metrics.
type MetricSet map[string]struct{}

func (ms *MetricSet) String() string {
	s := *ms
	ss := s.asSlice()
	sort.Strings(ss)
	return strings.Join(ss, ",")
}

// Set converts a comma-separated string of metrics into a slice and appends it to the MetricSet.
func (ms *MetricSet) Set(value string) error {
	s := *ms
	metrics := strings.Split(value, ",")
	for _, metric := range metrics {
		metric = strings.TrimSpace(metric)
		if len(metric) != 0 {
			s[metric] = struct{}{}
		}
	}
	return nil
}

// asSlice returns the MetricSet in the form of plain string slice.
func (ms MetricSet) asSlice() []string {
	metrics := make([]string, 0, len(ms))
	for metric := range ms {
		metrics = append(metrics, metric)
	}
	return metrics
}

// Type returns a descriptive string about the MetricSet type.
func (ms *MetricSet) Type() string {
	return "string"
}

// ResourceSet represents a collection which has a unique set of resources.
type ResourceSet map[string]struct{}

func (r *ResourceSet) String() string {
	s := *r
	ss := s.AsSlice()
	sort.Strings(ss)
	return strings.Join(ss, ",")
}

// Set converts a comma-separated string of resources into a slice and appends it to the ResourceSet.
func (r *ResourceSet) Set(value string) error {
	s := *r
	cols := strings.Split(value, ",")
	for _, col := range cols {
		col = strings.TrimSpace(col)
		if len(col) != 0 {
			s[col] = struct{}{}
		}
	}
	return nil
}

// AsSlice returns the Resource in the form of a plain string slice.
func (r ResourceSet) AsSlice() []string {
	cols := make([]string, 0, len(r))
	for col := range r {
		cols = append(cols, col)
	}
	return cols
}

// Type returns a descriptive string about the ResourceSet type.
func (r *ResourceSet) Type() string {
	return "string"
}

// NamespaceList represents a list of namespaces to query from.
type NamespaceList []string

func (n *NamespaceList) String() string {
	return strings.Join(*n, ",")
}

// IsAllNamespaces checks if the Namespace selector is that of `NamespaceAll` which is used for
// selecting or filtering across all namespaces.
func (n *NamespaceList) IsAllNamespaces() bool {
	return len(*n) == 1 && (*n)[0] == metav1.NamespaceAll
}

// Set converts a comma-separated string of namespaces into a slice and appends it to the NamespaceList
func (n *NamespaceList) Set(value string) error {
	splitNamespaces := strings.Split(value, ",")
	for _, ns := range splitNamespaces {
		ns = strings.TrimSpace(ns)
		if len(ns) != 0 {
			*n = append(*n, ns)
		}
	}
	return nil
}

// Type returns a descriptive string about the NamespaceList type.
func (n *NamespaceList) Type() string {
	return "string"
}
