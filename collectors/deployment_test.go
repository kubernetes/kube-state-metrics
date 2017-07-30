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
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

var (
	depl1Replicas int32 = 200
	depl2Replicas int32 = 5

	depl1MaxUnavailable = intstr.FromInt(10)
	depl2MaxUnavailable = intstr.FromString("20%")
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
						Name:      "depl1",
						Namespace: "ns1",
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
					},
				},
			},
			want: metadata + `
				kube_deployment_metadata_generation{namespace="ns1",deployment="depl1"} 21
				kube_deployment_metadata_generation{namespace="ns2",deployment="depl2"} 14
				kube_deployment_spec_paused{namespace="ns1",deployment="depl1"} 0
				kube_deployment_spec_paused{namespace="ns2",deployment="depl2"} 1
				kube_deployment_spec_replicas{namespace="ns1",deployment="depl1"} 200
				kube_deployment_spec_replicas{namespace="ns2",deployment="depl2"} 5
				kube_deployment_spec_strategy_rollingupdate_max_unavailable{deployment="depl1",namespace="ns1"} 10
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
		}
		if err := gatherAndCompare(dc, c.want, nil); err != nil {
			t.Errorf("unexpected collecting result:\n%s", err)
		}
	}
}

// gatherAndCompare retrieves all metrics exposed by a collector and compares it
// to an expected output in the Prometheus text exposition format.
// metricNames allows only comparing the given metrics. All are compared if it's nil.
func gatherAndCompare(c prometheus.Collector, expected string, metricNames []string) error {
	expected = removeUnusedWhitespace(expected)

	reg := prometheus.NewPedanticRegistry()
	if err := reg.Register(c); err != nil {
		return fmt.Errorf("registering collector failed: %s", err)
	}
	metrics, err := reg.Gather()
	if err != nil {
		return fmt.Errorf("gathering metrics failed: %s", err)
	}
	if metricNames != nil {
		metrics = filterMetrics(metrics, metricNames)
	}
	var tp expfmt.TextParser
	expectedMetrics, err := tp.TextToMetricFamilies(bytes.NewReader([]byte(expected)))
	if err != nil {
		return fmt.Errorf("parsing expected metrics failed: %s", err)
	}

	if !reflect.DeepEqual(metrics, normalizeMetricFamilies(expectedMetrics)) {
		// Encode the gathered output to the readbale text format for comparison.
		var buf1 bytes.Buffer
		enc := expfmt.NewEncoder(&buf1, expfmt.FmtText)
		for _, mf := range metrics {
			if err := enc.Encode(mf); err != nil {
				return fmt.Errorf("encoding result failed: %s", err)
			}
		}
		// Encode normalized expected metrics again to generate them in the same ordering
		// the registry does to spot differences more easily.
		var buf2 bytes.Buffer
		enc = expfmt.NewEncoder(&buf2, expfmt.FmtText)
		for _, mf := range normalizeMetricFamilies(expectedMetrics) {
			if err := enc.Encode(mf); err != nil {
				return fmt.Errorf("encoding result failed: %s", err)
			}
		}

		return fmt.Errorf(`
metric output does not match expectation; want:

%s

got:

%s
`, buf2.String(), buf1.String())
	}
	return nil
}

func filterMetrics(metrics []*dto.MetricFamily, names []string) []*dto.MetricFamily {
	var filtered []*dto.MetricFamily
	for _, m := range metrics {
		drop := true
		for _, name := range names {
			if m.GetName() == name {
				drop = false
				break
			}
		}
		if !drop {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func removeUnusedWhitespace(s string) string {
	var (
		trimmedLine  string
		trimmedLines []string
		lines        = strings.Split(s, "\n")
	)

	for _, l := range lines {
		trimmedLine = strings.TrimSpace(l)

		if len(trimmedLine) > 0 {
			trimmedLines = append(trimmedLines, trimmedLine)
		}
	}

	// The Prometheus metrics representation parser expects an empty line at the
	// end otherwise fails with an unexpected EOF error.
	return strings.Join(trimmedLines, "\n") + "\n"
}

// The below sorting code is copied form the Prometheus client library modulo the added
// label pair sorting.
// https://github.com/prometheus/client_golang/blob/ea6e1db4cb8127eeb0b6954f7320363e5451820f/prometheus/registry.go#L642-L684

// metricSorter is a sortable slice of *dto.Metric.
type metricSorter []*dto.Metric

func (s metricSorter) Len() int {
	return len(s)
}

func (s metricSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s metricSorter) Less(i, j int) bool {
	sort.Sort(prometheus.LabelPairSorter(s[i].Label))
	sort.Sort(prometheus.LabelPairSorter(s[j].Label))

	if len(s[i].Label) != len(s[j].Label) {
		return len(s[i].Label) < len(s[j].Label)
	}

	for n, lp := range s[i].Label {
		vi := lp.GetValue()
		vj := s[j].Label[n].GetValue()
		if vi != vj {
			return vi < vj
		}
	}

	if s[i].TimestampMs == nil {
		return false
	}
	if s[j].TimestampMs == nil {
		return true
	}
	return s[i].GetTimestampMs() < s[j].GetTimestampMs()
}

// normalizeMetricFamilies returns a MetricFamily slice with empty
// MetricFamilies pruned and the remaining MetricFamilies sorted by name within
// the slice, with the contained Metrics sorted within each MetricFamily.
func normalizeMetricFamilies(metricFamiliesByName map[string]*dto.MetricFamily) []*dto.MetricFamily {
	for _, mf := range metricFamiliesByName {
		sort.Sort(metricSorter(mf.Metric))
	}
	names := make([]string, 0, len(metricFamiliesByName))
	for name, mf := range metricFamiliesByName {
		if len(mf.Metric) > 0 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	result := make([]*dto.MetricFamily, 0, len(names))
	for _, name := range names {
		result = append(result, metricFamiliesByName[name])
	}
	return result
}
