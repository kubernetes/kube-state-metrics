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

var _ ref.Val = &WithLabels{}

// WithLabels represents a metric value with additional labels.
// Implements ref.Val.
type WithLabels struct {
	Val              interface{}
	AdditionalLabels map[string]string
}

var (
	// WithLabelsObjectType is the CEL type representation for WithLabels objects.
	WithLabelsObjectType = cel.ObjectType("kubestatemetrics.WithLabels")
)

// ConvertToNative implements the ref.Val interface method.
func (r *WithLabels) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	if reflect.TypeOf(r).AssignableTo(typeDesc) {
		return r, nil
	}
	return nil, fmt.Errorf("type conversion error from 'WithLabels' to '%v'", typeDesc)
}

// ConvertToType implements the ref.Val interface method.
func (r *WithLabels) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case WithLabelsObjectType:
		return r
	case types.TypeType:
		return WithLabelsObjectType
	}
	return types.NewErr("type conversion error from '%s' to '%s'", WithLabelsObjectType, typeVal)
}

// Equal implements the ref.Val interface method.
func (r *WithLabels) Equal(other ref.Val) ref.Val {
	otherResult, ok := other.(*WithLabels)
	if !ok {
		return types.False
	}

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

// Type implements the ref.Val interface method.
func (r *WithLabels) Type() ref.Type {
	return WithLabelsObjectType
}

// Value implements the ref.Val interface method.
func (r *WithLabels) Value() interface{} {
	return r
}
