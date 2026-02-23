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
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"

	ksmcel "k8s.io/kube-state-metrics/v2/pkg/cel"
)

// KSM provides CEL custom functions for kube-state-metrics.
//
// # WithLabels
//
// Wraps a metric value with additional labels.
//
//	WithLabels(<any>, <map<string, string>>) <WithLabels>
//
// Examples:
//
//	WithLabels(100.0, {}) // returns value 100.0 with no additional labels
//	WithLabels(42, {'severity': 'high'}) // returns value 42 with label severity=high
//	WithLabels(double(value) * 10.0, {'multiplied': 'true'}) // returns computed value with label
func KSM() cel.EnvOption {
	return cel.Lib(ksmLib)
}

var ksmLib = &ksm{}

type ksm struct{}

func (*ksm) LibraryName() string {
	return "kubestatemetrics"
}

func (*ksm) Types() []*cel.Type {
	return []*cel.Type{ksmcel.WithLabelsObjectType}
}

var ksmLibraryDecls = map[string][]cel.FunctionOpt{
	"WithLabels": {
		cel.Overload("withlabels_any_map",
			[]*cel.Type{cel.DynType, cel.MapType(cel.StringType, cel.StringType)},
			ksmcel.WithLabelsObjectType,
			cel.BinaryBinding(withLabelsConstructor)),
	},
}

func (*ksm) CompileOptions() []cel.EnvOption {
	options := []cel.EnvOption{cel.Types(ksmcel.WithLabelsObjectType)}
	for name, overloads := range ksmLibraryDecls {
		options = append(options, cel.Function(name, overloads...))
	}
	options = append(options, cel.Container("kubestatemetrics"))
	return options
}

func (*ksm) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{}
}

// withLabelsConstructor is the implementation of the WithLabels constructor function.
// It takes a value and a map of labels and returns a WithLabels result.
func withLabelsConstructor(val, labels ref.Val) ref.Val {
	result := &ksmcel.WithLabels{
		Val:              val.Value(),
		AdditionalLabels: make(map[string]string),
	}

	// Extract labels from the map
	if labelsMap, ok := labels.(traits.Mapper); ok {
		it := labelsMap.Iterator()
		for it.HasNext() == types.True {
			key := it.Next()
			value := labelsMap.Get(key)

			keyStr := fmt.Sprintf("%v", key.Value())
			valueStr := fmt.Sprintf("%v", value.Value())
			result.AdditionalLabels[keyStr] = valueStr
		}
	}

	return result
}
