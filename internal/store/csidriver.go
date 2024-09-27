/*
Copyright 2024 The Kubernetes Authors All rights reserved.
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
	"context"
	"strconv"

	basemetrics "k8s.io/component-base/metrics"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	defaultSELinuxMount              = false
	descCSIDriverLabelsDefaultLabels = []string{"csi_driver"}
)

func csiDriverMetricFamilies(allowAnnotationsList, allowLabelsList []string) []generator.FamilyGenerator {
	return []generator.FamilyGenerator{
		*generator.NewFamilyGeneratorWithStability(
			"kube_csidriver_info",
			"Information about CSI drivers.",
			metric.Gauge,
			basemetrics.ALPHA,
			"",
			wrapCSIDriverFunc(func(c *storagev1.CSIDriver) *metric.Family {
				if c.Spec.SELinuxMount == nil {
					c.Spec.SELinuxMount = &defaultSELinuxMount
				}
				m := metric.Metric{
					LabelKeys:   []string{"selinux_mount"},
					LabelValues: []string{strconv.FormatBool(*c.Spec.SELinuxMount)},
					Value:       1,
				}
				return &metric.Family{Metrics: []*metric.Metric{&m}}
			}),
		),
	}
}

func wrapCSIDriverFunc(f func(*storagev1.CSIDriver) *metric.Family) func(interface{}) *metric.Family {
	return func(obj interface{}) *metric.Family {
		csiDriver := obj.(*storagev1.CSIDriver)

		metricFamily := f(csiDriver)

		for _, m := range metricFamily.Metrics {
			m.LabelKeys, m.LabelValues = mergeKeyValues(descCSIDriverLabelsDefaultLabels, []string{csiDriver.Name}, m.LabelKeys, m.LabelValues)
		}

		return metricFamily
	}
}

func createCSIDriverListWatch(kubeClient clientset.Interface, _ string, _ string) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return kubeClient.StorageV1().CSIDrivers().List(context.TODO(), opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return kubeClient.StorageV1().CSIDrivers().Watch(context.TODO(), opts)
		},
	}
}
