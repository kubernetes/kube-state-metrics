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

type ksmRunner func(context.Context, *options.Options)

type wrapperLifecycleHooks struct {
	// holdingReloadMuBeforeFirstLaunch runs while reloadMu is held, before the initial
	// goroutine launch. proceedWithFirstLaunch must be called to continue startup.
	holdingReloadMuBeforeFirstLaunch func(restart func(fsnotify.Event), proceedWithFirstLaunch func())
	restartPortReleaseDelay          time.Duration
	// done, when non-nil, causes runWrapperLifecycle to shut down once closed.
	done chan struct{}
}

func runWrapperLifecycle(opts *options.Options, run ksmRunner, hooks *wrapperLifecycleHooks) {
	baseOpts := opts.Clone()

	var restart func(fsnotify.Event)
	optionsWatch := &movableWatch{name: "options configuration file"}
	crcWatch := &movableWatch{name: "Custom resource configuration file"}
	kubeconfigWatch := &movableWatch{name: "kubeconfig file"}

	var reloadMu sync.Mutex
	var shuttingDown bool
	runCtx, runCancel := context.WithCancel(context.Background())

	restartDelay := 3 * time.Second
	if hooks != nil && hooks.restartPortReleaseDelay > 0 {
		restartDelay = hooks.restartPortReleaseDelay
	}

	restart = func(e fsnotify.Event) {
		reloadMu.Lock()
		defer reloadMu.Unlock()
		if shuttingDown {
			return
		}

		klog.InfoS("Changes detected", "name", e.Name)
		snapshot, err := app.PrepareRunOptionsForReload(baseOpts)
		registerReloadInputWatches(snapshot, optionsWatch, crcWatch, kubeconfigWatch, restart)
		if err != nil {
			klog.ErrorS(err, "Invalid reload configuration; keeping the current instance running")
			return
		}
		runCancel()
		timer := time.NewTimer(restartDelay)
		defer timer.Stop()
		<-timer.C
		if shuttingDown {
			return
		}
		runCtx, runCancel = context.WithCancel(context.Background())
		go run(runCtx, snapshot)
	}

	shutdown := func() {
		reloadMu.Lock()
		defer reloadMu.Unlock()
		if shuttingDown {
			return
		}
		shuttingDown = true
		optionsWatch.watch("", restart)
		crcWatch.watch("", restart)
		kubeconfigWatch.watch("", restart)
		runCancel()
	}

	reloadMu.Lock()

	if file := options.GetConfigFile(*opts); file != "" {
		if configFile, err := os.ReadFile(filepath.Clean(file)); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				klog.ErrorS(err, "Options configuration file not found at startup", "file", file)
			} else {
				klog.ErrorS(err, "Error reading options configuration file", "file", file)
			}
			if !opts.ContinueWithoutConfig {
				reloadMu.Unlock()
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		} else if err := yaml.Unmarshal(configFile, opts); err != nil {
			klog.ErrorS(err, "failed to unmarshal options configuration file", "file", file)
			if !opts.ContinueWithoutConfig {
				reloadMu.Unlock()
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		} else if opts.CustomResourceConfigFileDeprecated != "" {
			klog.Warning("The 'custom_resource_config_file' config key is deprecated and will be removed in a future release; use 'custom_resource_state_config_file' instead")
			if opts.CustomResourceConfigFile == "" {
				opts.CustomResourceConfigFile = opts.CustomResourceConfigFileDeprecated
			}
		}
		optionsWatch.watch(file, restart)
	}
	if opts.CustomResourceConfigFile != "" {
		if err := readYAMLFile(opts.CustomResourceConfigFile); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				klog.ErrorS(err, "Custom resource configuration file not found at startup", "file", opts.CustomResourceConfigFile)
			} else {
				klog.ErrorS(err, "Error reading Custom resource configuration file", "file", opts.CustomResourceConfigFile)
			}
			if !opts.ContinueWithoutCustomResourceConfigFile {
				reloadMu.Unlock()
				klog.FlushAndExit(klog.ExitFlushTimeout, 1)
			}
		}
		crcWatch.watch(opts.CustomResourceConfigFile, restart)
	}
	if opts.Kubeconfig != "" {
		if err := readYAMLFile(opts.Kubeconfig); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				klog.ErrorS(err, "kubeconfig file not found", "file", opts.Kubeconfig)
			} else {
				klog.ErrorS(err, "Error reading kubeconfig file", "file", opts.Kubeconfig)
			}
			reloadMu.Unlock()
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		kubeconfigWatch.watch(opts.Kubeconfig, restart)
	}
	klog.InfoS("Starting kube-state-metrics")
	startup, err := app.PrepareRunOptionsForReload(baseOpts)
	registerReloadInputWatches(startup, optionsWatch, crcWatch, kubeconfigWatch, restart)
	if err != nil {
		reloadMu.Unlock()
		klog.ErrorS(err, "Invalid startup configuration")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
	if hooks != nil && hooks.holdingReloadMuBeforeFirstLaunch != nil {
		proceed := make(chan struct{})
		hooks.holdingReloadMuBeforeFirstLaunch(restart, func() { close(proceed) })
		<-proceed
	}
	go run(runCtx, startup)
	reloadMu.Unlock()
	if hooks != nil && hooks.done != nil {
		<-hooks.done
		shutdown()
		return
	}
	select {}
}

func defaultKSMRunner(ctx context.Context, runOpts *options.Options) {
	if err := app.RunKubeStateMetricsWrapper(ctx, runOpts); err != nil {
		klog.ErrorS(err, "Failed to run kube-state-metrics")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
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

// reloadPreflight merges and validates the configuration a new run would use.
func registerReloadInputWatches(snapshot *options.Options, optionsWatch, crcWatch, kubeconfigWatch *movableWatch, restart func(fsnotify.Event)) {
	if snapshot == nil {
		return
	}
	if file := options.GetConfigFile(*snapshot); file != "" {
		optionsWatch.watch(file, restart)
	}
	crcWatch.watch(snapshot.CustomResourceConfigFile, restart)
	kubeconfigWatch.watch(snapshot.Kubeconfig, restart)
}

func reloadedOptions(baseOpts *options.Options) (*options.Options, bool) {
	opts, err := app.PrepareRunOptionsForReload(baseOpts)
	if err != nil {
		return opts, false
	}
	return opts, true
}
