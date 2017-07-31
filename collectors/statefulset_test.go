package collectors

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"
)

var (
	statefulSet1Replicas int32 = 3
	statefulSet2Replicas int32 = 6
	statefulSet3Replicas int32 = 9

	statefulSet1ObservedGeneration int64 = 1
	statefulSet2ObservedGeneration int64 = 2
)

type mockStatefulSetStore struct {
	f func() ([]v1beta1.StatefulSet, error)
}

func (ds mockStatefulSetStore) List() (deployments []v1beta1.StatefulSet, err error) {
	return ds.f()
}

func TestStatefuleSetCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
 		# HELP kube_statefulset_status_replicas The number of replicas per StatefulSet.
 		# TYPE kube_statefulset_status_replicas gauge
 		# HELP kube_statefulset_status_observed_generation The generation observed by the StatefulSet controller.
 		# TYPE kube_statefulset_status_observed_generation gauge
 		# HELP kube_statefulset_replicas Number of desired pods for a StatefulSet.
 		# TYPE kube_statefulset_replicas gauge
 		# HELP kube_statefulset_metadata_generation Sequence number representing a specific generation of the desired state for the StatefulSet.
 		# TYPE kube_statefulset_metadata_generation gauge
		# HELP kube_statefulset_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_statefulset_labels gauge
 	`
	cases := []struct {
		depls []v1beta1.StatefulSet
		want  string
	}{
		{
			depls: []v1beta1.StatefulSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "statefulset1",
						Namespace: "ns1",
						Labels: map[string]string{
							"app": "example1",
						},
						Generation: 3,
					},
					Spec: v1beta1.StatefulSetSpec{
						Replicas:    &statefulSet1Replicas,
						ServiceName: "statefulset1service",
					},
					Status: v1beta1.StatefulSetStatus{
						ObservedGeneration: &statefulSet1ObservedGeneration,
						Replicas:           2,
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "statefulset2",
						Namespace: "ns2",
						Labels: map[string]string{
							"app": "example2",
						},
						Generation: 21,
					},
					Spec: v1beta1.StatefulSetSpec{
						Replicas:    &statefulSet2Replicas,
						ServiceName: "statefulset2service",
					},
					Status: v1beta1.StatefulSetStatus{
						ObservedGeneration: &statefulSet2ObservedGeneration,
						Replicas:           5,
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "statefulset3",
						Namespace: "ns3",
						Labels: map[string]string{
							"app": "example3",
						},
						Generation: 36,
					},
					Spec: v1beta1.StatefulSetSpec{
						Replicas:    &statefulSet3Replicas,
						ServiceName: "statefulset2service",
					},
					Status: v1beta1.StatefulSetStatus{
						ObservedGeneration: nil,
						Replicas:           7,
					},
				},
			},
			want: metadata + `
 				kube_statefulset_status_replicas{namespace="ns1",statefulset="statefulset1"} 2
 				kube_statefulset_status_replicas{namespace="ns2",statefulset="statefulset2"} 5
 				kube_statefulset_status_replicas{namespace="ns3",statefulset="statefulset3"} 7
 				kube_statefulset_status_observed_generation{namespace="ns1",statefulset="statefulset1"} 1
 				kube_statefulset_status_observed_generation{namespace="ns2",statefulset="statefulset2"} 2
 				kube_statefulset_replicas{namespace="ns1",statefulset="statefulset1"} 3
 				kube_statefulset_replicas{namespace="ns2",statefulset="statefulset2"} 6
 				kube_statefulset_replicas{namespace="ns3",statefulset="statefulset3"} 9
 				kube_statefulset_metadata_generation{namespace="ns1",statefulset="statefulset1"} 3
 				kube_statefulset_metadata_generation{namespace="ns2",statefulset="statefulset2"} 21
 				kube_statefulset_metadata_generation{namespace="ns3",statefulset="statefulset3"} 36
				kube_statefulset_labels{label_app="example1",namespace="ns1",statefulset="statefulset1"} 1
				kube_statefulset_labels{label_app="example2",namespace="ns2",statefulset="statefulset2"} 1
				kube_statefulset_labels{label_app="example3",namespace="ns3",statefulset="statefulset3"} 1
 			`,
		},
	}
	for _, c := range cases {
		sc := &statefulSetCollector{
			store: mockStatefulSetStore{
				f: func() ([]v1beta1.StatefulSet, error) { return c.depls, nil },
			},
		}
		if err := gatherAndCompare(sc, c.want, nil); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
