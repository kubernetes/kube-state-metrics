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

package cel

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// CELResult represents the result of a CEL expression evaluation with additional labels.
type CELResult struct {
	Val              interface{}
	AdditionalLabels map[string]string
}

var (
	CELResultObjectType = cel.ObjectType("kubestatemetrics.CELResult")
)

// ConvertToNative implements ref.Val.ConvertToNative.
func (r *CELResult) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	if reflect.TypeOf(r).AssignableTo(typeDesc) {
		return r, nil
	}
	return nil, fmt.Errorf("type conversion error from 'CELResult' to '%v'", typeDesc)
}

// ConvertToType implements ref.Val.ConvertToType.
func (r *CELResult) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case CELResultObjectType:
		return r
	case types.TypeType:
		return CELResultObjectType
	}
	return types.NewErr("type conversion error from '%s' to '%s'", CELResultObjectType, typeVal)
}

// Equal implements ref.Val.Equal.
func (r *CELResult) Equal(other ref.Val) ref.Val {
	otherResult, ok := other.(*CELResult)
	if !ok {
		return types.False
	}

	// Simple equality check for values
	if !reflect.DeepEqual(r.Val, otherResult.Val) {
		return types.False
	}

	if len(r.AdditionalLabels) != len(otherResult.AdditionalLabels) {
		return types.False
	}

	for k, v := range r.AdditionalLabels {
		if otherResult.AdditionalLabels[k] != v {
			return types.False
		}
	}

	return types.True
}

// Type implements ref.Val.Type.
func (r *CELResult) Type() ref.Type {
	return CELResultObjectType
}

// Value implements ref.Val.Value.
func (r *CELResult) Value() interface{} {
	return r
}
