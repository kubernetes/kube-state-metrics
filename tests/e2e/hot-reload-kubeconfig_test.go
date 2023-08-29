/*
Copyright 2023 The Kubernetes Authors All rights reserved.

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
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/kube-state-metrics/v2/internal"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestKubeConfigHotReload(t *testing.T) {

	// Initialise options.
	opts := options.NewOptions()
	cmd := options.InitCommand
	opts.AddFlags(cmd)

	// Open kubeconfig
	originalKubeconfig := os.Getenv("KUBECONFIG")
	if originalKubeconfig == "" {
		// Assume $HOME is always defined.
		originalKubeconfig = os.Getenv("HOME") + "/.kube/config"
	}
	originalKubeconfigFp, err := os.Open(filepath.Clean(originalKubeconfig))
	if err != nil {
		t.Fatalf("failed to open kubeconfig: %v", err)
	}
	defer originalKubeconfigFp.Close()

	// Create temporal kubeconfig based on original one
	kubeconfigFp, err := os.CreateTemp("", "ksm-hot-reload-kubeconfig")
	if err != nil {
		t.Fatalf("failed to create temporal kubeconfig: %v", err)
	}
	defer os.Remove(kubeconfigFp.Name())

	if _, err := io.Copy(kubeconfigFp, originalKubeconfigFp); err != nil {
		t.Fatalf("failed to copy from original kubeconfig to new one: %v", err)
	}
	kubeconfig := kubeconfigFp.Name()

	opts.Kubeconfig = kubeconfig

	// Run general validation on options.
	if err := opts.Parse(); err != nil {
		t.Fatalf("failed to parse options: %v", err)
	}

	// Make the process asynchronous.
	go internal.RunKubeStateMetricsWrapper(opts)

	// Wait for port 8080 to come up.
	err = wait.PollUntilContextTimeout(context.TODO(), 1*time.Second, 20*time.Second, true, func(ctx context.Context) (bool, error) {
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
		t.Fatalf("failed to wait for port 8080 to come up for the first time: %v", err)
	}

	// Modify config to trigger hot reload.
	err = exec.Command("kubectl", "config", "set-cluster", "ksm-hot-reload-kubeconfig-test", "--kubeconfig", kubeconfig).Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to modify kubeconfig: %v", err)
	}

	// Revert kubeconfig to original one.
	defer func() {
		err := exec.Command("kubectl", "config", "delete-cluster", "ksm-hot-reload-kubeconfig-test", "--kubeconfig", kubeconfig).Run() //nolint:gosec
		if err != nil {
			t.Fatalf("failed to revert kubeconfig: %v", err)
		}
	}()

	// Wait for new kubeconfig to be reloaded.
	time.Sleep(5 * time.Second)

	// Wait for port 8080 to come up.
	ch := make(chan bool, 1)
	err = wait.PollUntilContextTimeout(context.TODO(), 1*time.Second, 20*time.Second, true, func(ctx context.Context) (bool, error) {
		conn, err := net.Dial("tcp", "localhost:8080")
		if err != nil {
			return false, nil
		}
		err = conn.Close()
		if err != nil {
			return false, err
		}
		// Indicate that the test has passed.
		ch <- true
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed to wait for port 8080 to come up after restarting the process: %v", err)
	}

	// Wait for process to exit.
	select {
	case <-ch:
		t.Log("test passed successfully")
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for test to pass, check the logs for more info")
	}
}
