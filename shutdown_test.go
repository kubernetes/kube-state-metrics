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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
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

// startKSM runs kube-state-metrics against apiserver, with args added to the
// command line. The process is killed and waited for when the test ends, if it
// is still running.
func startKSM(t *testing.T, apiserver string, args ...string) *ksmProcess {
	t.Helper()
	if testing.Short() {
		t.Skip("runs kube-state-metrics as a child process")
	}
	p := &ksmProcess{port: freePort(t), output: &syncBuffer{}, done: make(chan error, 1)}
	p.cmd = exec.Command(os.Args[0], append([]string{ //nolint:gosec // the test binary itself, see TestMain
		"--apiserver=" + apiserver,
		"--resources=configmaps",
		"--host=127.0.0.1",
		"--port=" + strconv.Itoa(p.port),
		"--telemetry-host=127.0.0.1",
		"--telemetry-port=" + strconv.Itoa(freePort(t)),
	}, args...)...)
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
	p.sigterm(t)
	p.waitExit(t, timeout)
}

func (p *ksmProcess) sigterm(t *testing.T) {
	t.Helper()
	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
}

// waitExit expects the process to exit with code 0 within timeout.
func (p *ksmProcess) waitExit(t *testing.T, timeout time.Duration) {
	t.Helper()
	select {
	case err := <-p.done:
		if err != nil {
			t.Fatalf("kube-state-metrics did not exit cleanly on SIGTERM: %v\n%s", err, p.output)
		}
	case <-time.After(timeout):
		t.Fatalf("kube-state-metrics did not exit within %s of SIGTERM\n%s", timeout, p.output)
	}
}

// testConfigMapMetric is a series kube-state-metrics serves for the ConfigMap
// the fake API server lists.
const testConfigMapMetric = `kube_configmap_info{namespace="default",configmap="shutdown-test"} 1`

// fakeAPIServer serves just enough of the Kubernetes API for kube-state-metrics
// to start and fill its ConfigMap store with the ConfigMap shutdown-test and
// padding more ConfigMaps with long names, through either a watch list or a
// list. Watches are then held open until the client goes away.
func fakeAPIServer(t *testing.T, padding int) *httptest.Server {
	t.Helper()
	configMaps := []string{`{"kind":"ConfigMap","apiVersion":"v1","metadata":{"name":"shutdown-test","namespace":"default","uid":"shutdown-test","resourceVersion":"1"}}`}
	for i := range padding {
		configMaps = append(configMaps, fmt.Sprintf(`{"kind":"ConfigMap","apiVersion":"v1","metadata":{"name":"%0200d","namespace":"default","uid":"%[1]d","resourceVersion":"1"}}`, i))
	}
	apiserver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query()
		switch {
		case r.URL.Path == "/version":
			fmt.Fprint(w, `{"major":"1","minor":"35","gitVersion":"v1.35.0"}`)
		case r.URL.Path == "/api/v1/configmaps" && query.Get("watch") == "":
			fmt.Fprint(w, `{"kind":"ConfigMapList","apiVersion":"v1","metadata":{"resourceVersion":"1"},"items":[`+strings.Join(configMaps, ",")+`]}`)
		case r.URL.Path == "/api/v1/configmaps":
			if query.Get("sendInitialEvents") == "true" {
				for _, configMap := range configMaps {
					fmt.Fprintln(w, `{"type":"ADDED","object":`+configMap+`}`)
				}
				fmt.Fprintln(w, `{"type":"BOOKMARK","object":{"kind":"ConfigMap","apiVersion":"v1",`+
					`"metadata":{"resourceVersion":"1","annotations":{"k8s.io/initial-events-end":"true"}}}}`)
			}
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	// Registered before startKSM, so the process is stopped before the server
	// waits for its outstanding requests.
	t.Cleanup(apiserver.Close)
	return apiserver
}

func (p *ksmProcess) metricsAddr() string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(p.port))
}

// scrapeClient bounds each scrape, including reading the body, so a stalled
// server fails the test instead of blocking past the polling deadline.
var scrapeClient = &http.Client{Timeout: 10 * time.Second}

// scrape fetches /metrics and returns an error unless it is served in full,
// including the series of the fake API server's ConfigMap.
func (p *ksmProcess) scrape() error {
	resp, err := scrapeClient.Get("http://" + p.metricsAddr() + "/metrics") //nolint:gosec // local test server
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkScrape(resp)
}

func checkScrape(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got status %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), testConfigMapMetric) {
		return fmt.Errorf("missing %s in:\n%s", testConfigMapMetric, body)
	}
	return nil
}

