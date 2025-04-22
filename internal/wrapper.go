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
		if err := cfgViper.ReadInConfig(); err != nil {
			if errors.Is(err, viper.ConfigFileNotFoundError{}) {
				klog.ErrorS(err, "Options configuration file not found", "file", file)
			} else {
				klog.ErrorS(err, "Error reading options configuration file", "file", file)
			}
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
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

		// Merge configFile values with opts so we get the CustomResourceConfigFile from config as well
		configFile, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			klog.ErrorS(err, "failed to read options configuration file", "file", file)
		}

		yaml.Unmarshal(configFile, opts)
	}
	if opts.CustomResourceConfigFile != "" {
		crcViper := viper.New()
		crcViper.SetConfigType("yaml")
		crcViper.SetConfigFile(opts.CustomResourceConfigFile)
		if err := crcViper.ReadInConfig(); err != nil {
			if errors.Is(err, viper.ConfigFileNotFoundError{}) {
				klog.ErrorS(err, "Custom resource configuration file not found", "file", opts.CustomResourceConfigFile)
			} else {
				klog.ErrorS(err, "Error reading Custom resource configuration file", "file", opts.CustomResourceConfigFile)
			}
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
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
