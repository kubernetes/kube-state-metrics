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

package customresourcestate

import (
	"fmt"
	"strings"
)

// pathValueExtractor implements path-based value extraction.
type pathValueExtractor struct {
	valueFrom     valuePath            // The path to the value to extract
	path          valuePath            // The path of compiledCommon
	labelFromPath map[string]valuePath // Labels to extract from paths
	nilIsZero     bool                 // Whether nil should be treated as zero
	labelFromKey  string               // Label name for map keys
}

func (s *pathValueExtractor) extractValues(v interface{}) (result []eachValue, errs []error) {
	onError := func(err error) {
		errs = append(errs, fmt.Errorf("%s: %v", s.path, err))
	}

	switch iter := v.(type) {
	case map[string]interface{}:
		for key, it := range iter {
			// TODO: Handle multi-length valueFrom paths (https://github.com/kubernetes/kube-state-metrics/pull/1958#discussion_r1099243161).
			// TODO: This could be potentially represented in CEL.
			// Try to deduce `valueFrom`'s value from the current element.
			var ev *eachValue
			var err error
			var didResolveValueFrom bool
			// `valueFrom` will ultimately be rendered into a string and sent to the fallback in place, which also expects a string.
			// So we'll do the same and operate on the string representation of `valueFrom`'s value.
			sValueFrom := s.valueFrom.String()
			// No comma means we're looking at a unit-length path (in an array).
			if !strings.Contains(sValueFrom, ",") &&
				sValueFrom[0] == '[' && sValueFrom[len(sValueFrom)-1] == ']' &&
				// "[...]" and not "[]".
				len(sValueFrom) > 2 {
				extractedValueFrom := sValueFrom[1 : len(sValueFrom)-1]
				if key == extractedValueFrom {
					gotFloat, err := toFloat64(it, s.nilIsZero)
					if err != nil {
						onError(fmt.Errorf("[%s]: %w", key, err))
						continue
					}
					labels := make(map[string]string)
					ev = &eachValue{
						Labels: labels,
						Value:  gotFloat,
					}
					didResolveValueFrom = true
				}
			}
			// Fallback to the regular path resolution, if we didn't manage to resolve `valueFrom`'s value.
			if !didResolveValueFrom {
				ev, err = s.extractValue(it)
				if ev == nil {
					continue
				}
			}
			if err != nil {
				onError(fmt.Errorf("[%s]: %w", key, err))
				continue
			}
			if _, ok := ev.Labels[s.labelFromKey]; ok {
				onError(fmt.Errorf("labelFromKey (%s) generated labels conflict with labelsFromPath, consider renaming it", s.labelFromKey))
				continue
			}
			if key != "" && s.labelFromKey != "" {
				ev.Labels[s.labelFromKey] = key
			}
			addPathLabels(it, s.labelFromPath, ev.Labels)
			// Evaluate path from parent's context as well (search w.r.t. the root element, not just specific fields).
			addPathLabels(v, s.labelFromPath, ev.Labels)
			result = append(result, *ev)
		}
	case []interface{}:
		for i, it := range iter {
			value, err := s.extractValue(it)
			if err != nil {
				onError(fmt.Errorf("[%d]: %w", i, err))
				continue
			}
			if value == nil {
				continue
			}
			addPathLabels(it, s.labelFromPath, value.Labels)
			result = append(result, *value)
		}
	default:
		value, err := s.extractValue(v)
		if err != nil {
			onError(err)
			break
		}
		if value == nil {
			break
		}
		addPathLabels(v, s.labelFromPath, value.Labels)
		result = append(result, *value)
	}
	return
}

func (s *pathValueExtractor) extractValue(it interface{}) (*eachValue, error) {
	labels := make(map[string]string)
	got := s.valueFrom.Get(it)
	// If `valueFrom` was not resolved, respect `NilIsZero` and return.
	if got == nil {
		if s.nilIsZero {
			return &eachValue{
				Labels: labels,
				Value:  0,
			}, nil
		}
		// no it means no iterables were passed down, meaning that the path resolution never happened
		if it == nil {
			return nil, fmt.Errorf("got nil while resolving path")
		}
		// Don't error if there was not a type-casting issue (`toFloat64`).
		return nil, nil
	}
	value, err := toFloat64(got, s.nilIsZero)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", s.valueFrom, err)
	}
	return &eachValue{
		Labels: labels,
		Value:  value,
	}, nil
}
