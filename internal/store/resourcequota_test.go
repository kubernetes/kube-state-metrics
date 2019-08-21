/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/kube-state-metrics/pkg/metric"
)

func TestResourceQuotaStore(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
	# HELP kube_resourcequota Information about resource quota.
	# TYPE kube_resourcequota gauge
	# HELP kube_resourcequota_created Unix creation timestamp
	# TYPE kube_resourcequota_created gauge
	`
	cases := []generateMetricsTestCase{
		// Verify populating base metric and that metric for unset fields are skipped.
		{
			Obj: &v1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "quotaTest",
					CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
					Namespace:         "testNS",
				},
				Status: v1.ResourceQuotaStatus{},
			},
			Want: metadata + `
			kube_resourcequota_created{namespace="testNS",resourcequota="quotaTest"} 1.5e+09
			`,
		},
		// Verify resource metric.
		{
			Obj: &v1.ResourceQuota{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "quotaTest",
					Namespace: "testNS",
				},
				Spec: v1.ResourceQuotaSpec{
					Hard: v1.ResourceList{
						v1.ResourceCPU:                    resource.MustParse("4.3"),
						v1.ResourceMemory:                 resource.MustParse("2.1G"),
						v1.ResourceStorage:                resource.MustParse("10G"),
						v1.ResourcePods:                   resource.MustParse("9"),
						v1.ResourceServices:               resource.MustParse("8"),
						v1.ResourceReplicationControllers: resource.MustParse("7"),
						v1.ResourceQuotas:                 resource.MustParse("6"),
						v1.ResourceSecrets:                resource.MustParse("5"),
						v1.ResourceConfigMaps:             resource.MustParse("4"),
						v1.ResourcePersistentVolumeClaims: resource.MustParse("3"),
						v1.ResourceServicesNodePorts:      resource.MustParse("2"),
						v1.ResourceServicesLoadBalancers:  resource.MustParse("1"),
					},
				},
				Status: v1.ResourceQuotaStatus{
					Hard: v1.ResourceList{
						v1.ResourceCPU:                    resource.MustParse("4.3"),
						v1.ResourceMemory:                 resource.MustParse("2.1G"),
						v1.ResourceStorage:                resource.MustParse("10G"),
						v1.ResourcePods:                   resource.MustParse("9"),
						v1.ResourceServices:               resource.MustParse("8"),
						v1.ResourceReplicationControllers: resource.MustParse("7"),
						v1.ResourceQuotas:                 resource.MustParse("6"),
						v1.ResourceSecrets:                resource.MustParse("5"),
						v1.ResourceConfigMaps:             resource.MustParse("4"),
						v1.ResourcePersistentVolumeClaims: resource.MustParse("3"),
						v1.ResourceServicesNodePorts:      resource.MustParse("2"),
						v1.ResourceServicesLoadBalancers:  resource.MustParse("1"),
					},
					Used: v1.ResourceList{
						v1.ResourceCPU:                    resource.MustParse("2.1"),
						v1.ResourceMemory:                 resource.MustParse("500M"),
						v1.ResourceStorage:                resource.MustParse("9G"),
						v1.ResourcePods:                   resource.MustParse("8"),
						v1.ResourceServices:               resource.MustParse("7"),
						v1.ResourceReplicationControllers: resource.MustParse("6"),
						v1.ResourceQuotas:                 resource.MustParse("5"),
						v1.ResourceSecrets:                resource.MustParse("4"),
						v1.ResourceConfigMaps:             resource.MustParse("3"),
						v1.ResourcePersistentVolumeClaims: resource.MustParse("2"),
						v1.ResourceServicesNodePorts:      resource.MustParse("1"),
						v1.ResourceServicesLoadBalancers:  resource.MustParse("0"),
					},
				},
			},
			Want: metadata + `
			kube_resourcequota{namespace="testNS",resource="configmaps",resourcequota="quotaTest",type="hard"} 4
			kube_resourcequota{namespace="testNS",resource="configmaps",resourcequota="quotaTest",type="used"} 3
			kube_resourcequota{namespace="testNS",resource="cpu",resourcequota="quotaTest",type="hard"} 4.3
			kube_resourcequota{namespace="testNS",resource="cpu",resourcequota="quotaTest",type="used"} 2.1
			kube_resourcequota{namespace="testNS",resource="memory",resourcequota="quotaTest",type="hard"} 2.1e+09
			kube_resourcequota{namespace="testNS",resource="memory",resourcequota="quotaTest",type="used"} 5e+08
			kube_resourcequota{namespace="testNS",resource="persistentvolumeclaims",resourcequota="quotaTest",type="hard"} 3
			kube_resourcequota{namespace="testNS",resource="persistentvolumeclaims",resourcequota="quotaTest",type="used"} 2
			kube_resourcequota{namespace="testNS",resource="pods",resourcequota="quotaTest",type="hard"} 9
			kube_resourcequota{namespace="testNS",resource="pods",resourcequota="quotaTest",type="used"} 8
			kube_resourcequota{namespace="testNS",resource="replicationcontrollers",resourcequota="quotaTest",type="hard"} 7
			kube_resourcequota{namespace="testNS",resource="replicationcontrollers",resourcequota="quotaTest",type="used"} 6
			kube_resourcequota{namespace="testNS",resource="resourcequotas",resourcequota="quotaTest",type="hard"} 6
			kube_resourcequota{namespace="testNS",resource="resourcequotas",resourcequota="quotaTest",type="used"} 5
			kube_resourcequota{namespace="testNS",resource="secrets",resourcequota="quotaTest",type="hard"} 5
			kube_resourcequota{namespace="testNS",resource="secrets",resourcequota="quotaTest",type="used"} 4
			kube_resourcequota{namespace="testNS",resource="services",resourcequota="quotaTest",type="hard"} 8
			kube_resourcequota{namespace="testNS",resource="services",resourcequota="quotaTest",type="used"} 7
			kube_resourcequota{namespace="testNS",resource="services.loadbalancers",resourcequota="quotaTest",type="hard"} 1
			kube_resourcequota{namespace="testNS",resource="services.loadbalancers",resourcequota="quotaTest",type="used"} 0
			kube_resourcequota{namespace="testNS",resource="services.nodeports",resourcequota="quotaTest",type="hard"} 2
			kube_resourcequota{namespace="testNS",resource="services.nodeports",resourcequota="quotaTest",type="used"} 1
			kube_resourcequota{namespace="testNS",resource="storage",resourcequota="quotaTest",type="hard"} 1e+10
			kube_resourcequota{namespace="testNS",resource="storage",resourcequota="quotaTest",type="used"} 9e+09
			`,
		},
	}
	for i, c := range cases {
		c.Func = metric.ComposeMetricGenFuncs(resourceQuotaMetricFamilies)
		c.Headers = metric.ExtractMetricFamilyHeaders(resourceQuotaMetricFamilies)
		if err := c.run(); err != nil {
			t.Errorf("unexpected collecting result in %vth run:\n%s", i, err)
		}
	}
}
