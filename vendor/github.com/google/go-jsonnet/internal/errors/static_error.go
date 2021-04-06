/*
Copyright 2016 Google Inc. All rights reserved.

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

package errors

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
)

//////////////////////////////////////////////////////////////////////////////
// StaticError

// StaticError represents an error during parsing/lexing or static analysis.
// TODO(sbarzowski) Make it possible to have multiple static errors and warnings
type StaticError interface {
	// WithContext returns a new StaticError with additional context before the error message.
	WithContext(string) StaticError
	// Error returns the string representation of a StaticError.
	Error() string
	// Loc returns the place in the source code that triggerred the error.
	Loc() ast.LocationRange
}

type staticError struct {
	loc ast.LocationRange
	msg string
}

func (err staticError) WithContext(context string) StaticError {
	return staticError{
		loc: err.loc,
		msg: fmt.Sprintf("%v while %s", err.msg, context),
	}
}

func (err staticError) Error() string {
	loc := ""
	if err.loc.IsSet() {
		loc = err.loc.String()
	}
	return fmt.Sprintf("%v %v", loc, err.msg)
}

func (err staticError) Loc() ast.LocationRange {
	return err.loc
}

// MakeStaticErrorMsg returns a staticError with a message.
func MakeStaticErrorMsg(msg string) StaticError {
	return staticError{msg: msg}
}

// MakeStaticError returns a StaticError with a message and a LocationRange.
func MakeStaticError(msg string, lr ast.LocationRange) StaticError {
	return staticError{msg: msg, loc: lr}
}