// waitFor polls cond until it returns nil, failing the test if the process
// exits or timeout passes first.
func (p *ksmProcess) waitFor(t *testing.T, what string, timeout time.Duration, cond func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		err := cond()
		if err == nil {
			return
		}
		select {
		case err := <-p.done:
			t.Fatalf("kube-state-metrics exited while waiting for %s: %v\n%s", what, err, p.output)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s: %v\n%s", what, err, p.output)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// waitServing waits until the metrics server serves the ConfigMap metrics.
func (p *ksmProcess) waitServing(t *testing.T) {
	t.Helper()
	p.waitFor(t, "metrics to be served", 30*time.Second, p.scrape)
}

// waitLog waits until the process has logged msg.
func (p *ksmProcess) waitLog(t *testing.T, msg string, timeout time.Duration) {
	t.Helper()
	p.waitFor(t, fmt.Sprintf("log %q", msg), timeout, func() error {
		if !strings.Contains(p.output.String(), msg) {
			return errors.New("not logged yet")
		}
		return nil
	})
}

// TestShutdownOnSIGTERM runs kube-state-metrics against a minimal fake API
// server and checks that SIGTERM, which Kubernetes sends to stop a container,
// stops it through the graceful shutdown path with exit code 0. Without
// --shutdown-delay it exits promptly.
func TestShutdownOnSIGTERM(t *testing.T) {
	p := startKSM(t, fakeAPIServer(t, 0).URL)
	p.waitServing(t)

	p.terminate(t, 5*time.Second)
	if !strings.Contains(p.output.String(), "shutting down") {
		t.Fatalf("kube-state-metrics did not go through the shutdown path\n%s", p.output)
	}
	if strings.Contains(p.output.String(), "Delaying shutdown") {
		t.Fatalf("kube-state-metrics delayed shutdown without --shutdown-delay\n%s", p.output)
	}
}

// TestShutdownDelay checks that with --shutdown-delay kube-state-metrics keeps
// serving full metrics for the delay after SIGTERM, so scrapes sent before
// service discovery has caught up still succeed, and then exits with code 0.
func TestShutdownDelay(t *testing.T) {
	const delay = 2 * time.Second
	p := startKSM(t, fakeAPIServer(t, 0).URL, "--shutdown-delay="+delay.String())
	p.waitServing(t)

	p.sigterm(t)
	signalled := time.Now()
	p.waitLog(t, "Delaying shutdown", delay/2)
	// Scrape throughout most of the delay; every scrape must succeed in full.
	for time.Since(signalled) < delay*3/4 {
		if err := p.scrape(); err != nil {
			t.Fatalf("scrape %s after SIGTERM failed: %v\n%s", time.Since(signalled), err, p.output)
		}
		time.Sleep(50 * time.Millisecond)
	}

	p.waitExit(t, delay+5*time.Second)
	if elapsed := time.Since(signalled); elapsed < delay {
		t.Fatalf("kube-state-metrics exited %s after SIGTERM, before the %s shutdown delay", elapsed, delay)
	}
}

// TestShutdownDrainsInFlightScrape checks that a scrape in flight when the
// servers shut down is still answered in full: the stores are only stopped
// after the servers have drained.
func TestShutdownDrainsInFlightScrape(t *testing.T) {
	// Enough ConfigMaps for a response of several MB, more than the socket
	// buffers hold, so the handler is blocked writing it until it is read.
	p := startKSM(t, fakeAPIServer(t, 20000).URL)
	p.waitServing(t)

	conn, err := (&net.Dialer{Control: func(_, _ string, c syscall.RawConn) error {
		var err error
		if cerr := c.Control(func(fd uintptr) {
			err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4096)
		}); cerr != nil {
			return cerr
		}
		return err
	}}).Dial("tcp", p.metricsAddr())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := fmt.Fprintf(conn, "GET /metrics HTTP/1.1\r\nHost: %s\r\n\r\n", p.metricsAddr()); err != nil {
		t.Fatal(err)
	}
	// Reading the start of the response shows the handler is running; it
	// then blocks on the unread rest.
	r := bufio.NewReader(conn)
	if _, err := r.Peek(1); err != nil {
		t.Fatal(err)
	}

	p.sigterm(t)
	// Once the listener is closed, the server is shutting down and waits for
	// the scrape above.
	p.waitFor(t, "the metrics listener to close", 3*time.Second, func() error {
		c, err := net.Dial("tcp", p.metricsAddr())
		if err != nil {
			return nil
		}
		c.Close()
		return errors.New("still accepting connections")
	})

	resp, err := http.ReadResponse(r, nil)
	if err != nil {
		t.Fatalf("in-flight scrape failed: %v\n%s", err, p.output)
	}
	defer resp.Body.Close()
	if err := checkScrape(resp); err != nil {
		t.Fatalf("in-flight scrape failed: %v\n%s", err, p.output)
	}

	p.waitExit(t, 5*time.Second)
}

// TestShutdownDelaySkippedOnReload checks that a config reload restarts
// kube-state-metrics without waiting for the shutdown delay. A run that did
// wait would keep the ports, so the restarted run could not listen on them.
func TestShutdownDelaySkippedOnReload(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(config, []byte("shutdown_delay: 1m\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p := startKSM(t, fakeAPIServer(t, 0).URL, "--config="+config)
	p.waitServing(t)

	if err := os.WriteFile(config, []byte("shutdown_delay: 1m\n# reload\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	p.waitLog(t, "Restarting kube-state-metrics", 15*time.Second)
	p.waitServing(t)
	if strings.Contains(p.output.String(), "Delaying shutdown") {
		t.Fatalf("config reload waited for the shutdown delay\n%s", p.output)
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
