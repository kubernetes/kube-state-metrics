/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package namespacelist

import (
	"fmt"
	"sort"
	"strings"
)

// NamespaceList represents a list of namespaces by name and labels.
type NamespaceList map[string]map[string]string

// String returns the map of namespaces and labels as a string
func (n *NamespaceList) String() string {
	namespaces := make([]string, 0, len(*n))
	for namespace, labels := range *n {
		if len(labels) == 0 {
			namespaces = append(namespaces, namespace)
		} else {
			concatenatedLabelValues := make([]string, 0, len(labels))
			for label, value := range labels {
				concatenatedLabelValues = append(concatenatedLabelValues, fmt.Sprintf("%v=%v", label, value))
			}
			sort.Strings(concatenatedLabelValues)
			namespaces = append(namespaces, fmt.Sprintf("%v=[%v]", namespace, strings.Join(concatenatedLabelValues, ",")))
		}
	}
	sort.Strings(namespaces)
	return strings.Join(namespaces, ",")
}

// Set converts a comma-separated string of namespaces and labels into a map.
func (n *NamespaceList) Set(value string) error {
	// TODO: implement the receiver function
	return nil
}

// Type returns a descriptive string about the NamespaceList type.
func (n *NamespaceList) Type() string {
	return "string"
}