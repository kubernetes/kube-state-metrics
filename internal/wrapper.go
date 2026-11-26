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
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog/v2"

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

	runOrDie := func(run func() error) {
		if err := run(); err != nil {
			klog.ErrorS(err, "Failed to run kube-state-metrics")
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
	}

	// SIGTERM is what Kubernetes sends to stop a container. Cancelling the
	// runs on it shuts their servers down and drains in-flight scrapes,
	// instead of the default signal action killing the process.
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// runs tracks the current run, so shutdown can wait for it to finish.
	var runs sync.WaitGroup
	// A run cancelled for a reload is told so by its cancel cause, so it skips
	// the shutdown delay that applies on termination.
	ctx, cancel := context.WithCancelCause(sigCtx)
	defer cancel(nil)

	// The custom resource config file and the kubeconfig can be set by the
	// options config file, so the files in effect can change with each reload.
	var onOptionsChange, onFileChange func(fsnotify.Event)
	crcWatch := &movableWatch{name: "Custom resource configuration file"}
	kubeconfigWatch := &movableWatch{name: "kubeconfig file"}

	// Every watched file gets its own watcher, and each watcher runs its
	// callback on its own goroutine. They all restart KSM through the same ctx
	// and cancel, and a ConfigMap holding more than one watched file updates
	// them together, so the callbacks have to take turns. Otherwise one cancel
	// is overwritten before it is used, the instance it belonged to is never
	// stopped, and two of them contend for the listen ports.
	var reloadMu sync.Mutex
	// current is the options config file the running instance was started
	// with, guarded by reloadMu.
	var current []byte
	status := &configStatus{}
	restart := func(e fsnotify.Event, optionsChanged bool) {
		reloadMu.Lock()
		defer reloadMu.Unlock()
		if sigCtx.Err() != nil {
			// Shutting down: do not start another run.
			return
		}

		klog.InfoS("Changes detected", "name", e.Name)
		// The new run gets the options config file as it was checked here, as
		// the file can change again before the run starts.
		configFile, next, err := loadConfigFile(baseOpts)
		rejected := err != nil
		if rejected {
			file := options.GetConfigFile(*baseOpts)
			status.reject()
			// Stopping the running instance takes the metrics endpoints down
			// until the new one has started, so keep it if only the options
			// config file changed.
			if optionsChanged {
				klog.ErrorS(err, "Invalid options configuration file, keeping the running configuration", "file", file)
				return
			}
			// A file the running instance uses changed, such as a rotated
			// kubeconfig, so restart with the options config file it runs with.
			// The watchers already follow the files of that configuration.
			klog.ErrorS(err, "Invalid options configuration file, restarting with the running configuration", "file", file)
			configFile = current
		} else {
			current = configFile
			// Follow the files the restarted run will use.
			crcWatch.watch(next.CustomResourceConfigFile, onFileChange)
			kubeconfigWatch.watch(next.Kubeconfig, onFileChange)
		}
		cancel(app.ErrConfigReload)
		// Wait for the ports to be released.
		select {
		case <-time.After(3 * time.Second):
		case <-sigCtx.Done():
			return
		}
		klog.InfoS("Restarting kube-state-metrics, metrics will be reset")
		ctx, cancel = context.WithCancelCause(sigCtx)
		runCtx := ctx
		runOpts := []app.RunOption{app.WithOptionsConfigFile(configFile), app.WithConfigReloadReporter(status.newRun(rejected))}
		runs.Go(func() {
			runOrDie(func() error {
				return app.RunKubeStateMetricsWrapper(runCtx, baseOpts.Clone(), runOpts...)
			})
		})
	}
	onOptionsChange = func(e fsnotify.Event) { restart(e, true) }
	onFileChange = func(e fsnotify.Event) { restart(e, false) }

	// Registered before any watcher starts, so that a run started by a reload
	// replaces the initial run's reporter rather than the other way around.
	initialReporter := status.newRun(false)
	// The initial run uses the options config file as read here, nil if it
	// does not exist or cannot be read. A file that cannot be decoded is still
	// passed on, and the run waits for it to be fixed.
	var configFile []byte

	if file := options.GetConfigFile(*opts); file != "" {
		// Merge configFile values with opts so we get the CustomResourceConfigFile from config as well
		contents, merged, err := loadConfigFile(opts)
		configFile = contents
		switch {
		case err != nil:
			klog.ErrorS(err, "Invalid options configuration file", "file", file)
		case contents == nil:
			klog.ErrorS(fs.ErrNotExist, "Options configuration file not found at startup", "file", file)
		default:
			opts = merged
		}
		if (err != nil || contents == nil) && !opts.ContinueWithoutConfig {
			klog.FlushAndExit(klog.ExitFlushTimeout, 1)
		}
		reloadMu.Lock()
		current = configFile
		reloadMu.Unlock()
		if _, err := watchFile(file, onOptionsChange); err != nil {
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
		crcWatch.watch(opts.CustomResourceConfigFile, onFileChange)
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
		kubeconfigWatch.watch(opts.Kubeconfig, onFileChange)
		reloadMu.Unlock()
	}
	klog.InfoS("Starting kube-state-metrics")
	reloadMu.Lock()
	runCtx := ctx
	runs.Go(func() {
		runOrDie(func() error {
			return app.RunKubeStateMetricsWrapper(runCtx, baseOpts.Clone(), app.WithOptionsConfigFile(configFile), app.WithConfigReloadReporter(initialReporter))
		})
	})
	reloadMu.Unlock()

	<-sigCtx.Done()
	// Restore the default action, so a second signal stops the process at once.
	stop()
	klog.InfoS("Received termination signal, shutting down kube-state-metrics")
	// Wait for a restart in progress to return. It sees sigCtx is done and
	// starts no new run, so runs.Wait does not race a runs.Go.
	reloadMu.Lock()
	defer reloadMu.Unlock()
	runs.Wait()
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

// configStatus routes rejected reloads of the options config file to the run
// that serves kube_state_metrics_last_config_reload_successful. A reload can be
// rejected while the run started by the previous reload is still setting up its
// metrics and has yet to report the file it was given as loaded, so a rejection
// is kept and applied once that run reports in.
type configStatus struct {
	mu       sync.Mutex
	run      int
	rejected bool
	report   func()
}

// newRun registers a new run, whose options config file has been rejected if
// rejected is set, and returns the callback the run reports in with. Runs
// registered before it can no longer report in.
func (s *configStatus) newRun(rejected bool) func(reportRejected func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.run++
	run := s.run
	s.rejected = rejected
	s.report = nil
	return func(reportRejected func()) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if run != s.run {
			return
		}
		s.report = reportRejected
		if s.rejected {
			reportRejected()
		}
	}
}

// reject marks the options config file as rejected in the latest run, now or
// once it reports in.
func (s *configStatus) reject() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rejected = true
	if s.report != nil {
		s.report()
	}
}

// loadConfigFile reads the options config file of baseOpts and returns its
// contents along with the options a run starting from baseOpts uses. The
// contents are nil if no options config file is set or it does not exist; the
// run then starts without it. A file that exists but cannot be read or decoded
// is an error, returned with the contents that were read, if any, and no
// options.
func loadConfigFile(baseOpts *options.Options) ([]byte, *options.Options, error) {
	opts := baseOpts.Clone()
	file := options.GetConfigFile(*opts)
	if file == "" {
		return nil, opts, nil
	}
	configFile, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, opts, nil
		}
		return nil, nil, err
	}
	// Decoded onto a copy, as a file that is rejected can still have set some
	// options, such as continue_without_config.
	if err := options.UnmarshalConfigFile(configFile, opts); err != nil {
		return configFile, nil, err
	}
	return configFile, opts, nil
}
