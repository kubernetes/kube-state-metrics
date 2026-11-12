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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	yaml "sigs.k8s.io/yaml/goyaml.v3"

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
	"k8s.io/kube-state-metrics/v2/pkg/options"
	"k8s.io/kube-state-metrics/v2/pkg/runconfig"
	"k8s.io/kube-state-metrics/v2/pkg/util"
)

type prepareMode int

const (
	prepareStartup prepareMode = iota
	prepareReload
)

// PrepareRunOptionsForReload merges and validates on a clone of base; base is unchanged.
// On validation failure the returned opts may still be set so watches can retry after file correction.
func PrepareRunOptionsForReload(base *options.Options) (*options.Options, error) {
	opts, prepared, err := prepareRun(base, prepareReload)
	if err != nil {
		return returnPrepareErr(opts, err)
	}
	opts.Prepared = prepared
	return opts, nil
}

func returnPrepareErr(opts *options.Options, err error) (*options.Options, error) {
	if opts != nil {
		return opts, err
	}
	return nil, err
}

func prepareRun(base *options.Options, mode prepareMode) (*options.Options, *runconfig.Prepared, error) {
	var opts *options.Options
	switch mode {
	case prepareReload:
		opts = base.Clone()
	default:
		opts = base
	}

	prepared := &runconfig.Prepared{
		Apiserver: opts.Apiserver,
	}

	configPath := options.GetConfigFile(*opts)
	var configFile []byte

	if configPath != "" {
		var err error
		configFile, err = os.ReadFile(filepath.Clean(configPath))
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if !opts.ContinueWithoutConfig {
					return nil, nil, fmt.Errorf("options config file not found: %w", err)
				}
			} else {
				return nil, nil, fmt.Errorf("read options config file: %w", err)
			}
		} else {
			if err := yaml.Unmarshal(configFile, opts); err != nil {
				return returnPrepareErrMode(opts, mode, fmt.Errorf("unmarshal options config file: %w", err))
			}
			opts = configureResourcesAndMetrics(opts, configFile)
			prepared.OptionsConfigPath = configPath
			prepared.OptionsConfig = append([]byte(nil), configFile...)
		}
	}

	if opts.CustomResourceConfigFile == "" {
		opts.CustomResourceConfigFile = opts.CustomResourceConfigFileDeprecated
	}
	if opts.StoreSyncTimeout < 0 {
		return returnPrepareErrMode(opts, mode, fmt.Errorf("value for --store-sync-timeout=%s must be equal to or greater than 0", opts.StoreSyncTimeout))
	}

	prepared.Apiserver = opts.Apiserver
	prepared.KubeconfigPath = opts.Kubeconfig

	if err := validateAndCaptureCRS(opts, prepared, mode); err != nil {
		return returnPrepareErrMode(opts, mode, err)
	}
	if err := validateKubeconfig(opts, prepared, mode); err != nil {
		return returnPrepareErrMode(opts, mode, err)
	}
	if err := validateRunConfigForStartup(opts); err != nil {
		return returnPrepareErrMode(opts, mode, err)
	}

	if mode == prepareReload {
		return opts, prepared.Clone(), nil
	}
	return opts, prepared, nil
}

func returnPrepareErrMode(opts *options.Options, mode prepareMode, err error) (*options.Options, *runconfig.Prepared, error) {
	if mode == prepareReload && opts != nil {
		return opts, nil, err
	}
	return nil, nil, err
}

func validateAndCaptureCRS(opts *options.Options, prepared *runconfig.Prepared, mode prepareMode) error {
	if s := strings.TrimSpace(opts.CustomResourceConfig); s != "" {
		if err := decodeCRSConfig(yaml.NewDecoder(strings.NewReader(s))); err != nil {
			return err
		}
		prepared.CRSInline = s
		return nil
	}
	file := opts.CustomResourceConfigFile
	if file == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		if opts.ContinueWithoutCustomResourceConfigFile && os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read custom resource config file: %w", err)
	}
	if err := decodeCRSConfig(yaml.NewDecoder(bytes.NewReader(data))); err != nil {
		return err
	}
	prepared.CRSFilePath = file
	if mode == prepareReload {
		prepared.CRSFileContent = append([]byte(nil), data...)
	}
	return nil
}

func decodeCRSConfig(decoder customresourcestate.ConfigDecoder) error {
	var cfg customresourcestate.Metrics
	if err := decoder.Decode(&cfg); err != nil {
		return fmt.Errorf("failed to parse Custom Resource State metrics: %w", err)
	}
	return nil
}

func validateKubeconfig(opts *options.Options, prepared *runconfig.Prepared, mode prepareMode) error {
	path := opts.Kubeconfig
	if path == "" {
		_, err := util.BuildRESTConfig(opts.Apiserver, "", nil)
		return err
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("read kubeconfig: %w", err)
	}
	var content []byte
	if mode == prepareReload {
		content = make([]byte, len(data))
		copy(content, data)
		prepared.KubeconfigContent = content
	}
	_, err = util.BuildRESTConfig(opts.Apiserver, path, content)
	return err
}
