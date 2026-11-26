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
	"bufio"
	"context"
	"net"
	"net/http"
	"os"
	"strings"
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
	err = wait.PollUntilContextTimeout(context.TODO(), 1*time.Second, 20*time.Second, true, func(_ context.Context) (bool, error) {
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

	// The empty file is loaded at startup.
	before := waitForConfigMetrics(t, "the initial options config file to be loaded", func(m map[string]string) bool {
		return m["kube_state_metrics_last_config_reload_successful"] == "1" && m["kube_state_metrics_config_hash"] != ""
	})

	// A valid change restarts kube-state-metrics with the new file.
	if err := os.WriteFile(opts.Config, []byte("enable_gzip_encoding: true\n"), 0600 /* rw------- */); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	waitForConfigMetrics(t, "the changed options config file to be loaded", func(m map[string]string) bool {
		return m["kube_state_metrics_last_config_reload_successful"] == "1" &&
			m["kube_state_metrics_config_hash"] != "" && m["kube_state_metrics_config_hash"] != before["kube_state_metrics_config_hash"]
	})

	// An unknown key is rejected: the running instance keeps serving and the
	// reload is reported as failed.
	if err := os.WriteFile(opts.Config, []byte(`foo: "bar"`), 0600 /* rw------- */); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	waitForConfigMetrics(t, "the rejected reload to be reported", func(m map[string]string) bool {
		return m["kube_state_metrics_last_config_reload_successful"] == "0"
	})
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Fatalf("expected the running instance to keep serving after a rejected reload: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
}

// waitForConfigMetrics polls the telemetry endpoint until the options config
// file's self metrics, keyed by metric name, satisfy cond, and returns them.
func waitForConfigMetrics(t *testing.T, what string, cond func(map[string]string) bool) map[string]string {
	t.Helper()
	var got map[string]string
	err := wait.PollUntilContextTimeout(context.TODO(), 1*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		m, err := configMetrics(ctx)
		if err != nil {
			// The telemetry server is down while kube-state-metrics restarts.
			return false, nil
		}
		got = m
		return cond(m), nil
	})
	if err != nil {
		t.Fatalf("failed to wait for %s, last config metrics %v: %v", what, got, err)
	}
	return got
}

// configMetrics returns the values of the options config file's self metrics,
// keyed by metric name.
func configMetrics(ctx context.Context) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8081/metrics", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	values := map[string]string{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || !strings.Contains(line, `type="config"`) {
			continue
		}
		name, rest, ok := strings.Cut(line, "{")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		values[name] = fields[len(fields)-1]
	}
	return values, scanner.Err()
}
