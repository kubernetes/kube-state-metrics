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
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	"k8s.io/kube-state-metrics/v2/pkg/app"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

// RunKubeStateMetricsWrapper is a wrapper around KSM, delegated to the root command.
func RunKubeStateMetricsWrapper(opts *options.Options) {
	// Snapshot the command line options before any config file is merged in.
	// RunKubeStateMetrics decodes the config file onto the options it is given,
	// so every run, including each reload, has to start from a fresh copy.
	// Otherwise keys removed from the config file keep their previous values.
	baseOpts := opts.Clone()

	KSMRunOrDie := func(ctx context.Context) {
		if err := app.RunKubeStateMetricsWrapper(ctx, baseOpts.Clone()); err != nil {
			klog.ErrorS(err, "Failed to run kube-state-metrics")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The custom resource config file and the kubeconfig can be set by the
	// options config file, so the files in effect can change with each reload.
	var restart func(fsnotify.Event)
	crcWatch := &movableWatch{name: "Custom resource configuration file"}
	kubeconfigWatch := &movableWatch{name: "kubeconfig file"}

	// Every watched file gets its own watcher, and each watcher runs its
	// callback on its own goroutine. They all restart KSM through the same ctx
	// and cancel, and a ConfigMap holding more than one watched file updates
	// them together, so the callbacks have to take turns. Otherwise one cancel
	// is overwritten before it is used, the instance it belonged to is never
	// stopped, and two of them contend for the listen ports.
	var reloadMu sync.Mutex
	restart = func(e fsnotify.Event) {
		reloadMu.Lock()
		defer reloadMu.Unlock()

		klog.InfoS("Changes detected", "name", e.Name)
		// Follow the files the restarted run will use. An options config file
		// that cannot be read keeps the current watchers; the run waits for it
		// to be fixed and the fix triggers another restart.
		if next, ok := reloadedOptions(baseOpts); ok {
			crcWatch.watch(next.CustomResourceConfigFile, restart)
			kubeconfigWatch.watch(next.Kubeconfig, restart)
		}
		cancel()
		// Wait for the ports to be released.
		<-time.After(3 * time.Second)
		ctx, cancel = context.WithCancel(context.Background())
		go KSMRunOrDie(ctx)
	}

	if file := options.GetConfigFile(*opts); file != "" {
		// Merge configFile values with opts so we get the CustomResourceConfigFile from config as well
		if configFile, err := os.ReadFile(filepath.Clean(file)); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				klog.ErrorS(err, "Options configuration file not found at startup", "file", file)
			} else {
				klog.ErrorS(err, "Error reading options configuration file", "file", file)
			}
			if !opts.ContinueWithoutConfig {
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		} else if err := yaml.Unmarshal(configFile, opts); err != nil {
			klog.ErrorS(err, "failed to unmarshal options configuration file", "file", file)
			if !opts.ContinueWithoutConfig {
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
			// TODO: Remove deprecated custom_resource_config_file support in KSM 2.21.
		} else if opts.CustomResourceConfigFileDeprecated != "" {
			klog.Warning("The 'custom_resource_config_file' config key is deprecated and will be removed in a future release; use 'custom_resource_state_config_file' instead")
			if opts.CustomResourceConfigFile == "" {
				opts.CustomResourceConfigFile = opts.CustomResourceConfigFileDeprecated
			}
		}
		if _, err := watchFile(file, restart); err != nil {
			klog.ErrorS(err, "Failed to watch options configuration file", "file", file)
		}
	}
	if opts.CustomResourceConfigFile != "" {
		if err := readYAMLFile(opts.CustomResourceConfigFile); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				klog.ErrorS(err, "Custom resource configuration file not found at startup", "file", opts.CustomResourceConfigFile)
			} else {
				klog.ErrorS(err, "Error reading Custom resource configuration file", "file", opts.CustomResourceConfigFile)
			}
			if !opts.ContinueWithoutCustomResourceConfigFile {
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		}
		// The options file watcher is already running and may restart first.
		reloadMu.Lock()
		crcWatch.watch(opts.CustomResourceConfigFile, restart)
		reloadMu.Unlock()
	}
	if opts.Kubeconfig != "" {
		if err := readYAMLFile(opts.Kubeconfig); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				klog.ErrorS(err, "kubeconfig file not found", "file", opts.Kubeconfig)
			} else {
				klog.ErrorS(err, "Error reading kubeconfig file", "file", opts.Kubeconfig)
			}
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		reloadMu.Lock()
		kubeconfigWatch.watch(opts.Kubeconfig, restart)
		reloadMu.Unlock()
	}
	klog.InfoS("Starting kube-state-metrics")
	KSMRunOrDie(ctx)
	select {}
}

// movableWatch watches one file whose path can change between runs.
type movableWatch struct {
	name string
	file string
	stop func()
}

// watch moves the watch to file, unless it already follows it. An empty file
// stops the watch.
func (w *movableWatch) watch(file string, onChange func(fsnotify.Event)) {
	if file == w.file {
		return
	}
	if w.stop != nil {
		w.stop()
		w.stop = nil
	}
	w.file = file
	if file == "" {
		return
	}
	stop, err := watchFile(file, onChange)
	if err != nil {
		klog.ErrorS(err, "Failed to watch "+w.name, "file", file)
		return
	}
	w.stop = stop
}

// reloadedOptions returns the options a run starting from baseOpts uses, with
// the options config file merged in. It returns false when the options config
// file cannot be read or parsed.
func reloadedOptions(baseOpts *options.Options) (*options.Options, bool) {
	opts := baseOpts.Clone()
	if file := options.GetConfigFile(*opts); file != "" {
		configFile, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			return nil, false
		}
		if err := yaml.Unmarshal(configFile, opts); err != nil {
			return nil, false
		}
	}
	// TODO: Remove deprecated custom_resource_config_file support in KSM 2.21.
	if opts.CustomResourceConfigFile == "" {
		opts.CustomResourceConfigFile = opts.CustomResourceConfigFileDeprecated
	}
	return opts, true
}
