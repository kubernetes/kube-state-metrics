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

package discovery

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type crdExtractor struct{}

// SourceID returns a unique identifier for the CRD.
func (e *crdExtractor) SourceID(obj interface{}) string {
	u := obj.(*unstructured.Unstructured)
	return "crd:" + u.GetName()
}

// ExtractGVKs extracts GVK information from a CRD object.
func (e *crdExtractor) ExtractGVKs(obj interface{}) []*DiscoveredResource {
	objSpec := obj.(*unstructured.Unstructured).Object["spec"].(map[string]interface{})
	var resources []*DiscoveredResource
	for _, version := range objSpec["versions"].([]interface{}) {
		g := objSpec["group"].(string)
		v := version.(map[string]interface{})["name"].(string)
		k := objSpec["names"].(map[string]interface{})["kind"].(string)
		p := objSpec["names"].(map[string]interface{})["plural"].(string)
		resources = append(resources, &DiscoveredResource{
			GroupVersionKind: schema.GroupVersionKind{
				Group:   g,
				Version: v,
				Kind:    k,
			},
			Plural: p,
		})
	}
	return resources
}
