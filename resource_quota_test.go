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

package main

import (
	"testing"

	"k8s.io/client-go/1.5/pkg/api/resource"
	"k8s.io/client-go/1.5/pkg/api/v1"
)

type mockResourceQuotaStore struct {
	list func() (v1.ResourceQuotaList, error)
}

func (ns mockResourceQuotaStore) List() (v1.ResourceQuotaList, error) {
	return ns.list()
}

func TestResourceQuotaCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
	# HELP kube_resource_quota_cpu Information about CPU resource quota.
	# TYPE kube_resource_quota_cpu gauge
	# HELP kube_resource_quota_memory Information about memory resource quota.
	# TYPE kube_resource_quota_memory gauge
	# HELP kube_resource_quota_storage Information about storage resource quota.
	# TYPE kube_resource_quota_storage gauge
	# HELP kube_resource_quota_pods Information about Pods resource quota.
	# TYPE kube_resource_quota_pods gauge
	# HELP kube_resource_quota_services Information about services resource quota.
	# TYPE kube_resource_quota_services gauge
	# HELP kube_resource_quota_replication_controllers Information about replication controllers resource quota.
	# TYPE kube_resource_quota_replication_controllers gauge
	# HELP kube_resource_quota_resource_quota Information about resource quotas resource quota.
	# TYPE kube_resource_quota_resource_quota gauge
	# HELP kube_resource_quota_secrets Information about secrets resource quota.
	# TYPE kube_resource_quota_secrets gauge
	# HELP kube_resource_quota_config_maps Information about configmaps hard resource quota.
	# TYPE kube_resource_quota_config_maps gauge
	# HELP kube_resource_quota_persistent_volume_claims Information about persistent volume claims resource quota.
	# TYPE kube_resource_quota_persistent_volume_claims gauge
	# HELP kube_resource_quota_node_ports Information about node ports resource quota.
	# TYPE kube_resource_quota_node_ports gauge
	# HELP kube_resource_quota_load_balancers Information about load balancers resource quota.
	# TYPE kube_resource_quota_load_balancers gauge
	`
	cases := []struct {
		quotas  []v1.ResourceQuota
		metrics []string // which metrics should be checked
		want    string
	}{
		// Verify populating base metrics and that metrics for unset fields are skipped.
		{
			quotas: []v1.ResourceQuota{
				{
					ObjectMeta: v1.ObjectMeta{
						Name:      "quotaTest",
						Namespace: "testNS",
					},
					Status: v1.ResourceQuotaStatus{},
				},
			},
			want: metadata,
		},
		// Verify resource metrics.
		{
			quotas: []v1.ResourceQuota{
				{
					ObjectMeta: v1.ObjectMeta{
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
			},
			want: metadata + `
			kube_resource_quota_cpu{name="quotaTest",namespace="testNS",resource="cpu",type="hard"} 4.3
			kube_resource_quota_cpu{name="quotaTest",namespace="testNS",resource="cpu",type="used"} 2.1
			kube_resource_quota_memory{name="quotaTest",namespace="testNS",resource="memory",type="hard"} 2.1e+09
			kube_resource_quota_memory{name="quotaTest",namespace="testNS",resource="memory",type="used"} 5e+08
			kube_resource_quota_storage{name="quotaTest",namespace="testNS",resource="storage",type="hard"} 1e+10
			kube_resource_quota_storage{name="quotaTest",namespace="testNS",resource="storage",type="used"} 9e+09
			kube_resource_quota_pods{name="quotaTest",namespace="testNS",resource="pods",type="hard"} 9
			kube_resource_quota_pods{name="quotaTest",namespace="testNS",resource="pods",type="used"} 8
			kube_resource_quota_services{name="quotaTest",namespace="testNS",resource="services",type="hard"} 8
			kube_resource_quota_services{name="quotaTest",namespace="testNS",resource="services",type="used"} 7
			kube_resource_quota_replication_controllers{name="quotaTest",namespace="testNS",resource="replicationcontrollers",type="hard"} 7
			kube_resource_quota_replication_controllers{name="quotaTest",namespace="testNS",resource="replicationcontrollers",type="used"} 6
			kube_resource_quota_resource_quota{name="quotaTest",namespace="testNS",resource="resourcequotas",type="hard"} 6
			kube_resource_quota_resource_quota{name="quotaTest",namespace="testNS",resource="resourcequotas",type="used"} 5
			kube_resource_quota_secrets{name="quotaTest",namespace="testNS",resource="secrets",type="hard"} 5
			kube_resource_quota_secrets{name="quotaTest",namespace="testNS",resource="secrets",type="used"} 4
			kube_resource_quota_config_maps{name="quotaTest",namespace="testNS",resource="configmaps",type="hard"} 4
			kube_resource_quota_config_maps{name="quotaTest",namespace="testNS",resource="configmaps",type="used"} 3
			kube_resource_quota_persistent_volume_claims{name="quotaTest",namespace="testNS",resource="persistentvolumeclaims",type="hard"} 3
			kube_resource_quota_persistent_volume_claims{name="quotaTest",namespace="testNS",resource="persistentvolumeclaims",type="used"} 2
			kube_resource_quota_node_ports{name="quotaTest",namespace="testNS",resource="services.nodeports",type="hard"} 2
			kube_resource_quota_node_ports{name="quotaTest",namespace="testNS",resource="services.nodeports",type="used"} 1
			kube_resource_quota_load_balancers{name="quotaTest",namespace="testNS",resource="services.loadbalancers",type="hard"} 1
			kube_resource_quota_load_balancers{name="quotaTest",namespace="testNS",resource="services.loadbalancers",type="used"} 0
			`,
		},
	}
	for _, c := range cases {
		dc := &resourceQuotaCollector{
			store: &mockResourceQuotaStore{
				list: func() (v1.ResourceQuotaList, error) {
					return v1.ResourceQuotaList{Items: c.quotas}, nil
				},
			},
		}
		if err := gatherAndCompare(dc, c.want, c.metrics); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
