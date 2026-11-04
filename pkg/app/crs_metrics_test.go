/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"k8s.io/kube-state-metrics/v2/pkg/options"
	"k8s.io/kube-state-metrics/v2/pkg/runconfig"
)

func TestApplyCustomResourceConfigMetrics_PreparedAbsenceSkipsDisk(t *testing.T) {
	dir := t.TempDir()
	crcPath := filepath.Join(dir, "crc.yaml")

	opts := options.NewOptions()
	opts.CustomResourceConfigFile = crcPath
	opts.Prepared = &runconfig.Prepared{
		CRSInline: `kind: CustomResourceStateMetrics
apiVersion: k8s.io/kube-state-metrics/v2/customresourcestate/v1alpha1
spec:
  resources: []
`,
		CRSFilePath: crcPath,
	}

	if err := os.Mkdir(crcPath, 0o755); err != nil {
		t.Fatal(err)
	}

	reg := prometheus.NewRegistry()
	configHash := promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{Name: "test_config_hash"}, []string{"type", "filename"})
	configSuccess := promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{Name: "test_config_success"}, []string{"type", "filename"})
	configSuccessTime := promauto.With(reg).NewGaugeVec(
		prometheus.GaugeOpts{Name: "test_config_success_time"}, []string{"type", "filename"})

	if err := applyCustomResourceConfigMetrics(opts, configHash, configSuccess, configSuccessTime); err != nil {
		t.Fatalf("prepared absence must not touch disk: %v", err)
	}
}
