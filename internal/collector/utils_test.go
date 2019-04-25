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
	testCases := []struct {
		kubeLabels   map[string]string
		expectKeys   []string
		expectValues []string
	}{
		{
			kubeLabels: map[string]string{
				"app1": "normal",
			},
			expectKeys:   []string{"label_app1"},
			expectValues: []string{"normal"},
		},
		{
			kubeLabels: map[string]string{
				"0_app3": "starts_with_digit",
			},
			expectKeys:   []string{"label_0_app3"},
			expectValues: []string{"starts_with_digit"},
		},
		{
			kubeLabels: map[string]string{
				"": "empty",
			},
			expectKeys:   []string{"label_"},
			expectValues: []string{"empty"},
		},
		{
			kubeLabels: map[string]string{
				"$app4": "special_char",
			},
			expectKeys:   []string{"label__app4"},
			expectValues: []string{"special_char"},
		},
		{
			kubeLabels: map[string]string{
				"_app5": "starts_with_underscore",
			},
			expectKeys:   []string{"label__app5"},
			expectValues: []string{"starts_with_underscore"},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("kubelabels input=%v , expected prometheus keys=%v, expected prometheus values=%v", tc.kubeLabels, tc.expectKeys, tc.expectValues), func(t *testing.T) {
			labelKeys, labelValues := kubeLabelsToPrometheusLabels(tc.kubeLabels)
			if len(labelKeys) != len(tc.expectKeys) {
				t.Errorf("Got Prometheus label keys with len %d but expected %d", len(labelKeys), len(tc.expectKeys))
			}

			if len(labelValues) != len(tc.expectValues) {
				t.Errorf("Got Prometheus label values with len %d but expected %d", len(labelValues), len(tc.expectValues))
			}

			for i := range tc.expectKeys {
				if !(tc.expectKeys[i] == labelKeys[i] && tc.expectValues[i] == labelValues[i]) {
					t.Errorf("Got Prometheus label %q: %q but expected %q: %q", labelKeys[i], labelValues[i], tc.expectKeys[i], tc.expectValues[i])
				}
			}
		})
	}

}

func TestKubeAnnotationsToPrometheusAnnotations(t *testing.T) {
	testCases := []struct {
		kubeAnnotations map[string]string
		expectKeys      []string
		expectValues    []string
	}{
		{
			kubeAnnotations: map[string]string{
				"app1": "normal",
			},
			expectKeys:   []string{"annotation_app1"},
			expectValues: []string{"normal"},
		},
		{
			kubeAnnotations: map[string]string{
				"0_app3": "starts_with_digit",
			},
			expectKeys:   []string{"annotation_0_app3"},
			expectValues: []string{"starts_with_digit"},
		},
		{
			kubeAnnotations: map[string]string{
				"": "empty",
			},
			expectKeys:   []string{"annotation_"},
			expectValues: []string{"empty"},
		},
		{
			kubeAnnotations: map[string]string{
				"$app4": "special_char",
			},
			expectKeys:   []string{"annotation__app4"},
			expectValues: []string{"special_char"},
		},
		{
			kubeAnnotations: map[string]string{
				"_app5": "starts_with_underscore",
			},
			expectKeys:   []string{"annotation__app5"},
			expectValues: []string{"starts_with_underscore"},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("kubeannotations input=%v , expected prometheus keys=%v, expected prometheus values=%v", tc.kubeAnnotations, tc.expectKeys, tc.expectValues), func(t *testing.T) {
			annotationKeys, annotationValues := kubeAnnotationsToPrometheusAnnotations(tc.kubeAnnotations)
			if len(annotationKeys) != len(tc.expectKeys) {
				t.Errorf("Got Prometheus label keys with len %d but expected %d", len(annotationKeys), len(tc.expectKeys))
			}

			if len(annotationValues) != len(tc.expectValues) {
				t.Errorf("Got Prometheus label values with len %d but expected %d", len(annotationValues), len(tc.expectValues))
			}

			for i := range tc.expectKeys {
				if !(tc.expectKeys[i] == annotationKeys[i] && tc.expectValues[i] == annotationValues[i]) {
					t.Errorf("Got Prometheus label %q: %q but expected %q: %q", annotationKeys[i], annotationValues[i], tc.expectKeys[i], tc.expectValues[i])
				}
			}
		})
	}

}
