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

package collectors

import (
	"testing"
	"time"

	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kube-state-metrics/pkg/collectors/testutils"
	"k8s.io/kube-state-metrics/pkg/options"
)

var (
	depl1Replicas int32 = 200
	depl2Replicas int32 = 5

	depl1MaxUnavailable = intstr.FromInt(10)
	depl2MaxUnavailable = intstr.FromString("20%")

	depl1MaxSurge = intstr.FromInt(10)
	depl2MaxSurge = intstr.FromString("20%")
)

type mockDeploymentStore struct {
	f func() ([]v1beta1.Deployment, error)
}

func (ds mockDeploymentStore) List() (deployments []v1beta1.Deployment, err error) {
	return ds.f()
}

func TestDeploymentCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_deployment_created Unix creation timestamp
		# TYPE kube_deployment_created gauge
		# HELP kube_deployment_metadata_generation Sequence number representing a specific generation of the desired state.
		# TYPE kube_deployment_metadata_generation gauge
		# HELP kube_deployment_spec_paused Whether the deployment is paused and will not be processed by the deployment controller.
		# TYPE kube_deployment_spec_paused gauge
		# HELP kube_deployment_spec_replicas Number of desired pods for a deployment.
		# TYPE kube_deployment_spec_replicas gauge
		# HELP kube_deployment_status_replicas The number of replicas per deployment.
		# TYPE kube_deployment_status_replicas gauge
		# HELP kube_deployment_status_replicas_available The number of available replicas per deployment.
		# TYPE kube_deployment_status_replicas_available gauge
		# HELP kube_deployment_status_replicas_unavailable The number of unavailable replicas per deployment.
		# TYPE kube_deployment_status_replicas_unavailable gauge
		# HELP kube_deployment_status_replicas_updated The number of updated replicas per deployment.
		# TYPE kube_deployment_status_replicas_updated gauge
		# HELP kube_deployment_status_observed_generation The generation observed by the deployment controller.
		# TYPE kube_deployment_status_observed_generation gauge
		# HELP kube_deployment_spec_strategy_rollingupdate_max_unavailable Maximum number of unavailable replicas during a rolling update of a deployment.
		# TYPE kube_deployment_spec_strategy_rollingupdate_max_unavailable gauge
		# HELP kube_deployment_spec_strategy_rollingupdate_max_surge Maximum number of replicas that can be scheduled above the desired number of replicas during a rolling update of a deployment.
		# TYPE kube_deployment_spec_strategy_rollingupdate_max_surge gauge
		# HELP kube_deployment_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_deployment_labels gauge
	`
	cases := []struct {
		depls []v1beta1.Deployment
		want  string
	}{
		{
			depls: []v1beta1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "depl1",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Namespace:         "ns1",
						Labels: map[string]string{
							"app": "example1",
						},
						Generation: 21,
					},
					Status: v1beta1.DeploymentStatus{
						Replicas:            15,
						AvailableReplicas:   10,
						UnavailableReplicas: 5,
						UpdatedReplicas:     2,
						ObservedGeneration:  111,
					},
					Spec: v1beta1.DeploymentSpec{
						Replicas: &depl1Replicas,
						Strategy: v1beta1.DeploymentStrategy{
							RollingUpdate: &v1beta1.RollingUpdateDeployment{
								MaxUnavailable: &depl1MaxUnavailable,
								MaxSurge:       &depl1MaxSurge,
							},
						},
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "depl2",
						Namespace: "ns2",
						Labels: map[string]string{
							"app": "example2",
						},
						Generation: 14,
					},
					Status: v1beta1.DeploymentStatus{
						Replicas:            10,
						AvailableReplicas:   5,
						UnavailableReplicas: 0,
						UpdatedReplicas:     1,
						ObservedGeneration:  1111,
					},
					Spec: v1beta1.DeploymentSpec{
						Paused:   true,
						Replicas: &depl2Replicas,
						Strategy: v1beta1.DeploymentStrategy{
							RollingUpdate: &v1beta1.RollingUpdateDeployment{
								MaxUnavailable: &depl2MaxUnavailable,
								MaxSurge:       &depl2MaxSurge,
							},
						},
					},
				},
			},
			want: metadata + `
				kube_deployment_created{deployment="depl1",namespace="ns1"} 1.5e+09
				kube_deployment_metadata_generation{namespace="ns1",deployment="depl1"} 21
				kube_deployment_metadata_generation{namespace="ns2",deployment="depl2"} 14
				kube_deployment_spec_paused{namespace="ns1",deployment="depl1"} 0
				kube_deployment_spec_paused{namespace="ns2",deployment="depl2"} 1
				kube_deployment_spec_replicas{namespace="ns1",deployment="depl1"} 200
				kube_deployment_spec_replicas{namespace="ns2",deployment="depl2"} 5
				kube_deployment_spec_strategy_rollingupdate_max_unavailable{deployment="depl1",namespace="ns1"} 10
				kube_deployment_spec_strategy_rollingupdate_max_unavailable{deployment="depl2",namespace="ns2"} 1
				kube_deployment_spec_strategy_rollingupdate_max_surge{deployment="depl1",namespace="ns1"} 10
				kube_deployment_spec_strategy_rollingupdate_max_surge{deployment="depl2",namespace="ns2"} 1
				kube_deployment_status_observed_generation{namespace="ns1",deployment="depl1"} 111
				kube_deployment_status_observed_generation{namespace="ns2",deployment="depl2"} 1111
				kube_deployment_status_replicas{namespace="ns1",deployment="depl1"} 15
				kube_deployment_status_replicas{namespace="ns2",deployment="depl2"} 10
				kube_deployment_status_replicas_available{namespace="ns1",deployment="depl1"} 10
				kube_deployment_status_replicas_available{namespace="ns2",deployment="depl2"} 5
				kube_deployment_status_replicas_unavailable{namespace="ns1",deployment="depl1"} 5
				kube_deployment_status_replicas_unavailable{namespace="ns2",deployment="depl2"} 0
				kube_deployment_status_replicas_updated{namespace="ns1",deployment="depl1"} 2
				kube_deployment_status_replicas_updated{namespace="ns2",deployment="depl2"} 1
				kube_deployment_labels{label_app="example1",namespace="ns1",deployment="depl1"} 1
				kube_deployment_labels{label_app="example2",namespace="ns2",deployment="depl2"} 1
			`,
		},
	}
	for _, c := range cases {
		dc := &deploymentCollector{
			store: mockDeploymentStore{
				f: func() ([]v1beta1.Deployment, error) { return c.depls, nil },
			},
			opts: &options.Options{},
		}
		if err := testutils.GatherAndCompare(dc, c.want, nil); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
