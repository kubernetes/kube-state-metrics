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
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
)

// crdExtractor extracts DiscoveredResources from CustomResourceDefinition
// objects delivered as *unstructured.Unstructured by a dynamic informer.
type crdExtractor struct{}

// SourceID returns "crd:<name>" for a CRD object, or "" if obj is not a
// *unstructured.Unstructured or has no name.
func (e *crdExtractor) SourceID(obj interface{}) string {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		klog.InfoS("crdExtractor.SourceID: unexpected type, skipping", "type", fmt.Sprintf("%T", obj))
		return ""
	}
	name := u.GetName()
	if name == "" {
		klog.InfoS("crdExtractor.SourceID: CRD has no name, skipping")
		return ""
	}
	return "crd:" + name
}

// ExtractGVKs returns the GVKs declared by a CRD.
// Returns nil if the object cannot be parsed as a CRD; individually malformed
// versions are skipped while valid versions in the same CRD are still returned.
func (e *crdExtractor) ExtractGVKs(obj interface{}) []DiscoveredResource {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		klog.InfoS("crdExtractor.ExtractGVKs: unexpected type, skipping", "type", fmt.Sprintf("%T", obj))
		return nil
	}

	name := u.GetName()

	group, gErr := requiredString(u.Object, "spec", "group")
	kind, kErr := requiredString(u.Object, "spec", "names", "kind")
	plural, pErr := requiredString(u.Object, "spec", "names", "plural")
	versions, found, vErr := unstructured.NestedSlice(u.Object, "spec", "versions")
	if vErr != nil {
		vErr = fmt.Errorf("field spec.versions: %w", vErr)
	} else if !found {
		vErr = fmt.Errorf("field spec.versions: missing")
	}

	if err := errors.Join(gErr, kErr, pErr, vErr); err != nil {
		klog.InfoS("crdExtractor.ExtractGVKs: malformed CRD, skipping", "name", name, "err", err)
		return nil
	}

	var resources []DiscoveredResource
	for i, raw := range versions {
		version, ok := raw.(map[string]interface{})
		if !ok {
			klog.InfoS("crdExtractor.ExtractGVKs: skipping malformed version entry", "name", name, "index", i)
			continue
		}
		v, found, err := unstructured.NestedString(version, "name")
		if err != nil || !found || v == "" {
			klog.InfoS("crdExtractor.ExtractGVKs: skipping version with missing or empty name", "name", name, "index", i)
			continue
		}
		resources = append(resources, DiscoveredResource{
			GroupVersionKind: schema.GroupVersionKind{
				Group:   group,
				Version: v,
				Kind:    kind,
			},
			Plural: plural,
		})
	}
	return resources
}

// requiredString reads a non-empty string at the given path. Returns an
// error if the field is missing, of the wrong type, or empty.
func requiredString(obj map[string]interface{}, path ...string) (string, error) {
	val, found, err := unstructured.NestedString(obj, path...)
	if err != nil {
		return "", fmt.Errorf("field %s: %w", strings.Join(path, "."), err)
	}
	if !found {
		return "", fmt.Errorf("field %s: missing", strings.Join(path, "."))
	}
	if val == "" {
		return "", fmt.Errorf("field %s: empty", strings.Join(path, "."))
	}
	return val, nil
}
