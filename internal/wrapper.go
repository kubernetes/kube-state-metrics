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
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"k8s.io/klog/v2"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	"k8s.io/kube-state-metrics/v2/pkg/app"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

// RunKubeStateMetricsWrapper is a wrapper around KSM, delegated to the root command.
func RunKubeStateMetricsWrapper(opts *options.Options) {

	KSMRunOrDie := func(ctx context.Context) {
		if err := app.RunKubeStateMetricsWrapper(ctx, opts); err != nil {
			klog.ErrorS(err, "Failed to run kube-state-metrics")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	if file := options.GetConfigFile(*opts); file != "" {
		cfgViper := viper.New()
		cfgViper.SetConfigType("yaml")
		cfgViper.SetConfigFile(file)
		var cfgViperReadInConfigErr error
		if cfgViperReadInConfigErr = cfgViper.ReadInConfig(); cfgViperReadInConfigErr != nil {
			if errors.Is(cfgViperReadInConfigErr, viper.ConfigFileNotFoundError{}) {
				klog.ErrorS(cfgViperReadInConfigErr, "Options configuration file not found at startup", "file", file)
			} else if _, isNotExisterr := os.Stat(filepath.Clean(file)); isNotExisterr != nil {
				// TODO: Remove this check once viper.ConfigFileNotFoundError is working as expected, see this issue -
				// https://github.com/spf13/viper/issues/1783
				klog.ErrorS(isNotExisterr, "Options configuration file not found at startup", "file", file)
			} else {
				klog.ErrorS(cfgViperReadInConfigErr, "Error reading options configuration file", "file", file)
			}
			if !opts.ContinueWithoutConfig {
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		}
		cfgViper.OnConfigChange(func(e fsnotify.Event) {
			klog.InfoS("Changes detected", "name", e.Name)
			cancel()
			// Wait for the ports to be released.
			<-time.After(3 * time.Second)
			ctx, cancel = context.WithCancel(context.Background())
			go KSMRunOrDie(ctx)
		})
		cfgViper.WatchConfig()

		if cfgViperReadInConfigErr == nil {
			// Merge configFile values with opts so we get the CustomResourceConfigFile from config as well
			configFile, err := os.ReadFile(filepath.Clean(file))
			if err != nil {
				klog.ErrorS(err, "failed to read options configuration file", "file", file)
			}

			yaml.Unmarshal(configFile, opts)
		}
	}
	if opts.CustomResourceConfigFile != "" {
		crcViper := viper.New()
		crcViper.SetConfigType("yaml")
		crcViper.SetConfigFile(opts.CustomResourceConfigFile)
		if cfgViperReadInConfigErr := crcViper.ReadInConfig(); cfgViperReadInConfigErr != nil {
			if errors.Is(cfgViperReadInConfigErr, viper.ConfigFileNotFoundError{}) {
				klog.ErrorS(cfgViperReadInConfigErr, "Custom resource configuration file not found at startup", "file", opts.CustomResourceConfigFile)
			} else if _, isNotExisterr := os.Stat(filepath.Clean(opts.CustomResourceConfigFile)); isNotExisterr != nil {
				// Adding this check in addition to the above since viper.ConfigFileNotFoundError is not working as expected due to this issue -
				// https://github.com/spf13/viper/issues/1783
				klog.ErrorS(isNotExisterr, "Custom resource configuration file not found at startup", "file", opts.CustomResourceConfigFile)
			} else {
				klog.ErrorS(cfgViperReadInConfigErr, "Error reading Custom resource configuration file", "file", opts.CustomResourceConfigFile)
			}
			if !opts.ContinueWithoutCustomResourceConfigFile {
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		}
		crcViper.OnConfigChange(func(e fsnotify.Event) {
			klog.InfoS("Changes detected", "name", e.Name)
			cancel()
			// Wait for the ports to be released.
			<-time.After(3 * time.Second)
			ctx, cancel = context.WithCancel(context.Background())
			go KSMRunOrDie(ctx)
		})
		crcViper.WatchConfig()
	}
	if opts.Kubeconfig != "" {
		kubecfgViper := viper.New()
		kubecfgViper.SetConfigType("yaml")
		kubecfgViper.SetConfigFile(opts.Kubeconfig)
		if err := kubecfgViper.ReadInConfig(); err != nil {
			if errors.Is(err, viper.ConfigFileNotFoundError{}) {
				klog.ErrorS(err, "kubeconfig file not found", "file", opts.Kubeconfig)
			} else {
				klog.ErrorS(err, "Error reading kubeconfig file", "file", opts.Kubeconfig)
			}
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		kubecfgViper.OnConfigChange(func(e fsnotify.Event) {
			klog.InfoS("Changes detected", "name", e.Name)
			cancel()
			// Wait for the ports to be released.
			<-time.After(3 * time.Second)
			ctx, cancel = context.WithCancel(context.Background())
			go KSMRunOrDie(ctx)
		})
		kubecfgViper.WatchConfig()
	}
	klog.InfoS("Starting kube-state-metrics")
	KSMRunOrDie(ctx)
	select {}
}
