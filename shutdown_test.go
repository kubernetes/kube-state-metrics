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
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// runMainEnv makes the test binary run main instead of the tests, so the
// shutdown tests can run kube-state-metrics as a child process without
// building it separately.
const runMainEnv = "KSM_TEST_RUN_MAIN"

func TestMain(m *testing.M) {
	if os.Getenv(runMainEnv) == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

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

// ksmProcess is kube-state-metrics running as a child process.
type ksmProcess struct {
	cmd    *exec.Cmd
	port   int
	output *syncBuffer
	// done carries the Wait result.
	done chan error
}

// startKSM runs kube-state-metrics against apiserver. The process is killed
// and waited for when the test ends, if it is still running.
func startKSM(t *testing.T, apiserver string) *ksmProcess {
	t.Helper()
	if testing.Short() {
		t.Skip("runs kube-state-metrics as a child process")
	}
	p := &ksmProcess{port: freePort(t), output: &syncBuffer{}, done: make(chan error, 1)}
	p.cmd = exec.Command(os.Args[0], //nolint:gosec // the test binary itself, see TestMain
		"--apiserver="+apiserver,
		"--resources=configmaps",
		"--host=127.0.0.1",
		"--port="+strconv.Itoa(p.port),
		"--telemetry-host=127.0.0.1",
		"--telemetry-port="+strconv.Itoa(freePort(t)),
	)
	p.cmd.Env = append(os.Environ(), runMainEnv+"=1")
	p.cmd.Stdout = p.output
	p.cmd.Stderr = p.output
	if err := p.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	// exited is closed once Wait has returned, so cleanup stays synchronized
	// with the process even after the test has consumed the result from done.
	exited := make(chan struct{})
	go func() {
		p.done <- p.cmd.Wait()
		close(exited)
	}()
	t.Cleanup(func() {
		select {
		case <-exited:
		default:
			_ = p.cmd.Process.Kill()
			<-exited
		}
	})
	return p
}

// terminate sends SIGTERM and expects the process to exit with code 0 within timeout.
func (p *ksmProcess) terminate(t *testing.T, timeout time.Duration) {
	t.Helper()
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-p.done:
		if err != nil {
			t.Fatalf("kube-state-metrics did not exit cleanly on SIGTERM: %v\n%s", err, p.output)
		}
	case <-time.After(timeout):
		t.Fatalf("kube-state-metrics did not exit within %s of SIGTERM\n%s", timeout, p.output)
	}
}

// TestShutdownOnSIGTERM runs kube-state-metrics against a minimal fake API
// server and checks that SIGTERM, which Kubernetes sends to stop a container,
// stops it through the graceful shutdown path with exit code 0.
func TestShutdownOnSIGTERM(t *testing.T) {
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
	// Registered before startKSM, so the process is stopped before the server
	// waits for its outstanding requests.
	t.Cleanup(apiserver.Close)

	p := startKSM(t, apiserver.URL)

	metricsURL := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(p.port)) + "/metrics"
	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := http.Get(metricsURL) //nolint:gosec // local test server
		if err == nil {
			resp.Body.Close()
			break
		}
		select {
		case err := <-p.done:
			t.Fatalf("kube-state-metrics exited before serving: %v\n%s", err, p.output)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("metrics server did not start: %v\n%s", err, p.output)
		}
		time.Sleep(100 * time.Millisecond)
	}

	p.terminate(t, 15*time.Second)
	if !strings.Contains(p.output.String(), "shutting down") {
		t.Fatalf("kube-state-metrics did not go through the shutdown path\n%s", p.output)
	}
}

// TestShutdownOnSIGTERMDuringStartup checks that SIGTERM interrupts the startup
// request to an unresponsive API server, so the process exits promptly with code
// 0 instead of waiting for the client timeout, which can outlast the default
// termination grace period.
func TestShutdownOnSIGTERMDuringStartup(t *testing.T) {
	versionRequested := make(chan struct{})
	var once sync.Once
	apiserver := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/version" {
			once.Do(func() { close(versionRequested) })
		}
		// Never answer: hold the request until the client gives up.
		<-r.Context().Done()
	}))
	// Registered before startKSM, so the process is stopped before the server
	// waits for its outstanding requests.
	t.Cleanup(apiserver.Close)

	p := startKSM(t, apiserver.URL)

	select {
	case <-versionRequested:
	case err := <-p.done:
		t.Fatalf("kube-state-metrics exited before contacting the API server: %v\n%s", err, p.output)
	case <-time.After(30 * time.Second):
		t.Fatalf("kube-state-metrics did not contact the API server\n%s", p.output)
	}

	p.terminate(t, 10*time.Second)
}
