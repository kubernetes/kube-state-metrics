/*
Copyright 2017 Google Inc. All rights reserved.

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

package jsonnet

import (
	"github.com/google/go-jsonnet/ast"
)

// readyValue
// -------------------------------------

// readyValue is a wrapper which allows to use a concrete value where normally
// some evaluation would be expected (e.g. object fields). It's not part
// of the value interface for increased type safety (it would be very easy
// to "overevaluate" otherwise) and conveniently it also saves us from implementing
// these methods for all value types.
type readyValue struct {
	content value
}

func (rv *readyValue) evaluate(i *interpreter, sb selfBinding, origBinding bindingFrame, fieldName string) (value, error) {
	return rv.content, nil
}

// potentialValues
// -------------------------------------

// cachedThunk is a wrapper that caches the value of a potentialValue after
// the first evaluation.
// Note: All potentialValues are required to provide the same value every time,
// so it's only there for efficiency.
type cachedThunk struct {
	// The environment is a pointer because it may be a cyclic structure.  A thunk
	// may refer to itself, so inside `env` there will be a variable bound back to us.
	env  *environment
	body ast.Node
	// If nil, use err.
	content value
	// If also nil, content is not cached yet.
	err error
}

func readyThunk(content value) *cachedThunk {
	return &cachedThunk{content: content}
}

func (t *cachedThunk) getValue(i *interpreter) (value, error) {
	if t.content != nil {
		return t.content, nil
	}
	if t.err != nil {
		return nil, t.err
	}
	v, err := i.EvalInCleanEnv(t.env, t.body, false)
	if err != nil {
		// TODO(sbarzowski) perhaps cache errors as well
		// may be necessary if we allow handling them in any way
		return nil, err
	}
	t.content = v
	// No need to keep the environment around anymore.
	// So, this might reduce memory pressure:
	t.env = nil
	return v, nil
}

func (t *cachedThunk) aPotentialValue() {}

// unboundFields
// -------------------------------------

type codeUnboundField struct {
	body ast.Node
}

func (f *codeUnboundField) evaluate(i *interpreter, sb selfBinding, origBindings bindingFrame, fieldName string) (value, error) {
	env := makeEnvironment(origBindings, sb)
	return i.EvalInCleanEnv(&env, f.body, false)
}

// Provide additional bindings for a field. It shadows bindings from the object.
type bindingsUnboundField struct {
	inner unboundField
	// in addition to "generic" binding frame from the object
	bindings bindingFrame
}

func (f *bindingsUnboundField) evaluate(i *interpreter, sb selfBinding, origBindings bindingFrame, fieldName string) (value, error) {
	upValues := make(bindingFrame)
	for variable, pvalue := range origBindings {
		upValues[variable] = pvalue
	}
	for variable, pvalue := range f.bindings {
		upValues[variable] = pvalue
	}
	return f.inner.evaluate(i, sb, upValues, fieldName)
}

// plusSuperUnboundField represents a `field+: ...` that hasn't been bound to an object.
type plusSuperUnboundField struct {
	inner unboundField
}

func (f *plusSuperUnboundField) evaluate(i *interpreter, sb selfBinding, origBinding bindingFrame, fieldName string) (value, error) {
	right, err := f.inner.evaluate(i, sb, origBinding, fieldName)
	if err != nil {
		return nil, err
	}
	if !objectHasField(sb.super(), fieldName, withHidden) {
		return right, nil
	}
	left, err := objectIndex(i, sb.super(), fieldName)
	if err != nil {
		return nil, err
	}
	return builtinPlus(i, left, right)
}

// evalCallables
// -------------------------------------

type closure struct {
	// base environment of a closure
	// arguments should be added to it, before executing it
	env      environment
	function *ast.Function
	params   []namedParameter
}

func forceThunks(i *interpreter, args *bindingFrame) error {
	for _, arg := range *args {
		_, err := arg.getValue(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (closure *closure) evalCall(arguments callArguments, i *interpreter) (value, error) {
	argThunks := make(bindingFrame)
	parameters := closure.parameters()
	for i, arg := range arguments.positional {
		argThunks[parameters[i].name] = arg
	}

	for _, arg := range arguments.named {
		argThunks[arg.name] = arg.pv
	}

	var calledEnvironment environment

	for _, param := range parameters {
		if _, exists := argThunks[param.name]; !exists {
			argThunks[param.name] = &cachedThunk{
				// Default arguments are evaluated in the same environment as function body
				env:  &calledEnvironment,
				body: param.defaultArg,
			}
		}
	}

	if arguments.tailstrict {
		err := forceThunks(i, &argThunks)
		if err != nil {
			return nil, err
		}
	}

	calledEnvironment = makeEnvironment(
		addBindings(closure.env.upValues, argThunks),
		closure.env.selfBinding,
	)
	return i.EvalInCleanEnv(&calledEnvironment, closure.function.Body, arguments.tailstrict)
}

func (closure *closure) parameters() []namedParameter {
	return closure.params

}

func prepareClosureParameters(params []ast.Parameter, env environment) []namedParameter {
	preparedParams := make([]namedParameter, 0, len(params))
	for _, named := range params {
		preparedParams = append(preparedParams, namedParameter{
			name:       named.Name,
			defaultArg: named.DefaultArg,
		})
	}
	return preparedParams
}

func makeClosure(env environment, function *ast.Function) *closure {
	return &closure{
		env:      env,
		function: function,
		params:   prepareClosureParameters(function.Parameters, env),
	}
}

// NativeFunction represents a function implemented in Go.
type NativeFunction struct {
	Func   func([]interface{}) (interface{}, error)
	Params ast.Identifiers
	Name   string
}

// evalCall evaluates a call to a NativeFunction and returns the result.
func (native *NativeFunction) evalCall(arguments callArguments, i *interpreter) (value, error) {
	flatArgs := flattenArgs(arguments, native.parameters(), []value{})
	nativeArgs := make([]interface{}, 0, len(flatArgs))
	for _, arg := range flatArgs {
		v, err := i.evaluatePV(arg)
		if err != nil {
			return nil, err
		}
		json, err := i.manifestJSON(v)
		if err != nil {
			return nil, err
		}
		nativeArgs = append(nativeArgs, json)
	}
	resultJSON, err := native.Func(nativeArgs)
	if err != nil {
		return nil, i.Error(err.Error())
	}
	return jsonToValue(i, resultJSON)
}

// Parameters returns a NativeFunction's parameters.
func (native *NativeFunction) parameters() []namedParameter {
	ret := make([]namedParameter, len(native.Params))
	for i := range ret {
		ret[i].name = native.Params[i]
	}
	return ret
}
