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

package e2e

import (
	"context"
	"net"
	"os"
	"os/exec"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/internal"
	"k8s.io/kube-state-metrics/v2/internal/discovery"
	"k8s.io/kube-state-metrics/v2/pkg/options"
	ksmFramework "k8s.io/kube-state-metrics/v2/tests/e2e/framework"
)

// TestCRDInformerUpdateFuncHandler tests that when an existing CRD is updated
// to add new versions, KSM becomes aware of that change and updates metrics
// accordingly, thus also validating the UpdateFunc in the discovery informer.
func TestCRDInformerUpdateFuncHandler(t *testing.T) {
	const (
		assetDir        = "testdata/crd-informer-updatefunc"
		populateTimeout = 10 * time.Second
	)

	m := &struct {
		crV1          string
		crV1alpha1    string
		crdV1V1alpha1 string
		crdV1alpha1   string
		crsConfig     string
	}{
		crV1:          assetDir + "/cr_v1.yaml",
		crV1alpha1:    assetDir + "/cr_v1alpha1.yaml",
		crdV1V1alpha1: assetDir + "/crd_v1_v1alpha1.yaml",
		crdV1alpha1:   assetDir + "/crd_v1alpha1.yaml",
		crsConfig:     assetDir + "/crs_config.yaml",
	}

	defer func() {
		klog.InfoS("cleaning up test resources")
		_ = exec.Command("kubectl", "delete", "application", "test-app-alpha").Run()   //nolint:gosec
		_ = exec.Command("kubectl", "delete", "application", "test-app-stable").Run()  //nolint:gosec
		_ = exec.Command("kubectl", "delete", "crd", "applications.example.com").Run() //nolint:gosec
	}()

	opts := options.NewOptions()
	cmd := options.InitCommand
	opts.AddFlags(cmd)
	klog.InfoS("options", "options", opts)

	opts.CustomResourceConfigFile = m.crsConfig
	kubeconfig, found := os.LookupEnv("KUBECONFIG")
	if !found {
		t.Fatalf("KUBECONFIG environment variable not set")
	}
	opts.Kubeconfig = kubeconfig
	if err := opts.Parse(); err != nil {
		t.Fatalf("failed to parse options: %v", err)
	}
	klog.InfoS("parsed options", "options", opts)

	go internal.RunKubeStateMetricsWrapper(opts)

	err := wait.PollUntilContextTimeout(context.TODO(), 1*time.Second, 20*time.Second, true, func(_ context.Context) (bool, error) {
		conn, err := net.Dial("tcp", "localhost:8080")
		if err != nil {
			return false, nil
		}
		err = conn.Close()
		if err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed while waiting for port 8080 to come up: %v", err)
	}

	f, err := ksmFramework.New("http://localhost:8080", "http://localhost:8081")
	if err != nil {
		t.Fatalf("failed to create test framework: %v", err)
	}

	// Apply v1alpha assets.
	err = exec.Command("kubectl", "apply", "-f", m.crdV1alpha1).Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to apply v1alpha1 crd: %v", err)
	}
	err = exec.Command("kubectl", "apply", "-f", m.crV1alpha1).Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to apply v1alpha1 cr: %v", err)
	}

	v1alpha1Labels := map[string]string{
		"customresource_group":   "example.com",
		"customresource_kind":    "Application",
		"customresource_version": "v1alpha1",
		"name":                   "test-app-alpha",
	}
	err = wait.PollUntilContextTimeout(context.TODO(), discovery.Interval, populateTimeout, true, func(_ context.Context) (bool, error) {
		found, err := f.HasMetricWithLabels("kube_customresource_test_metric_info", v1alpha1Labels)
		if err != nil {
			klog.ErrorS(err, "failed to check for metric")
			return false, nil
		}
		if found {
			klog.InfoS("v1alpha1 metric found")
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("failed while waiting for v1alpha1 metrics to be available: %v", err)
	}

	// Update the CRD to include both v1alpha1 and v1 versions (with v1 as the storage version).
	err = exec.Command("kubectl", "apply", "-f", m.crdV1V1alpha1).Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to apply updated crd with both versions: %v", err)
	}

	// Apply v1 CR.
	err = exec.Command("kubectl", "apply", "-f", m.crV1).Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to apply v1 cr: %v", err)
	}

	// Wait for v1 metrics to be available.
	// Note: After the CRD update makes v1 the storage version, both CRs will be
	// converted to v1, so we expect to see v1 metrics for both.
	v1LabelsAlpha := map[string]string{
		"customresource_group":   "example.com",
		"customresource_kind":    "Application",
		"customresource_version": "v1",
		"name":                   "test-app-alpha",
	}
	v1LabelsStable := map[string]string{
		"customresource_group":   "example.com",
		"customresource_kind":    "Application",
		"customresource_version": "v1",
		"name":                   "test-app-stable",
	}
	err = wait.PollUntilContextTimeout(context.TODO(), discovery.Interval, populateTimeout*2, true, func(_ context.Context) (bool, error) {
		hasV1Alpha, err := f.HasMetricWithLabels("kube_customresource_test_metric_info", v1LabelsAlpha)
		if err != nil {
			klog.ErrorS(err, "failed to check for v1 metric (test-app-alpha)")
			return false, nil
		}
		hasV1Stable, err := f.HasMetricWithLabels("kube_customresource_test_metric_info", v1LabelsStable)
		if err != nil {
			klog.ErrorS(err, "failed to check for v1 metric (test-app-stable)")
			return false, nil
		}

		if hasV1Alpha && hasV1Stable {
			klog.InfoS("both CRs now showing with v1 version label")
			return true, nil
		}

		if !hasV1Alpha {
			klog.InfoS("v1 metric for test-app-alpha not found yet")
		}
		if !hasV1Stable {
			klog.InfoS("v1 metric for test-app-stable not found yet")
		}

		return false, nil
	})
	if err != nil {
		t.Fatalf("failed while waiting for v1 version metrics to be available: %v", err)
	}
}
