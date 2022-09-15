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

package store

import (
	"testing"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscaling "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func TestVPAStore(t *testing.T) {
	const metadata = `
		# HELP kube_verticalpodautoscaler_labels (Deprecated since v2.9.0) Kubernetes labels converted to Prometheus labels.
        # HELP kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed (Deprecated since v2.9.0) Maximum resources the VerticalPodAutoscaler can set for containers matching the name.
        # HELP kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed (Deprecated since v2.9.0) Minimum resources the VerticalPodAutoscaler can set for containers matching the name.
        # HELP kube_verticalpodautoscaler_spec_updatepolicy_updatemode (Deprecated since v2.9.0) Update mode of the VerticalPodAutoscaler.
        # HELP kube_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound (Deprecated since v2.9.0) Minimum resources the container can use before the VerticalPodAutoscaler updater evicts it.
        # HELP kube_verticalpodautoscaler_status_recommendation_containerrecommendations_target (Deprecated since v2.9.0) Target resources the VerticalPodAutoscaler recommends for the container.
        # HELP kube_verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget (Deprecated since v2.9.0) Target resources the VerticalPodAutoscaler recommends for the container ignoring bounds.
        # HELP kube_verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound (Deprecated since v2.9.0) Maximum resources the container can use before the VerticalPodAutoscaler updater evicts it.
        # TYPE kube_verticalpodautoscaler_labels gauge
        # TYPE kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed gauge
        # TYPE kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed gauge
        # TYPE kube_verticalpodautoscaler_spec_updatepolicy_updatemode gauge
        # TYPE kube_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound gauge
        # TYPE kube_verticalpodautoscaler_status_recommendation_containerrecommendations_target gauge
        # TYPE kube_verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget gauge
        # TYPE kube_verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound gauge
	`

	updateMode := autoscaling.UpdateModeRecreate

	v1Resource := func(cpu, mem string) v1.ResourceList {
		return v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(cpu),
			v1.ResourceMemory: resource.MustParse(mem),
		}
	}

	cases := []generateMetricsTestCase{
		{
			Obj: &autoscaling.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
					Name:       "vpa1",
					Namespace:  "ns1",
					Labels: map[string]string{
						"app": "foobar",
					},
				},
				Spec: autoscaling.VerticalPodAutoscalerSpec{
					TargetRef: &autoscalingv1.CrossVersionObjectReference{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "deployment1",
					},
					UpdatePolicy: &autoscaling.PodUpdatePolicy{
						UpdateMode: &updateMode,
					},
					ResourcePolicy: &autoscaling.PodResourcePolicy{
						ContainerPolicies: []autoscaling.ContainerResourcePolicy{
							{
								ContainerName: "*",
								MinAllowed:    v1Resource("1", "4Gi"),
								MaxAllowed:    v1Resource("4", "8Gi"),
							},
						},
					},
				},
				Status: autoscaling.VerticalPodAutoscalerStatus{
					Recommendation: &autoscaling.RecommendedPodResources{
						ContainerRecommendations: []autoscaling.RecommendedContainerResources{
							{
								ContainerName:  "container1",
								LowerBound:     v1Resource("1", "4Gi"),
								UpperBound:     v1Resource("4", "8Gi"),
								Target:         v1Resource("3", "7Gi"),
								UncappedTarget: v1Resource("6", "10Gi"),
							},
						},
					},
				},
			},
			Want: metadata + `
				kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed{container="*",namespace="ns1",resource="cpu",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="core",verticalpodautoscaler="vpa1"} 4
				kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed{container="*",namespace="ns1",resource="memory",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="byte",verticalpodautoscaler="vpa1"} 8.589934592e+09
				kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed{container="*",namespace="ns1",resource="cpu",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="core",verticalpodautoscaler="vpa1"} 1
				kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed{container="*",namespace="ns1",resource="memory",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="byte",verticalpodautoscaler="vpa1"} 4.294967296e+09
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound{container="container1",namespace="ns1",resource="cpu",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="core",verticalpodautoscaler="vpa1"} 1
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound{container="container1",namespace="ns1",resource="memory",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="byte",verticalpodautoscaler="vpa1"} 4.294967296e+09
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_target{container="container1",namespace="ns1",resource="cpu",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="core",verticalpodautoscaler="vpa1"} 3
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_target{container="container1",namespace="ns1",resource="memory",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="byte",verticalpodautoscaler="vpa1"} 7.516192768e+09
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget{container="container1",namespace="ns1",resource="cpu",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="core",verticalpodautoscaler="vpa1"} 6
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget{container="container1",namespace="ns1",resource="memory",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="byte",verticalpodautoscaler="vpa1"} 1.073741824e+10
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound{container="container1",namespace="ns1",resource="cpu",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="core",verticalpodautoscaler="vpa1"} 4
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound{container="container1",namespace="ns1",resource="memory",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",unit="byte",verticalpodautoscaler="vpa1"} 8.589934592e+09
				kube_verticalpodautoscaler_labels{namespace="ns1",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",verticalpodautoscaler="vpa1"} 1
				kube_verticalpodautoscaler_spec_updatepolicy_updatemode{namespace="ns1",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",update_mode="Auto",verticalpodautoscaler="vpa1"} 0
				kube_verticalpodautoscaler_spec_updatepolicy_updatemode{namespace="ns1",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",update_mode="Initial",verticalpodautoscaler="vpa1"} 0
				kube_verticalpodautoscaler_spec_updatepolicy_updatemode{namespace="ns1",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",update_mode="Off",verticalpodautoscaler="vpa1"} 0
				kube_verticalpodautoscaler_spec_updatepolicy_updatemode{namespace="ns1",target_api_version="apps/v1",target_kind="Deployment",target_name="deployment1",update_mode="Recreate",verticalpodautoscaler="vpa1"} 1
			`,
			MetricNames: []string{
				"kube_verticalpodautoscaler_labels",
				"kube_verticalpodautoscaler_spec_updatepolicy_updatemode",
				"kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed",
				"kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed",
				"kube_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound",
				"kube_verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound",
				"kube_verticalpodautoscaler_status_recommendation_containerrecommendations_target",
				"kube_verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget",
			},
		},
		{
			Obj: &autoscaling.VerticalPodAutoscaler{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 2,
					Name:       "vpa-without-target-ref",
					Namespace:  "ns2",
					Labels: map[string]string{
						"app": "foobar",
					},
				},
				Spec: autoscaling.VerticalPodAutoscalerSpec{
					UpdatePolicy: &autoscaling.PodUpdatePolicy{
						UpdateMode: &updateMode,
					},
					ResourcePolicy: &autoscaling.PodResourcePolicy{
						ContainerPolicies: []autoscaling.ContainerResourcePolicy{
							{
								ContainerName: "*",
								MinAllowed:    v1Resource("1", "4Gi"),
								MaxAllowed:    v1Resource("4", "8Gi"),
							},
						},
					},
				},
				Status: autoscaling.VerticalPodAutoscalerStatus{
					Recommendation: &autoscaling.RecommendedPodResources{
						ContainerRecommendations: []autoscaling.RecommendedContainerResources{
							{
								ContainerName:  "container1",
								LowerBound:     v1Resource("1", "4Gi"),
								UpperBound:     v1Resource("4", "8Gi"),
								Target:         v1Resource("3", "7Gi"),
								UncappedTarget: v1Resource("6", "10Gi"),
							},
						},
					},
				},
			},
			Want: metadata + `
				kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed{container="*",namespace="ns2",resource="cpu",target_api_version="",target_kind="",target_name="",unit="core",verticalpodautoscaler="vpa-without-target-ref"} 4
				kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed{container="*",namespace="ns2",resource="memory",target_api_version="",target_kind="",target_name="",unit="byte",verticalpodautoscaler="vpa-without-target-ref"} 8.589934592e+09
				kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed{container="*",namespace="ns2",resource="cpu",target_api_version="",target_kind="",target_name="",unit="core",verticalpodautoscaler="vpa-without-target-ref"} 1
				kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed{container="*",namespace="ns2",resource="memory",target_api_version="",target_kind="",target_name="",unit="byte",verticalpodautoscaler="vpa-without-target-ref"} 4.294967296e+09
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound{container="container1",namespace="ns2",resource="cpu",target_api_version="",target_kind="",target_name="",unit="core",verticalpodautoscaler="vpa-without-target-ref"} 1
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound{container="container1",namespace="ns2",resource="memory",target_api_version="",target_kind="",target_name="",unit="byte",verticalpodautoscaler="vpa-without-target-ref"} 4.294967296e+09
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_target{container="container1",namespace="ns2",resource="cpu",target_api_version="",target_kind="",target_name="",unit="core",verticalpodautoscaler="vpa-without-target-ref"} 3
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_target{container="container1",namespace="ns2",resource="memory",target_api_version="",target_kind="",target_name="",unit="byte",verticalpodautoscaler="vpa-without-target-ref"} 7.516192768e+09
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget{container="container1",namespace="ns2",resource="cpu",target_api_version="",target_kind="",target_name="",unit="core",verticalpodautoscaler="vpa-without-target-ref"} 6
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget{container="container1",namespace="ns2",resource="memory",target_api_version="",target_kind="",target_name="",unit="byte",verticalpodautoscaler="vpa-without-target-ref"} 1.073741824e+10
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound{container="container1",namespace="ns2",resource="cpu",target_api_version="",target_kind="",target_name="",unit="core",verticalpodautoscaler="vpa-without-target-ref"} 4
				kube_verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound{container="container1",namespace="ns2",resource="memory",target_api_version="",target_kind="",target_name="",unit="byte",verticalpodautoscaler="vpa-without-target-ref"} 8.589934592e+09				
				kube_verticalpodautoscaler_labels{namespace="ns2",target_api_version="",target_kind="",target_name="",verticalpodautoscaler="vpa-without-target-ref"} 1
				kube_verticalpodautoscaler_spec_updatepolicy_updatemode{namespace="ns2",target_api_version="",target_kind="",target_name="",update_mode="Auto",verticalpodautoscaler="vpa-without-target-ref"} 0
				kube_verticalpodautoscaler_spec_updatepolicy_updatemode{namespace="ns2",target_api_version="",target_kind="",target_name="",update_mode="Initial",verticalpodautoscaler="vpa-without-target-ref"} 0
				kube_verticalpodautoscaler_spec_updatepolicy_updatemode{namespace="ns2",target_api_version="",target_kind="",target_name="",update_mode="Off",verticalpodautoscaler="vpa-without-target-ref"} 0
				kube_verticalpodautoscaler_spec_updatepolicy_updatemode{namespace="ns2",target_api_version="",target_kind="",target_name="",update_mode="Recreate",verticalpodautoscaler="vpa-without-target-ref"} 1
			`,
			MetricNames: []string{
				"kube_verticalpodautoscaler_labels",
				"kube_verticalpodautoscaler_spec_updatepolicy_updatemode",
				"kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_minallowed",
				"kube_verticalpodautoscaler_spec_resourcepolicy_container_policies_maxallowed",
				"kube_verticalpodautoscaler_status_recommendation_containerrecommendations_lowerbound",
				"kube_verticalpodautoscaler_status_recommendation_containerrecommendations_upperbound",
				"kube_verticalpodautoscaler_status_recommendation_containerrecommendations_target",
				"kube_verticalpodautoscaler_status_recommendation_containerrecommendations_uncappedtarget",
			},
		},
	}
	for i, c := range cases {
		c.Func = generator.ComposeMetricGenFuncs(vpaMetricFamilies(nil, nil))
		c.Headers = generator.ExtractMetricFamilyHeaders(vpaMetricFamilies(nil, nil))
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
