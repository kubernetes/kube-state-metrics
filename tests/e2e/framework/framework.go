/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

package framework

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

const (
	epHealthz = "/healthz"
	epMetrics = "/metrics"
)

// The Framework stores a pointer to the KSMClient
type Framework struct {
	KsmClient *KSMClient
}

// New returns a new Framework given the kube-state-metrics service URLs.
// It delegates the url validation errs to NewKSMClient func.
func New(ksmHTTPMetricsURL, ksmTelemetryURL string) (*Framework, error) {
	ksmClient, err := NewKSMClient(ksmHTTPMetricsURL, ksmTelemetryURL)
	if err != nil {
		return nil, err
	}

	return &Framework{
		KsmClient: ksmClient,
	}, nil
}

// The KSMClient is the Kubernetes State Metric client.
type KSMClient struct {
	httpMetricsEndpoint *url.URL
	telemetryEndpoint   *url.URL
	client              *http.Client
}

// NewKSMClient retrieves a new KSMClient the kube-state-metrics service URLs.
// In case of error parsing the provided addresses, it returns an error.
func NewKSMClient(ksmHTTPMetricsAddress, ksmTelemetryAddress string) (*KSMClient, error) {
	ksmHTTPMetricsURL, err := validateURL(ksmHTTPMetricsAddress)
	if err != nil {
		return nil, err
	}
	ksmTelemetryURL, err := validateURL(ksmTelemetryAddress)
	if err != nil {
		return nil, err
	}

	return &KSMClient{
		httpMetricsEndpoint: ksmHTTPMetricsURL,
		telemetryEndpoint:   ksmTelemetryURL,
		client:              &http.Client{},
	}, nil
}

func validateURL(address string) (*url.URL, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, err
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u, nil
}

// IsHealthz makes a request to the /healthz endpoint to get the health status.
func (k *KSMClient) IsHealthz() (bool, error) {
	p := path.Join(k.httpMetricsEndpoint.Path, epHealthz)

	u := *k.httpMetricsEndpoint
	u.Path = p

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err
	}

	resp, err := k.client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	return true, nil
}

func (k *KSMClient) writeMetrics(endpoint *url.URL, w io.Writer) error {
	if endpoint == nil {
		return errors.New("endpoint is nil")
	}

	u := *endpoint
	u.Path = path.Join(endpoint.Path, epMetrics)

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := k.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	io.Copy(w, resp.Body)

	return nil
}

// Metrics makes a request to the /metrics endpoint on the "http-metrics" port,
// and writes its content to the writer w.
func (k *KSMClient) Metrics(w io.Writer) error {
	return k.writeMetrics(k.httpMetricsEndpoint, w)
}

// TelemetryMetrics makes a request to the /metrics endpoint on the "telemetry" port,
// and writes its content to the writer w.
func (k *KSMClient) TelemetryMetrics(w io.Writer) error {
	return k.writeMetrics(k.telemetryEndpoint, w)
}

// ParseMetrics uses a prometheus TextParser to parse metrics, given a function
// that fetches and writes metrics.
func (f *Framework) ParseMetrics(metrics func(io.Writer) error) (map[string]*dto.MetricFamily, error) {
	buf := &bytes.Buffer{}
	err := metrics(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	parser := expfmt.NewTextParser(model.UTF8Validation)
	return parser.TextToMetricFamilies(buf)
}

// HasMetricWithLabels checks if a metric with specific labels exists.
func (f *Framework) HasMetricWithLabels(metricName string, labels map[string]string) (bool, error) {
	metricFamilies, err := f.ParseMetrics(f.KsmClient.Metrics)
	if err != nil {
		return false, fmt.Errorf("failed to parse metrics: %w", err)
	}

	family, ok := metricFamilies[metricName]
	if !ok {
		return false, nil
	}

	for _, metric := range family.Metric {
		matchCount := 0
		for _, label := range metric.Label {
			if expectedValue, exists := labels[label.GetName()]; exists {
				if label.GetValue() == expectedValue {
					matchCount++
				}
			}
		}
		if matchCount == len(labels) {
			return true, nil
		}
	}

	return false, nil
}
