/*
Copyright 2025 The Kubernetes Authors All rights reserved.

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
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/internal"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestObjectLimits(t *testing.T) {

	// Initialise options.
	opts := options.NewOptions()
	cmd := options.InitCommand
	opts.AddFlags(cmd)
	klog.InfoS("options", "options", opts)

	// Populate options, and parse them.
	opts.ObjectLimit = 5
	opts.Resources = options.ResourceSet{"configmaps": struct{}{}}
	opts.Kubeconfig = os.Getenv("HOME") + "/.kube/config"
	if err := opts.Parse(); err != nil {
		t.Fatalf("failed to parse options: %v", err)
	}
	klog.InfoS("parsed options", "options", opts)

	// Create ConfigMaps as Test Objects
	for i := 0; i < 6; i++ {
		err := exec.Command("kubectl", "create", "configmap", fmt.Sprintf("testcm%v", i)).Run() //nolint:gosec
		if err != nil {
			t.Fatalf("failed to create configmap : %v", err)
		}
	}

	// Make the process asynchronous.
	go internal.RunKubeStateMetricsWrapper(opts)
	klog.InfoS("started KSM")

	// Wait for port 8080 to come up.
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
	klog.InfoS("port 8080 up")

	// Wait for the metric to be available.
	ch := make(chan bool, 1)
	klog.InfoS("waiting for first metrics to become available")
	testMetric := `kube_configmap_info{namespace="default"`
	err = wait.PollUntilContextTimeout(context.TODO(), 1*time.Second, 20*time.Second, true, func(_ context.Context) (bool, error) {
		out, err := exec.Command("curl", "localhost:8080/metrics").Output()

		if err != nil {
			return false, err
		}
		if string(out) == "" {
			return false, nil
		}
		// Note: we use count to make sure that only one metrics handler is running
		if strings.Count(string(out), testMetric) == 5 {
			// klog.InfoS("metrics available", "metric", string(out))
			// Signal the process to exit, since we know the metrics are being generated as expected.
			ch <- true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("failed while waiting for initial metrics to be available: %v", err)
	}

	// Delete ConfigMaps as Test Objects
	for i := 0; i < 6; i++ {
		err := exec.Command("kubectl", "delete", "configmap", fmt.Sprintf("testcm%v", i)).Run() //nolint:gosec
		if err != nil {
			t.Fatalf("failed to delete configmap : %v", err)
		}
	}

	// Wait for process to exit.
	select {
	case <-ch:
		t.Log("initial metrics are available")
	case <-time.After(40 * time.Second):
		t.Fatal("timed out waiting for test to pass, check the logs for more info")
	}
}
