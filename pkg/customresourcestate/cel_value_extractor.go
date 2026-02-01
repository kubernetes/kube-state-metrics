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

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"

	ksmcel "k8s.io/kube-state-metrics/v2/pkg/cel"
	"k8s.io/kube-state-metrics/v2/pkg/cel/library"
)

// celValueExtractor implements CEL-based value extraction.
type celValueExtractor struct {
	program       cel.Program
	expr          string
	path          valuePath
	labelFromPath map[string]valuePath
	nilIsZero     bool
}

// newCELValueExtractor creates a new CEL-based value extractor by compiling the given expression.
// The CEL expression has access to:
// - value: the value at the resolved path (any type)
//
// The CEL expression can return either:
// - A value directly.
// - A CELResult created via CELResult(value, labels) to return a value with additional labels.
func newCELValueExtractor(expr string, path valuePath, labelFromPath map[string]valuePath, nilIsZero bool) (*celValueExtractor, error) {
	if expr == "" {
		return nil, fmt.Errorf("CEL expression cannot be empty")
	}

	env, err := cel.NewEnv(
		library.KSM(),
		cel.Variable("value", cel.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile CEL expression %q: %w", expr, issues.Err())
	}

	program, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program from expression %q: %w", expr, err)
	}

	return &celValueExtractor{
		program:       program,
		expr:          expr,
		path:          path,
		labelFromPath: labelFromPath,
		nilIsZero:     nilIsZero,
	}, nil
}

func (s *celValueExtractor) extractValues(v interface{}) (result []eachValue, errs []error) {
	onError := func(err error) {
		errs = append(errs, fmt.Errorf("%s: %v", s.path, err))
	}

	values, err := s.extractValue(v)
	if err != nil {
		onError(err)
		return
	}
	if values == nil {
		return
	}

	for _, value := range values {

		labels := make(map[string]string)
		addPathLabels(v, s.labelFromPath, labels)
		// Apply AdditionalLabels last to avoid overwriting
		for k, v := range value.Labels {
			labels[k] = v
		}
		value.Labels = labels

		result = append(result, value)
	}

	return result, errs
}

func (s *celValueExtractor) extractValue(v interface{}) ([]eachValue, error) {
	celRes, err := s.evaluateCEL(v)
	if err != nil {
		return nil, err
	}

	switch celRes.Type() {
	case nil:
		// Handle nil values
		if s.nilIsZero {
			return []eachValue{
				{
					Labels: make(map[string]string),
					Value:  0,
				},
			}, nil
		}
		return nil, nil

	case types.ListType:
		// returned a list of values e.g. via `map` function
		list := celRes.Value().([]ref.Val)
		eachValues := make([]eachValue, 0, len(list))
		for _, elem := range list {
			ev, err := s.processVal(elem)
			if err != nil {
				return nil, err
			}
			if ev == nil {
				continue
			}
			eachValues = append(eachValues, *ev)
		}

		return eachValues, nil

	default:
		ev, err := s.processVal(celRes)
		if err != nil {
			return nil, err
		}
		if ev == nil {
			return nil, nil
		}
		return []eachValue{*ev}, nil
	}
}

func (s *celValueExtractor) processVal(val ref.Val) (*eachValue, error) {
	unwrapped := val.Value()
	switch v := unwrapped.(type) {
	case nil:
		// Handle nil values
		if s.nilIsZero {
			return &eachValue{
				Labels: make(map[string]string),
				Value:  0,
			}, nil
		}
		return nil, nil

	case *ksmcel.CELResult:
		// Value returned as CELResult
		value, err := toFloat64(v.Val, s.nilIsZero)
		if err != nil {
			return nil, err
		}

		return &eachValue{
			Labels: v.AdditionalLabels,
			Value:  value,
		}, nil
	default:
		// Value returned directly
		value, err := toFloat64(v, s.nilIsZero)
		if err != nil {
			return nil, err
		}

		return &eachValue{
			Labels: make(map[string]string),
			Value:  value,
		}, nil
	}
}

// evaluateCEL evaluates the CEL expression with the given context.
func (s *celValueExtractor) evaluateCEL(value interface{}) (ref.Val, error) {
	// Prepare input vars
	vars := map[string]interface{}{
		"value": value,
	}

	// Evaluate the CEL expression
	result, _, err := s.program.Eval(vars)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate CEL expression %q: %w", s.expr, err)
	}

	if types.IsUnknown(result) || types.IsError(result) {
		return nil, fmt.Errorf("CEL expression returned error: %v", result)
	}

	return result, nil
}
