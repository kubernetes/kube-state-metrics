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

import "github.com/google/go-jsonnet/ast"

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

func (rv *readyValue) evaluate(i *interpreter, trace traceElement, sb selfBinding, origBinding bindingFrame, fieldName string) (value, error) {
	return rv.content, nil
}

func (rv *readyValue) aPotentialValue() {}

// potentialValues
// -------------------------------------

// evaluable is something that can be evaluated and the result is always the same
// It may require computation every time evaluation is requested (in contrast with
// potentialValue which guarantees that computation happens at most once).
type evaluable interface {
	// fromWhere keeps the information from where the evaluation was requested.
	getValue(i *interpreter, fromWhere traceElement) (value, error)
}

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

func (t *cachedThunk) getValue(i *interpreter, trace traceElement) (value, error) {
	if t.content != nil {
		return t.content, nil
	}
	if t.err != nil {
		return nil, t.err
	}
	v, err := i.EvalInCleanEnv(trace, t.env, t.body, false)
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

func (f *codeUnboundField) evaluate(i *interpreter, trace traceElement, sb selfBinding, origBindings bindingFrame, fieldName string) (value, error) {
	env := makeEnvironment(origBindings, sb)
	return i.EvalInCleanEnv(trace, &env, f.body, false)
}

// Provide additional bindings for a field. It shadows bindings from the object.
type bindingsUnboundField struct {
	inner unboundField
	// in addition to "generic" binding frame from the object
	bindings bindingFrame
}

func (f *bindingsUnboundField) evaluate(i *interpreter, trace traceElement, sb selfBinding, origBindings bindingFrame, fieldName string) (value, error) {
	var upValues bindingFrame
	upValues = make(bindingFrame)
	for variable, pvalue := range origBindings {
		upValues[variable] = pvalue
	}
	for variable, pvalue := range f.bindings {
		upValues[variable] = pvalue
	}
	return f.inner.evaluate(i, trace, sb, upValues, fieldName)
}

// plusSuperUnboundField represents a `field+: ...` that hasn't been bound to an object.
type plusSuperUnboundField struct {
	inner unboundField
}

func (f *plusSuperUnboundField) evaluate(i *interpreter, trace traceElement, sb selfBinding, origBinding bindingFrame, fieldName string) (value, error) {
	right, err := f.inner.evaluate(i, trace, sb, origBinding, fieldName)
	if err != nil {
		return nil, err
	}
	if !objectHasField(sb.super(), fieldName, withHidden) {
		return right, nil
	}
	left, err := objectIndex(i, trace, sb.super(), fieldName)
	if err != nil {
		return nil, err
	}
	return builtinPlus(i, trace, left, right)
}

// evalCallables
// -------------------------------------

type closure struct {
	// base environment of a closure
	// arguments should be added to it, before executing it
	env      environment
	function *ast.Function
	params   parameters
}

func forceThunks(i *interpreter, trace traceElement, args *bindingFrame) error {
	for _, arg := range *args {
		_, err := arg.getValue(i, trace)
		if err != nil {
			return err
		}
	}
	return nil
}

func (closure *closure) evalCall(arguments callArguments, i *interpreter, trace traceElement) (value, error) {
	argThunks := make(bindingFrame)
	parameters := closure.Parameters()
	for i, arg := range arguments.positional {
		var name ast.Identifier
		if i < len(parameters.required) {
			name = parameters.required[i]
		} else {
			name = parameters.optional[i-len(parameters.required)].name
		}
		argThunks[name] = arg
	}

	for _, arg := range arguments.named {
		argThunks[arg.name] = arg.pv
	}

	var calledEnvironment environment

	for i := range parameters.optional {
		param := &parameters.optional[i]
		if _, exists := argThunks[param.name]; !exists {
			argThunks[param.name] = &cachedThunk{
				// Default arguments are evaluated in the same environment as function body
				env:  &calledEnvironment,
				body: param.defaultArg,
			}
		}
	}

	if arguments.tailstrict {
		err := forceThunks(i, trace, &argThunks)
		if err != nil {
			return nil, err
		}
	}

	calledEnvironment = makeEnvironment(
		addBindings(closure.env.upValues, argThunks),
		closure.env.selfBinding,
	)
	return i.EvalInCleanEnv(trace, &calledEnvironment, closure.function.Body, arguments.tailstrict)
}

func (closure *closure) Parameters() parameters {
	return closure.params

}

func prepareClosureParameters(params ast.Parameters, env environment) parameters {
	optionalParameters := make([]namedParameter, 0, len(params.Optional))
	for _, named := range params.Optional {
		optionalParameters = append(optionalParameters, namedParameter{
			name:       named.Name,
			defaultArg: named.DefaultArg,
		})
	}
	return parameters{
		required: params.Required,
		optional: optionalParameters,
	}
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
func (native *NativeFunction) evalCall(arguments callArguments, i *interpreter, trace traceElement) (value, error) {
	flatArgs := flattenArgs(arguments, native.Parameters(), []value{})
	nativeArgs := make([]interface{}, 0, len(flatArgs))
	for _, arg := range flatArgs {
		v, err := i.evaluatePV(arg, trace)
		if err != nil {
			return nil, err
		}
		json, err := i.manifestJSON(trace, v)
		if err != nil {
			return nil, err
		}
		nativeArgs = append(nativeArgs, json)
	}
	resultJSON, err := native.Func(nativeArgs)
	if err != nil {
		return nil, i.Error(err.Error(), trace)
	}
	return jsonToValue(i, trace, resultJSON)
}

// parameters returns a NativeFunction's parameters.
func (native *NativeFunction) Parameters() parameters {
	return parameters{required: native.Params}
}

// -------------------------------------

type defaultArgument struct {
	body ast.Node
}

func (da *defaultArgument) inEnv(env *environment) potentialValue {
	return &cachedThunk{env: env, body: da.body}
}
