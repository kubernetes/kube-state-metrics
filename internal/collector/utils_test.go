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

package collector

import (
	"fmt"
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestIsHugePageSizeFromResourceName(t *testing.T) {
	testCases := []struct {
		resourceName v1.ResourceName
		expectVal    bool
	}{
		{
			resourceName: "pod.alpha.kubernetes.io/opaque-int-resource-foo",
			expectVal:    false,
		},
		{
			resourceName: "hugepages-100m",
			expectVal:    true,
		},
		{
			resourceName: "",
			expectVal:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("resourceName input=%s, expected value=%v", tc.resourceName, tc.expectVal), func(t *testing.T) {
			v := isHugePageResourceName(tc.resourceName)
			if v != tc.expectVal {
				t.Errorf("Got %v but expected %v", v, tc.expectVal)
			}
		})
	}
}

func TestIsAttachableVolumeResourceName(t *testing.T) {
	testCases := []struct {
		resourceName v1.ResourceName
		expectVal    bool
	}{
		{
			resourceName: "pod.alpha.kubernetes.io/opaque-int-resource-foo",
			expectVal:    false,
		},
		{
			resourceName: "attachable-volumes-100m",
			expectVal:    true,
		},
		{
			resourceName: "",
			expectVal:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("resourceName input=%s, expected value=%v", tc.resourceName, tc.expectVal), func(t *testing.T) {
			v := isAttachableVolumeResourceName(tc.resourceName)
			if v != tc.expectVal {
				t.Errorf("Got %v but expected %v", v, tc.expectVal)
			}
		})
	}
}

func TestIsExtendedResourceName(t *testing.T) {
	testCases := []struct {
		resourceName v1.ResourceName
		expectVal    bool
	}{
		{
			resourceName: "pod.alpha.kubernetes.io/opaque-int-resource-foo",
			expectVal:    false,
		},
		{
			resourceName: "kubernetes.io/resource-foo",
			expectVal:    false,
		},
		{
			resourceName: "foo",
			expectVal:    false,
		},
		{
			resourceName: "a/b",
			expectVal:    true,
		},
		{
			resourceName: "requests.foobar",
			expectVal:    false,
		},
		{
			resourceName: "c/d/",
			expectVal:    false,
		},
		{
			resourceName: "",
			expectVal:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("resourceName input=%s, expected value=%v", tc.resourceName, tc.expectVal), func(t *testing.T) {
			v := isExtendedResourceName(tc.resourceName)
			if v != tc.expectVal {
				t.Errorf("Got %v but expected %v", v, tc.expectVal)
			}
		})
	}
}

func TestKubeLabelsToPrometheusLabels(t *testing.T) {
	t.Run("prometheus labels when kube labels has multiple items", func(t *testing.T) {

		kubeLabels := map[string]string{
			"app1":   "normal",
			"-app2":  "starts_with_hyphen",
			"0_app3": "starts_with_digit",
			"":       "empty",
			"$app4":  "special_char",
			"_app5":  "starts_with_underscore",
		}

		expectedPrometheusLabelKeys := []string{
			"label_app1",
			"label_-app2",
			"label_0_app3",
			"label_",
			"label__app4",
			"label__app5",
		}
		expectedPrometheusLabelValues := []string{
			"normal",
			"starts_with_hyphen",
			"starts_with_digit",
			"empty",
			"special_char",
			"starts_with_underscore",
		}

		labelKeys, labelValues := kubeLabelsToPrometheusLabels(kubeLabels)
		if len(labelKeys) != len(expectedPrometheusLabelKeys) {
			t.Errorf("Got Prometheus label keys with len %d but expected %d", len(labelKeys), len(expectedPrometheusLabelKeys))
		}

		if len(labelValues) != len(expectedPrometheusLabelValues) {
			t.Errorf("Got Prometheus label values with len %d but expected %d", len(labelValues), len(expectedPrometheusLabelValues))
		}

		for i := range expectedPrometheusLabelKeys {
			if !(expectedPrometheusLabelKeys[i] == labelKeys[i] && expectedPrometheusLabelValues[i] == labelValues[i]) {
				t.Errorf("Got Prometheus label %q: %q but expected %q: %q", labelKeys[i], labelValues[i], expectedPrometheusLabelKeys[i], expectedPrometheusLabelValues[i])
			}
		}
	})

	t.Run("prometheus labels when kube labels is empty", func(t *testing.T) {

		kubeLabels := map[string]string{}

		labelKeys, labelValues := kubeLabelsToPrometheusLabels(kubeLabels)
		if len(labelKeys) != 0 || len(labelValues) != 0 {
			t.Errorf("Got Prometheus label keys with len %d and values with len %d but expected len 0", len(labelKeys), len(labelValues))
		}
	})
}

func TestKubeAnnotationsToPrometheusAnnotations(t *testing.T) {

	t.Run("prometheus annotations when kube annotations has multiple items", func(t *testing.T) {

		kubeAnnotations := map[string]string{
			"app1":   "normal",
			"-app2":  "starts_with_hyphen",
			"0_app3": "starts_with_digit",
			"":       "empty",
			"$app4":  "special_char",
			"_app5":  "starts_with_underscore",
		}

		expectedPrometheusAnnotationKeys := []string{
			"annotation_app1",
			"annotation_-app2",
			"annotation_0_app3",
			"annotation_",
			"annotation__app4",
			"annotation__app5",
		}
		expectedPrometheusAnnotationValues := []string{
			"normal",
			"starts_with_hyphen",
			"starts_with_digit",
			"empty",
			"special_char",
			"starts_with_underscore",
		}

		annotationKeys, annotationValues := kubeAnnotationsToPrometheusAnnotations(kubeAnnotations)
		if len(annotationKeys) != len(expectedPrometheusAnnotationKeys) {
			t.Errorf("Got Prometheus annotation keys with len %d but expected %d", len(annotationKeys), len(expectedPrometheusAnnotationKeys))
		}

		if len(annotationValues) != len(expectedPrometheusAnnotationValues) {
			t.Errorf("Got Prometheus annotation values with len %d but expected %d", len(annotationValues), len(expectedPrometheusAnnotationValues))
		}

		for i := range expectedPrometheusAnnotationKeys {
			if !(expectedPrometheusAnnotationKeys[i] == annotationKeys[i] && expectedPrometheusAnnotationValues[i] == annotationValues[i]) {
				t.Errorf("Got Prometheus annotation %q: %q but expected %q: %q", annotationKeys[i], annotationValues[i], expectedPrometheusAnnotationKeys[i], expectedPrometheusAnnotationValues[i])
			}
		}
	})

	t.Run("prometheus annotations when kube annotations is empty", func(t *testing.T) {

		kubeAnnotations := map[string]string{}

		annotationKeys, annotationValues := kubeAnnotationsToPrometheusAnnotations(kubeAnnotations)
		if len(annotationKeys) != 0 || len(annotationValues) != 0 {
			t.Errorf("Got Prometheus annotation keys with len %d and values with len %d but expected len 0", len(annotationKeys), len(annotationValues))
		}
	})
}
