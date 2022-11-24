/*
Copyright 2022 The Kubernetes Authors All rights reserved.

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
	"net"
	"os"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/kube-state-metrics/v2/internal"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestConfigHotReload(t *testing.T) {

	// Initialise options.
	opts := options.NewOptions()
	cmd := options.InitCommand
	opts.AddFlags(cmd)

	// Create testdata.
	f, err := os.CreateTemp("", "config")
	if err != nil {
		t.Fatal(err)
	}

	// Delete artefacts.
	defer func() {
		err := os.Remove(opts.Config)
		if err != nil {
			t.Fatalf("failed to remove config file: %v", err)
		}
	}()

	// Populate options.
	opts.Config = f.Name()

	// Assume $HOME is always defined.
	opts.Kubeconfig = os.Getenv("HOME") + "/.kube/config"

	// Run general validation on options.
	if err := opts.Parse(); err != nil {
		t.Fatalf("failed to parse options: %v", err)
	}

	// Make the process asynchronous.
	go internal.RunKubeStateMetricsWrapper(opts)

	// Wait for port 8080 to come up.
	err = wait.PollImmediate(1*time.Second, 20*time.Second, func() (bool, error) {
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
	config := `foo: "bar"`
	err = os.WriteFile(opts.Config, []byte(config), 0600 /* rw------- */)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Wait for port 8080 to come up.
	ch := make(chan bool, 1)
	err = wait.PollImmediate(1*time.Second, 20*time.Second, func() (bool, error) {
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
		t.Fatalf("failed to wait for port 8080 to come up after restarting the process: %v", err)
	}

	// Indicate that the test has passed.
	ch <- true

	// Wait for process to exit.
	select {
	case <-ch:
		t.Log("test passed successfully")
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for test to pass, check the logs for more info")
	}
}
