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

package internal

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/pkg/app"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

// RunKubeStateMetricsWrapper is a wrapper around KSM, delegated to the root command.
func RunKubeStateMetricsWrapper(opts *options.Options) {
	var factories []customresource.RegistryFactory
	if config, set := resolveCustomResourceConfig(opts); set {
		crf, err := customresourcestate.FromConfig(config)
		if err != nil {
			klog.ErrorS(err, "Parsing from Custom Resource State Metrics file failed")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		factories = append(factories, crf...)
	}

	KSMRunOrDie := func(ctx context.Context) {
		if err := app.RunKubeStateMetricsWrapper(ctx, opts, factories...); err != nil {
			klog.ErrorS(err, "Failed to run kube-state-metrics")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	if file := options.GetConfigFile(*opts); file != "" {
		viper.SetConfigType("yaml")
		viper.SetConfigFile(file)
		if err := viper.ReadInConfig(); err != nil {
			if errors.Is(err, viper.ConfigFileNotFoundError{}) {
				klog.ErrorS(err, "Options configuration file not found", "file", file)
			} else {
				klog.ErrorS(err, "Error reading options configuration file", "file", file)
			}
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		viper.OnConfigChange(func(e fsnotify.Event) {
			klog.Infof("Changes detected: %s\n", e.Name)
			cancel()
			// Wait for the ports to be released.
			<-time.After(3 * time.Second)
			ctx, cancel = context.WithCancel(context.Background())
			go KSMRunOrDie(ctx)
		})
		viper.WatchConfig()
	}
	klog.Infoln("Starting kube-state-metrics")
	KSMRunOrDie(ctx)
	select {}
}

func resolveCustomResourceConfig(opts *options.Options) (customresourcestate.ConfigDecoder, bool) {
	if s := opts.CustomResourceConfig; s != "" {
		return yaml.NewDecoder(strings.NewReader(s)), true
	}
	if file := opts.CustomResourceConfigFile; file != "" {
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			klog.ErrorS(err, "Custom Resource State Metrics file could not be opened")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		return yaml.NewDecoder(f), true
	}
	return nil, false
}
