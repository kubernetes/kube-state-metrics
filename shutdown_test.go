//go:build !windows

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

package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// syncBuffer is a bytes.Buffer that is safe to read while the process writes to it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// TestShutdownOnSIGTERM builds the binary, runs it against a minimal fake API
// server, and checks that SIGTERM, which Kubernetes sends to stop a container,
// stops it through the graceful shutdown path with exit code 0.
func TestShutdownOnSIGTERM(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the binary")
	}

	// Only /version has to answer for kube-state-metrics to start; the
	// reflectors retry on the 404s.
	apiserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"major":"1","minor":"35","gitVersion":"v1.35.0"}`)
	}))
	defer apiserver.Close()

	bin := filepath.Join(t.TempDir(), "kube-state-metrics")
	build := exec.Command("go", "build", "-o", bin, ".") //nolint:gosec // test binary path
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	port := freePort(t)
	telemetryPort := freePort(t)
	var output syncBuffer
	cmd := exec.Command(bin, //nolint:gosec // the binary built above
		"--apiserver="+apiserver.URL,
		"--resources=configmaps",
		"--host=127.0.0.1",
		"--port="+strconv.Itoa(port),
		"--telemetry-host=127.0.0.1",
		"--telemetry-port="+strconv.Itoa(telemetryPort),
	)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// done carries the Wait result for the assertions; exited is closed once
	// Wait has returned, so cleanup stays synchronized with the process even
	// after the test has consumed the result from done.
	done := make(chan error, 1)
	exited := make(chan struct{})
	go func() {
		done <- cmd.Wait()
		close(exited)
	}()
	defer func() {
		select {
		case <-exited:
		default:
			_ = cmd.Process.Kill()
			<-exited
		}
	}()

	metricsURL := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/metrics"
	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := http.Get(metricsURL) //nolint:gosec // local test server
		if err == nil {
			resp.Body.Close()
			break
		}
		select {
		case err := <-done:
			t.Fatalf("kube-state-metrics exited before serving: %v\n%s", err, output.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("metrics server did not start: %v\n%s", err, output.String())
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				t.Fatalf("kube-state-metrics did not exit cleanly on SIGTERM: %v\n%s", err, output.String())
			}
			t.Fatal(err)
		}
	case <-time.After(15 * time.Second):
		t.Fatalf("kube-state-metrics did not exit within 15s of SIGTERM\n%s", output.String())
	}

	if !strings.Contains(output.String(), "shutting down") {
		t.Fatalf("kube-state-metrics did not go through the shutdown path\n%s", output.String())
	}
}

// TestShutdownOnSIGTERMDuringStartup checks that SIGTERM interrupts the startup
// request to an unresponsive API server, so the process exits promptly with code
// 0 instead of waiting for the client timeout, which can outlast the default
// termination grace period.
func TestShutdownOnSIGTERMDuringStartup(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the binary")
	}

	versionRequested := make(chan struct{})
	var once sync.Once
	apiserver := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/version" {
			once.Do(func() { close(versionRequested) })
		}
		// Never answer: hold the request until the client gives up.
		<-r.Context().Done()
	}))
	defer apiserver.Close()

	bin := filepath.Join(t.TempDir(), "kube-state-metrics")
	build := exec.Command("go", "build", "-o", bin, ".") //nolint:gosec // test binary path
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	var output syncBuffer
	cmd := exec.Command(bin, //nolint:gosec // the binary built above
		"--apiserver="+apiserver.URL,
		"--resources=configmaps",
		"--host=127.0.0.1",
		"--port="+strconv.Itoa(freePort(t)),
		"--telemetry-host=127.0.0.1",
		"--telemetry-port="+strconv.Itoa(freePort(t)),
	)
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	exited := make(chan struct{})
	go func() {
		done <- cmd.Wait()
		close(exited)
	}()
	defer func() {
		select {
		case <-exited:
		default:
			_ = cmd.Process.Kill()
			<-exited
		}
	}()

	select {
	case <-versionRequested:
	case err := <-done:
		t.Fatalf("kube-state-metrics exited before contacting the API server: %v\n%s", err, output.String())
	case <-time.After(30 * time.Second):
		t.Fatalf("kube-state-metrics did not contact the API server\n%s", output.String())
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("kube-state-metrics did not exit cleanly on SIGTERM during startup: %v\n%s", err, output.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("kube-state-metrics did not exit within 10s of SIGTERM during startup\n%s", output.String())
	}
}
