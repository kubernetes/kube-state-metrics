/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

func NewMetricFamilyDef(name, help string, labelKeys []string, constLabels prometheus.Labels) *MetricFamilyDef {
	return &MetricFamilyDef{name, help, labelKeys, constLabels}
}

// MetricFamilyDef represents a metric family definition
type MetricFamilyDef struct {
	Name        string
	Help        string
	LabelKeys   []string
	ConstLabels prometheus.Labels
}
