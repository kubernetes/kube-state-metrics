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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func applyPreparedRunConfig(opts *options.Options, configHash, configSuccess, configSuccessTime *prometheus.GaugeVec) error {
	p := opts.Prepared
	if p == nil {
		return fmt.Errorf("prepared configuration missing")
	}
	opts.CaptureStoreSyncTimeoutBaseline()
	var fileTimeout *time.Duration
	if len(p.OptionsConfig) > 0 {
		parsed, err := storeSyncTimeoutFromFile(p.OptionsConfig)
		if err != nil {
			return err
		}
		fileTimeout = parsed
	}
	opts.ResolveStoreSyncTimeout(fileTimeout)
	if opts.StoreSyncTimeout < 0 {
		return fmt.Errorf("value for --store-sync-timeout=%s must be equal to or greater than 0", opts.StoreSyncTimeout)
	}
	if p.OptionsConfigPath != "" && len(p.OptionsConfig) > 0 {
		clean := filepath.Clean(p.OptionsConfigPath)
		configSuccess.WithLabelValues("config", clean).Set(1)
		configSuccessTime.WithLabelValues("config", clean).SetToCurrentTime()
		configHash.WithLabelValues("config", clean).Set(md5HashAsMetricValue(p.OptionsConfig))
	}
	return nil
}

func kubeconfigContent(opts *options.Options) []byte {
	if opts.Prepared == nil {
		return nil
	}
	return opts.Prepared.KubeconfigContent
}

func resolveCustomResourceConfig(opts *options.Options) (customresourcestate.ConfigDecoder, error) {
	if p := opts.Prepared; p != nil {
		if s := strings.TrimSpace(p.CRSInline); s != "" {
			return yaml.NewDecoder(strings.NewReader(s)), nil
		}
		if len(p.CRSFileContent) > 0 {
			return yaml.NewDecoder(bytes.NewReader(p.CRSFileContent)), nil
		}
		return nil, nil
	}
	return resolveCustomResourceConfigFromDisk(opts)
}

func resolveCustomResourceConfigFromDisk(opts *options.Options) (customresourcestate.ConfigDecoder, error) {
	if s := opts.CustomResourceConfig; s != "" {
		return yaml.NewDecoder(strings.NewReader(s)), nil
	}
	if file := opts.CustomResourceConfigFile; file != "" {
		data, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			if opts.ContinueWithoutCustomResourceConfigFile && os.IsNotExist(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("unable to open Custom Resource State Metrics file: %w", err)
		}
		return yaml.NewDecoder(bytes.NewReader(data)), nil
	}
	return nil, nil
}

func applyCustomResourceConfigMetrics(opts *options.Options, configHash, configSuccess, configSuccessTime *prometheus.GaugeVec) error {
	if opts.Prepared != nil {
		p := opts.Prepared
		if p.CRSInline != "" {
			return nil
		}
		if len(p.CRSFileContent) > 0 && p.CRSFilePath != "" {
			clean := filepath.Clean(p.CRSFilePath)
			configSuccess.WithLabelValues("customresourceconfig", clean).Set(1)
			configSuccessTime.WithLabelValues("customresourceconfig", clean).SetToCurrentTime()
			configHash.WithLabelValues("customresourceconfig", clean).Set(md5HashAsMetricValue(p.CRSFileContent))
		}
		return nil
	}
	if opts.CustomResourceConfigFile != "" {
		if _, err := os.Stat(filepath.Clean(opts.CustomResourceConfigFile)); err != nil {
			klog.ErrorS(err, "encountered error while processing the file", "file", opts.CustomResourceConfigFile)
		} else {
			crcFile, err := os.ReadFile(filepath.Clean(opts.CustomResourceConfigFile))
			if err != nil {
				return fmt.Errorf("failed to read custom resource config file: %w", err)
			}
			configSuccess.WithLabelValues("customresourceconfig", filepath.Clean(opts.CustomResourceConfigFile)).Set(1)
			configSuccessTime.WithLabelValues("customresourceconfig", filepath.Clean(opts.CustomResourceConfigFile)).SetToCurrentTime()
			configHash.WithLabelValues("customresourceconfig", filepath.Clean(opts.CustomResourceConfigFile)).Set(md5HashAsMetricValue(crcFile))
		}
	}
	return nil
}
