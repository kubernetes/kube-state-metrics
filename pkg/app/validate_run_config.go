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
	"fmt"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	"k8s.io/kube-state-metrics/v2/pkg/optin"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func validateRunConfigForStartup(opts *options.Options) error {
	b := store.NewBuilder()
	names := enabledResourceNames(opts)
	if len(names) > 0 {
		if err := b.WithEnabledResources(names); err != nil {
			return err
		}
	}
	if err := b.WithAllowAnnotations(opts.AnnotationsAllowList); err != nil {
		return fmt.Errorf("failed to set up annotations allowlist: %w", err)
	}
	if err := b.WithAllowLabels(opts.LabelsAllowList); err != nil {
		return fmt.Errorf("failed to set up labels allowlist: %w", err)
	}

	allowDenyList, err := allowdenylist.New(opts.MetricAllowlist, opts.MetricDenylist)
	if err != nil {
		return err
	}
	if err := allowDenyList.Parse(); err != nil {
		return fmt.Errorf("error initializing the allowdeny list: %w", err)
	}

	if _, err := optin.NewMetricFamilyFilter(opts.MetricOptInList); err != nil {
		return fmt.Errorf("error initializing the opt-in metric list: %w", err)
	}
	if err := opts.Validate(); err != nil {
		return err
	}
	return nil
}

func enabledResourceNames(opts *options.Options) []string {
	switch {
	case len(opts.Resources) == 0 && !opts.CustomResourcesOnly:
		return options.DefaultResources.AsSlice()
	case opts.CustomResourcesOnly:
		return nil
	default:
		return opts.Resources.AsSlice()
	}
}
