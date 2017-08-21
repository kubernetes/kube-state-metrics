package collectors

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

type mockNamespaceStore struct {
	f func() ([]v1.Namespace, error)
}

func (ds mockNamespaceStore) List() (deployments []v1.Namespace, err error) {
	return ds.f()
}

func TestNamespaceCollector(t *testing.T) {
	// Fixed metadata on type and help text. We prepend this to every expected
	// output so we only have to modify a single place when doing adjustments.
	const metadata = `
		# HELP kube_namespace_created Unix creation timestamp
		# TYPE kube_namespace_created gauge
		# HELP kube_namespace_labels Kubernetes labels converted to Prometheus labels.
		# TYPE kube_namespace_labels gauge
 	`
	cases := []struct {
		depls []v1.Namespace
		want  string
	}{
		{
			depls: []v1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "ns1",
						CreationTimestamp: metav1.Time{Time: time.Unix(1500000000, 0)},
						Labels: map[string]string{
							"app": "example1",
						},
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns2",
						Labels: map[string]string{
							"app": "example2",
							"l2":  "label2",
						},
					},
				}, {
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns3",
					},
				},
			},

			want: metadata + `
				kube_namespace_created{namespace="ns1"} 1.5e+09
				kube_namespace_labels{label_app="example1",namespace="ns1"} 1
				kube_namespace_labels{label_app="example2",label_l2="label2",namespace="ns2"} 1
				kube_namespace_labels{namespace="ns3"} 1
 			`,
		},
	}
	for _, c := range cases {
		sc := &namespaceCollector{
			store: mockNamespaceStore{
				f: func() ([]v1.Namespace, error) { return c.depls, nil },
			},
		}
		if err := gatherAndCompare(sc, c.want, nil); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}
