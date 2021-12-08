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

package namespace_list

// NamespaceList represents a list of namespaces by name and labels.
type NamespaceList map[string]map[string]string

func (n *NamespaceList) String() string {
	// TODO: implement the receiver function
	return ""
}

func (n *NamespaceList) Set(value string) error {
	// TODO: implement the receiver function
	return nil
}

func (n *NamespaceList) Type() string {
	return "string"
}