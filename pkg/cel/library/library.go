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

package library

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	ksmcel "k8s.io/kube-state-metrics/v2/pkg/cel"
)

// KSM provides CEL custom functions for kube-state-metrics.
//
// # CELResult
//
// Converts a value and labels map into a CELResult type that can return both
// a metric value and additional labels.
//
//	CELResult(<any>, <map<string, string>>) <CELResult>
//
// Examples:
//
//	CELResult(100.0, {}) // returns CELResult with value 100.0 and no additional labels
//	CELResult(42, {'severity': 'high'}) // returns CELResult with value 42 and label severity=high
//	CELResult(double(value) * 10.0, {'multiplied': 'true'}) // returns CELResult with computed value and label
func KSM() cel.EnvOption {
	return cel.Lib(ksmLib)
}

var ksmLib = &ksm{}

type ksm struct{}

func (*ksm) LibraryName() string {
	return "kubestatemetrics"
}

func (*ksm) declarations() map[string][]cel.FunctionOpt {
	return ksmLibraryDecls
}

func (*ksm) Types() []*cel.Type {
	return []*cel.Type{ksmcel.CELResultObjectType}
}

var ksmLibraryDecls = map[string][]cel.FunctionOpt{
	"CELResult": {
		cel.Overload("celresult_any_map",
			[]*cel.Type{cel.DynType, cel.MapType(cel.StringType, cel.StringType)},
			ksmcel.CELResultObjectType,
			cel.BinaryBinding(celResultConstructor)),
	},
}

func (*ksm) CompileOptions() []cel.EnvOption {
	options := []cel.EnvOption{cel.Types(ksmcel.CELResultObjectType)}
	for name, overloads := range ksmLibraryDecls {
		options = append(options, cel.Function(name, overloads...))
	}
	options = append(options, cel.Container("kubestatemetrics"))
	return options
}

func (*ksm) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{}
}

// celResultConstructor is the implementation of the CELResult constructor function.
// It takes a value and a map of labels and returns a CELResult.
func celResultConstructor(val, labels ref.Val) ref.Val {
	celResult := &ksmcel.CELResult{
		Val:              val.Value(),
		AdditionalLabels: make(map[string]string),
	}

	// Extract labels from the map
	if labelsMap, ok := labels.(traits.Mapper); ok {
		it := labelsMap.Iterator()
		for it.HasNext() == types.True {
			key := it.Next()
			value := labelsMap.Get(key)
			if keyStr, ok := key.(types.String); ok {
				if valStr, ok := value.(types.String); ok {
					celResult.AdditionalLabels[string(keyStr)] = string(valStr)
				}
			}
		}
	}

	// CELResult implements ref.Val, so we can return it directly
	return celResult
}
