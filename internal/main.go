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
	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/pkg/options"
)

// Main parses the flags and runs kube-state-metrics. It backs both the main
// package at the module root and cmd/kube-state-metrics.
func Main() {
	opts := options.NewOptions()
	cmd := options.InitCommand
	cmd.Run = func(_ *cobra.Command, _ []string) {
		RunKubeStateMetricsWrapper(opts)
	}
	opts.AddFlags(cmd)
	// Parse only returns on error or shutdown; options are validated in PreRunE.
	if err := opts.Parse(); err != nil {
		klog.ErrorS(err, "Running kube-state-metrics failed")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}
}
