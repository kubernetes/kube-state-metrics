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

package jsonnet

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"

	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/astgen"
)

// TODO(sbarzowski) use it as a pointer in most places b/c it can sometimes be shared
// for example it can be shared between array elements and function arguments
type environment struct {
	selfBinding selfBinding

	// Bindings introduced in this frame. The way previous bindings are treated
	// depends on the type of a frame.
	// If isCall == true then previous bindings are ignored (it's a clean
	// environment with just the variables we have here).
	// If isCall == false then if this frame doesn't contain a binding
	// previous bindings will be used.
	upValues bindingFrame
}

func makeEnvironment(upValues bindingFrame, sb selfBinding) environment {
	return environment{
		upValues:    upValues,
		selfBinding: sb,
	}
}

func (i *interpreter) getCurrentStackTrace() []traceFrame {
	var result []traceFrame
	for _, f := range i.stack.stack {
		if f.isCall {
			result = append(result, traceElementToTraceFrame(f.trace))
		}
	}
	if i.stack.currentTrace.loc != nil {
		result = append(result, traceElementToTraceFrame(i.stack.currentTrace))
	}
	return result
}

type callFrame struct {
	// True if it switches to a clean environment (function call or array element)
	// False otherwise, e.g. for local
	// This makes callFrame a misnomer as it is technically not always a call...
	isCall bool

	// Tracing information about the place where it was called from.
	trace traceElement

	// Whether this frame can be removed from the stack when it doesn't affect
	// the evaluation result, but in case of an error, it won't appear on the
	// stack trace.
	// It's used for tail call optimization.
	trimmable bool

	env environment
}

func dumpCallFrame(c *callFrame) string {
	var loc ast.LocationRange
	if c.trace.loc == nil {
		loc = ast.MakeLocationRangeMessage("?")
	} else {
		loc = *c.trace.loc
	}
	return fmt.Sprintf("<callFrame isCall = %t location = %v trimmable = %t>",
		c.isCall,
		loc,
		c.trimmable,
	)
}

type callStack struct {
	calls        int
	limit        int
	stack        []*callFrame
	currentTrace traceElement
}

func dumpCallStack(c *callStack) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "<callStack calls = %d limit = %d stack:\n", c.calls, c.limit)
	for _, callFrame := range c.stack {
		fmt.Fprintf(&buf, "  %v\n", dumpCallFrame(callFrame))
	}
	buf.WriteString("\n>")
	return buf.String()
}

func (s *callStack) top() *callFrame {
	r := s.stack[len(s.stack)-1]
	return r
}

// It might've been popped already by tail call optimization.
// We check if it was trimmed by comparing the current stack size to the position
// of the frame we want to pop.
func (s *callStack) popIfExists(whichFrame int) {
	if len(s.stack) == whichFrame {
		if s.top().isCall {
			s.calls--
		}
		s.setCurrentTrace(s.stack[len(s.stack)-1].trace)
		s.stack = s.stack[:len(s.stack)-1]
	}
}

/** If there is a trimmable frame followed by some locals, pop them all. */
func (s *callStack) tailCallTrimStack() {
	for i := len(s.stack) - 1; i >= 0; i-- {
		if s.stack[i].isCall {
			if !s.stack[i].trimmable {
				return
			}
			// Remove this stack frame and everything above it
			s.stack = s.stack[:i]
			s.calls--
			return
		}
	}
}

func (s *callStack) setCurrentTrace(trace traceElement) {
	if s.currentTrace != (traceElement{}) {
		panic("Tried to change the traceElement while the old one was still there.")
	}
	s.currentTrace = trace
}

func (s *callStack) clearCurrentTrace() {
	s.currentTrace = traceElement{}
}

type tailCallStatus int

const (
	nonTailCall tailCallStatus = iota
	tailCall
)

func (s *callStack) newCall(env environment, trimmable bool) {
	if s.currentTrace == (traceElement{}) {
		panic("Saving empty traceElement on stack")
	}
	s.stack = append(s.stack, &callFrame{
		isCall:    true,
		trace:     s.currentTrace,
		env:       env,
		trimmable: trimmable,
	})
	s.clearCurrentTrace()
	s.calls++
}

func (s *callStack) newLocal(vars bindingFrame) {
	s.stack = append(s.stack, &callFrame{
		env:   makeEnvironment(vars, selfBinding{}),
		trace: s.currentTrace,
	})
	s.clearCurrentTrace()
}

// getSelfBinding resolves the self construct
func (s *callStack) getSelfBinding() selfBinding {
	for i := len(s.stack) - 1; i >= 0; i-- {
		if s.stack[i].isCall {
			return s.stack[i].env.selfBinding
		}
	}
	panic(fmt.Sprintf("malformed stack %v", dumpCallStack(s)))
}

// lookUpVar finds for the closest variable in scope that matches the given name.
func (s *callStack) lookUpVar(id ast.Identifier) *cachedThunk {
	for i := len(s.stack) - 1; i >= 0; i-- {
		bind, present := s.stack[i].env.upValues[id]
		if present {
			return bind
		}
		if s.stack[i].isCall {
			// Nothing beyond the captured environment of the thunk / closure.
			break
		}
	}
	return nil
}

func (s *callStack) lookUpVarOrPanic(id ast.Identifier) *cachedThunk {
	th := s.lookUpVar(id)
	if th == nil {
		panic(fmt.Sprintf("RUNTIME: Unknown variable: %v (we should have caught this statically)", id))
	}
	return th
}

func (s *callStack) getCurrentEnv(ast ast.Node) environment {
	return makeEnvironment(
		s.capture(ast.FreeVariables()),
		s.getSelfBinding(),
	)
}

// Build a binding frame containing specified variables.
func (s *callStack) capture(freeVars ast.Identifiers) bindingFrame {
	env := make(bindingFrame)
	for _, fv := range freeVars {
		env[fv] = s.lookUpVarOrPanic(fv)
	}
	return env
}

func makeCallStack(limit int) callStack {
	return callStack{
		calls: 0,
		limit: limit,
	}
}

// Keeps current execution context and evaluates things
type interpreter struct {
	// Current stack. It is used for:
	// 1) Keeping environment (object we're in, variables)
	// 2) Diagnostic information in case of failure
	stack callStack

	// External variables
	extVars map[string]*cachedThunk

	// Native functions
	nativeFuncs map[string]*NativeFunction

	// A part of std object common to all files
	baseStd *valueObject

	// Keeps imports
	importCache *importCache
}

// Map union, b takes precedence when keys collide.
func addBindings(a, b bindingFrame) bindingFrame {
	result := make(bindingFrame)

	for k, v := range a {
		result[k] = v
	}

	for k, v := range b {
		result[k] = v
	}

	return result
}

func (i *interpreter) newCall(env environment, trimmable bool) error {
	s := &i.stack
	if s.calls >= s.limit {
		return makeRuntimeError("max stack frames exceeded.", i.getCurrentStackTrace())
	}
	s.newCall(env, trimmable)
	return nil
}

func (i *interpreter) evaluate(a ast.Node, tc tailCallStatus) (value, error) {
	trace := traceElement{
		loc:     a.Loc(),
		context: a.Context(),
	}
	oldTrace := i.stack.currentTrace
	i.stack.clearCurrentTrace()
	i.stack.setCurrentTrace(trace)
	defer func() { i.stack.clearCurrentTrace(); i.stack.setCurrentTrace(oldTrace) }()

	switch node := a.(type) {
	case *ast.Array:
		sb := i.stack.getSelfBinding()
		var elements []*cachedThunk
		for _, el := range node.Elements {
			env := makeEnvironment(i.stack.capture(el.Expr.FreeVariables()), sb)
			elThunk := cachedThunk{env: &env, body: el.Expr}
			elements = append(elements, &elThunk)
		}
		return makeValueArray(elements), nil

	case *ast.Binary:
		if node.Op == ast.BopAnd {
			// Special case for shortcut semantics.
			xv, err := i.evaluate(node.Left, nonTailCall)
			if err != nil {
				return nil, err
			}
			x, err := i.getBoolean(xv)
			if err != nil {
				return nil, err
			}
			if !x.value {
				return x, nil
			}
			yv, err := i.evaluate(node.Right, tc)
			if err != nil {
				return nil, err
			}
			return i.getBoolean(yv)
		} else if node.Op == ast.BopOr {
			// Special case for shortcut semantics.
			xv, err := i.evaluate(node.Left, nonTailCall)
			if err != nil {
				return nil, err
			}
			x, err := i.getBoolean(xv)
			if err != nil {
				return nil, err
			}
			if x.value {
				return x, nil
			}
			yv, err := i.evaluate(node.Right, tc)
			if err != nil {
				return nil, err
			}
			return i.getBoolean(yv)

		} else {
			left, err := i.evaluate(node.Left, nonTailCall)
			if err != nil {
				return nil, err
			}
			right, err := i.evaluate(node.Right, tc)
			if err != nil {
				return nil, err
			}
			// TODO(dcunnin): The double dereference here is probably not necessary.
			builtin := bopBuiltins[node.Op]
			return builtin.function(i, left, right)
		}

	case *ast.Unary:
		value, err := i.evaluate(node.Expr, tc)
		if err != nil {
			return nil, err
		}

		builtin := uopBuiltins[node.Op]

		result, err := builtin.function(i, value)
		if err != nil {
			return nil, err
		}
		return result, nil

	case *ast.Conditional:
		cond, err := i.evaluate(node.Cond, nonTailCall)
		if err != nil {
			return nil, err
		}
		condBool, err := i.getBoolean(cond)
		if err != nil {
			return nil, err
		}
		if condBool.value {
			return i.evaluate(node.BranchTrue, tc)
		}
		return i.evaluate(node.BranchFalse, tc)

	case *ast.DesugaredObject:
		// Evaluate all the field names.  Check for null, dups, etc.
		fields := make(simpleObjectFieldMap)
		for _, field := range node.Fields {
			fieldNameValue, err := i.evaluate(field.Name, nonTailCall)
			if err != nil {
				return nil, err
			}
			var fieldName string
			switch fieldNameValue := fieldNameValue.(type) {
			case valueString:
				fieldName = fieldNameValue.getGoString()
			case *valueNull:
				// Omitted field.
				continue
			default:
				return nil, i.Error(fmt.Sprintf("Field name must be string, got %v", fieldNameValue.getType().name))
			}

			if _, ok := fields[fieldName]; ok {
				return nil, i.Error(duplicateFieldNameErrMsg(fieldName))
			}
			var f unboundField = &codeUnboundField{field.Body}
			if field.PlusSuper {
				f = &plusSuperUnboundField{f}
			}
			fields[fieldName] = simpleObjectField{field.Hide, f}
		}
		var asserts []unboundField
		for _, assert := range node.Asserts {
			asserts = append(asserts, &codeUnboundField{assert})
		}
		var locals []objectLocal
		for _, local := range node.Locals {
			locals = append(locals, objectLocal{name: local.Variable, node: local.Body})
		}
		upValues := i.stack.capture(node.FreeVariables())
		return makeValueSimpleObject(upValues, fields, asserts, locals), nil

	case *ast.Error:
		msgVal, err := i.evaluate(node.Expr, nonTailCall)
		if err != nil {
			// error when evaluating error message
			return nil, err
		}
		if msgVal.getType() != stringType {
			msgVal, err = builtinToString(i, msgVal)
			if err != nil {
				return nil, err
			}
		}
		msg, err := i.getString(msgVal)
		if err != nil {
			return nil, err
		}
		return nil, i.Error(msg.getGoString())

	case *ast.Index:
		targetValue, err := i.evaluate(node.Target, nonTailCall)
		if err != nil {
			return nil, err
		}
		index, err := i.evaluate(node.Index, nonTailCall)
		if err != nil {
			return nil, err
		}
		switch target := targetValue.(type) {
		case *valueObject:
			indexString, err := i.getString(index)
			if err != nil {
				return nil, err
			}
			return target.index(i, indexString.getGoString())
		case *valueArray:
			indexInt, err := i.getNumber(index)
			if err != nil {
				return nil, err
			}
			// TODO(https://github.com/google/jsonnet/issues/377): non-integer indexes should be an error
			return target.index(i, int(indexInt.value))

		case valueString:
			indexInt, err := i.getNumber(index)
			if err != nil {
				return nil, err
			}
			// TODO(https://github.com/google/jsonnet/issues/377): non-integer indexes should be an error
			return target.index(i, int(indexInt.value))
		}

		return nil, i.Error(fmt.Sprintf("Value non indexable: %v", reflect.TypeOf(targetValue)))

	case *ast.Import:
		codePath := node.Loc().FileName
		return i.importCache.importCode(codePath, node.File.Value, i)

	case *ast.ImportStr:
		codePath := node.Loc().FileName
		return i.importCache.importString(codePath, node.File.Value, i)

	case *ast.LiteralBoolean:
		return makeValueBoolean(node.Value), nil

	case *ast.LiteralNull:
		return makeValueNull(), nil

	case *ast.LiteralNumber:
		// Since the lexer ensures that OriginalString is of
		// the right form, this will only fail if the number is
		// too large to fit in a double.
		num, err := strconv.ParseFloat(node.OriginalString, 64)
		if err != nil {
			return nil, i.Error("overflow")
		}
		return makeValueNumber(num), nil

	case *ast.LiteralString:
		return makeValueString(node.Value), nil

	case *ast.Local:
		vars := make(bindingFrame)
		bindEnv := i.stack.getCurrentEnv(a)
		for _, bind := range node.Binds {
			th := cachedThunk{env: &bindEnv, body: bind.Body}

			// recursive locals
			vars[bind.Variable] = &th
			bindEnv.upValues[bind.Variable] = &th
		}
		i.stack.newLocal(vars)
		sz := len(i.stack.stack)
		// Add new stack frame, with new thunk for this variable
		// execute body WRT stack frame.
		v, err := i.evaluate(node.Body, tc)
		i.stack.popIfExists(sz)

		return v, err

	case *ast.Self:
		sb := i.stack.getSelfBinding()
		return sb.self, nil

	case *ast.Var:
		foo := i.stack.lookUpVarOrPanic(node.Id)
		return foo.getValue(i)

	case *ast.SuperIndex:
		index, err := i.evaluate(node.Index, nonTailCall)
		if err != nil {
			return nil, err
		}
		indexStr, err := i.getString(index)
		if err != nil {
			return nil, err
		}
		return objectIndex(i, i.stack.getSelfBinding().super(), indexStr.getGoString())

	case *ast.InSuper:
		index, err := i.evaluate(node.Index, nonTailCall)
		if err != nil {
			return nil, err
		}
		indexStr, err := i.getString(index)
		if err != nil {
			return nil, err
		}
		hasField := objectHasField(i.stack.getSelfBinding().super(), indexStr.getGoString(), withHidden)
		return makeValueBoolean(hasField), nil

	case *ast.Function:
		return &valueFunction{
			ec: makeClosure(i.stack.getCurrentEnv(a), node),
		}, nil

	case *ast.Apply:
		// Eval target
		target, err := i.evaluate(node.Target, nonTailCall)
		if err != nil {
			return nil, err
		}
		function, err := i.getFunction(target)
		if err != nil {
			return nil, err
		}

		// environment in which we can evaluate arguments
		argEnv := i.stack.getCurrentEnv(a)
		arguments := callArguments{
			positional: make([]*cachedThunk, len(node.Arguments.Positional)),
			named:      make([]namedCallArgument, len(node.Arguments.Named)),
			tailstrict: node.TailStrict,
		}
		for i, arg := range node.Arguments.Positional {
			arguments.positional[i] = &cachedThunk{env: &argEnv, body: arg.Expr}
		}

		for i, arg := range node.Arguments.Named {
			arguments.named[i] = namedCallArgument{name: arg.Name, pv: &cachedThunk{env: &argEnv, body: arg.Arg}}
		}
		return i.evaluateTailCall(function, arguments, tc)

	case *astMakeArrayElement:
		arguments := callArguments{
			positional: []*cachedThunk{
				&cachedThunk{
					content: intToValue(node.index),
				},
			},
		}
		return i.evaluateTailCall(node.function, arguments, tc)

	default:
		panic(fmt.Sprintf("Executing this AST type not implemented: %v", reflect.TypeOf(a)))
	}
}

// unparseString Wraps in "" and escapes stuff to make the string JSON-compliant and human-readable.
func unparseString(v string) string {
	var buf bytes.Buffer
	buf.WriteString("\"")
	for _, c := range v {
		switch c {
		case '"':
			buf.WriteString("\\\"")
		case '\\':
			buf.WriteString("\\\\")
		case '\b':
			buf.WriteString("\\b")
		case '\f':
			buf.WriteString("\\f")
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		case 0:
			buf.WriteString("\\u0000")
		default:
			if c < 0x20 || (c >= 0x7f && c <= 0x9f) {
				buf.WriteString(fmt.Sprintf("\\u%04x", int(c)))
			} else {
				buf.WriteRune(c)
			}
		}
	}
	buf.WriteString("\"")
	return buf.String()
}

func unparseNumber(v float64) string {
	if v == math.Floor(v) {
		return fmt.Sprintf("%.0f", v)
	}

	// See "What Every Computer Scientist Should Know About Floating-Point Arithmetic"
	// Theorem 15
	// http://docs.oracle.com/cd/E19957-01/806-3568/ncg_goldberg.html
	return fmt.Sprintf("%.17g", v)
}

// manifestJSON converts to standard JSON representation as in "encoding/json" package
func (i *interpreter) manifestJSON(v value) (interface{}, error) {
	// TODO(sbarzowski) Add nice stack traces indicating the part of the code which
	// evaluates to non-manifestable value (that might require passing context about
	// the root value)
	if i.stack.currentTrace == (traceElement{}) {
		panic("manifesting JSON with empty traceElement")
	}
	switch v := v.(type) {

	case *valueBoolean:
		return v.value, nil

	case *valueFunction:
		return nil, makeRuntimeError("couldn't manifest function as JSON", i.getCurrentStackTrace())

	case *valueNumber:
		return v.value, nil

	case valueString:
		return v.getGoString(), nil

	case *valueNull:
		return nil, nil

	case *valueArray:
		result := make([]interface{}, 0, len(v.elements))
		for _, th := range v.elements {
			elVal, err := i.evaluatePV(th)
			if err != nil {
				return nil, err
			}
			elem, err := i.manifestJSON(elVal)
			if err != nil {
				return nil, err
			}
			result = append(result, elem)
		}
		return result, nil

	case *valueObject:
		fieldNames := objectFields(v, withoutHidden)
		sort.Strings(fieldNames)

		err := checkAssertions(i, v)
		if err != nil {
			return nil, err
		}

		result := make(map[string]interface{})

		for _, fieldName := range fieldNames {
			fieldVal, err := v.index(i, fieldName)
			if err != nil {
				return nil, err
			}

			field, err := i.manifestJSON(fieldVal)
			if err != nil {
				return nil, err
			}
			result[fieldName] = field
		}

		return result, nil

	default:
		return nil, makeRuntimeError(
			fmt.Sprintf("manifesting this value not implemented yet: %s", reflect.TypeOf(v)),
			i.getCurrentStackTrace(),
		)

	}
}

func serializeJSON(v interface{}, multiline bool, indent string, buf *bytes.Buffer) {
	switch v := v.(type) {
	case nil:
		buf.WriteString("null")

	case []interface{}:
		if len(v) == 0 {
			buf.WriteString("[ ]")
		} else {
			var prefix string
			var indent2 string
			if multiline {
				prefix = "[\n"
				indent2 = indent + "   "
			} else {
				prefix = "["
				indent2 = indent
			}
			for _, elem := range v {
				buf.WriteString(prefix)
				buf.WriteString(indent2)
				serializeJSON(elem, multiline, indent2, buf)
				if multiline {
					prefix = ",\n"
				} else {
					prefix = ", "
				}
			}
			if multiline {
				buf.WriteString("\n")
			}
			buf.WriteString(indent)
			buf.WriteString("]")
		}

	case bool:
		if v {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}

	case float64:
		buf.WriteString(unparseNumber(v))

	case map[string]interface{}:
		fieldNames := make([]string, 0, len(v))
		for name := range v {
			fieldNames = append(fieldNames, name)
		}
		sort.Strings(fieldNames)

		if len(fieldNames) == 0 {
			buf.WriteString("{ }")
		} else {
			var prefix string
			var indent2 string
			if multiline {
				prefix = "{\n"
				indent2 = indent + "   "
			} else {
				prefix = "{"
				indent2 = indent
			}
			for _, fieldName := range fieldNames {
				fieldVal := v[fieldName]

				buf.WriteString(prefix)
				buf.WriteString(indent2)

				buf.WriteString(unparseString(fieldName))
				buf.WriteString(": ")

				serializeJSON(fieldVal, multiline, indent2, buf)

				if multiline {
					prefix = ",\n"
				} else {
					prefix = ", "
				}
			}

			if multiline {
				buf.WriteString("\n")
			}
			buf.WriteString(indent)
			buf.WriteString("}")
		}

	case string:
		buf.WriteString(unparseString(v))

	default:
		panic(fmt.Sprintf("Unsupported value for serialization %#+v", v))
	}
}

func (i *interpreter) manifestAndSerializeJSON(
	buf *bytes.Buffer, v value, multiline bool, indent string) error {
	manifested, err := i.manifestJSON(v)
	if err != nil {
		return err
	}
	serializeJSON(manifested, multiline, indent, buf)
	return nil
}

// manifestString expects the value to be a string and returns it.
func (i *interpreter) manifestString(buf *bytes.Buffer, v value) error {
	switch v := v.(type) {
	case valueString:
		buf.WriteString(v.getGoString())
		return nil
	default:
		return makeRuntimeError(fmt.Sprintf("expected string result, got: %s", v.getType().name), i.getCurrentStackTrace())
	}
}

func (i *interpreter) manifestAndSerializeMulti(v value, stringOutputMode bool) (r map[string]string, err error) {
	r = make(map[string]string)
	json, err := i.manifestJSON(v)
	if err != nil {
		return r, err
	}
	switch json := json.(type) {
	case map[string]interface{}:
		for filename, fileJSON := range json {
			if stringOutputMode {
				switch val := fileJSON.(type) {
				case string:
					r[filename] = val
				default:
					msg := fmt.Sprintf("multi mode: top-level object's key %s has a value of type %T, "+
						"should be a string", filename, val)
					return r, makeRuntimeError(msg, i.getCurrentStackTrace())
				}
			} else {
				var buf bytes.Buffer
				serializeJSON(fileJSON, true, "", &buf)
				buf.WriteString("\n")
				r[filename] = buf.String()
			}
		}
	default:
		msg := fmt.Sprintf("multi mode: top-level object was a %s, "+
			"should be an object whose keys are filenames and values hold "+
			"the JSON for that file.", v.getType().name)
		return r, makeRuntimeError(msg, i.getCurrentStackTrace())
	}
	return
}

func (i *interpreter) manifestAndSerializeYAMLStream(v value) (r []string, err error) {
	r = make([]string, 0)
	json, err := i.manifestJSON(v)
	if err != nil {
		return r, err
	}
	switch json := json.(type) {
	case []interface{}:
		for _, doc := range json {
			var buf bytes.Buffer
			serializeJSON(doc, true, "", &buf)
			buf.WriteString("\n")
			r = append(r, buf.String())
		}
	default:
		msg := fmt.Sprintf("stream mode: top-level object was a %s, "+
			"should be an array whose elements hold "+
			"the JSON for each document in the stream.", v.getType().name)
		return r, makeRuntimeError(msg, i.getCurrentStackTrace())
	}
	return
}

func jsonToValue(i *interpreter, v interface{}) (value, error) {
	switch v := v.(type) {
	case nil:
		return &nullValue, nil

	case []interface{}:
		elems := make([]*cachedThunk, len(v))
		for counter, elem := range v {
			val, err := jsonToValue(i, elem)
			if err != nil {
				return nil, err
			}
			elems[counter] = readyThunk(val)
		}
		return makeValueArray(elems), nil

	case bool:
		return makeValueBoolean(v), nil
	case float64:
		return makeDoubleCheck(i, v)

	case map[string]interface{}:
		fieldMap := map[string]value{}
		for name, f := range v {
			val, err := jsonToValue(i, f)
			if err != nil {
				return nil, err
			}
			fieldMap[name] = val
		}
		return buildObject(ast.ObjectFieldInherit, fieldMap), nil

	case string:
		return makeValueString(v), nil

	default:
		return nil, i.Error(fmt.Sprintf("Not a json type: %#+v", v))
	}
}

func (i *interpreter) EvalInCleanEnv(env *environment, ast ast.Node, trimmable bool) (value, error) {
	err := i.newCall(*env, trimmable)
	if err != nil {
		return nil, err
	}
	stackSize := len(i.stack.stack)

	val, err := i.evaluate(ast, tailCall)

	i.stack.popIfExists(stackSize)

	return val, err
}

func (i *interpreter) evaluatePV(ph potentialValue) (value, error) {
	return ph.getValue(i)
}

func (i *interpreter) evaluateTailCall(function *valueFunction, args callArguments, tc tailCallStatus) (value, error) {
	if tc == tailCall {
		i.stack.tailCallTrimStack()
	}
	return function.call(i, args)
}

func (i *interpreter) Error(s string) error {
	err := makeRuntimeError(s, i.getCurrentStackTrace())
	return err
}

func (i *interpreter) typeErrorSpecific(bad value, good value) error {
	return i.Error(
		fmt.Sprintf("Unexpected type %v, expected %v", bad.getType().name, good.getType().name),
	)
}

func (i *interpreter) typeErrorGeneral(bad value) error {
	return i.Error(
		fmt.Sprintf("Unexpected type %v", bad.getType().name),
	)
}

func (i *interpreter) getNumber(val value) (*valueNumber, error) {
	switch v := val.(type) {
	case *valueNumber:
		return v, nil
	default:
		return nil, i.typeErrorSpecific(val, &valueNumber{})
	}
}

//nolint:unused
func (i *interpreter) evaluateNumber(pv potentialValue) (*valueNumber, error) {
	v, err := i.evaluatePV(pv)
	if err != nil {
		return nil, err
	}
	return i.getNumber(v)
}

func (i *interpreter) getInt(val value) (int, error) {
	num, err := i.getNumber(val)
	if err != nil {
		return 0, err
	}
	// We conservatively convert ot int32, so that it can be machine-sized int
	// on any machine. And it's used only for indexing anyway.
	intNum := int(int32(num.value))
	if float64(intNum) != num.value {
		return 0, i.Error(fmt.Sprintf("Expected an integer, but got %v", num.value))
	}
	return intNum, nil
}

func (i *interpreter) evaluateInt(pv potentialValue) (int, error) {
	v, err := i.evaluatePV(pv)
	if err != nil {
		return 0, err
	}
	return i.getInt(v)
}

//nolint:unused
func (i *interpreter) getInt64(val value) (int64, error) {
	num, err := i.getNumber(val)
	if err != nil {
		return 0, err
	}
	intNum := int64(num.value)
	if float64(intNum) != num.value {
		return 0, i.Error(fmt.Sprintf("Expected an integer, but got %v", num.value))
	}
	return intNum, nil
}

//nolint:unused
func (i *interpreter) evaluateInt64(pv potentialValue) (int64, error) {
	v, err := i.evaluatePV(pv)
	if err != nil {
		return 0, err
	}
	return i.getInt64(v)
}

func (i *interpreter) getString(val value) (valueString, error) {
	switch v := val.(type) {
	case valueString:
		return v, nil
	default:
		return nil, i.typeErrorSpecific(val, emptyString())
	}
}

//nolint:unused
func (i *interpreter) evaluateString(pv potentialValue) (valueString, error) {
	v, err := i.evaluatePV(pv)
	if err != nil {
		return nil, err
	}
	return i.getString(v)
}

func (i *interpreter) getBoolean(val value) (*valueBoolean, error) {
	switch v := val.(type) {
	case *valueBoolean:
		return v, nil
	default:
		return nil, i.typeErrorSpecific(val, &valueBoolean{})
	}
}

//nolint:unused
func (i *interpreter) evaluateBoolean(pv potentialValue) (*valueBoolean, error) {
	v, err := i.evaluatePV(pv)
	if err != nil {
		return nil, err
	}
	return i.getBoolean(v)
}

func (i *interpreter) getArray(val value) (*valueArray, error) {
	switch v := val.(type) {
	case *valueArray:
		return v, nil
	default:
		return nil, i.typeErrorSpecific(val, &valueArray{})
	}
}

//nolint:unused
func (i *interpreter) evaluateArray(pv potentialValue) (*valueArray, error) {
	v, err := i.evaluatePV(pv)
	if err != nil {
		return nil, err
	}
	return i.getArray(v)
}

func (i *interpreter) getFunction(val value) (*valueFunction, error) {
	switch v := val.(type) {
	case *valueFunction:
		return v, nil
	default:
		return nil, i.typeErrorSpecific(val, &valueFunction{})
	}
}

//nolint:unused
func (i *interpreter) evaluateFunction(pv potentialValue) (*valueFunction, error) {
	v, err := i.evaluatePV(pv)
	if err != nil {
		return nil, err
	}
	return i.getFunction(v)
}

func (i *interpreter) getObject(val value) (*valueObject, error) {
	switch v := val.(type) {
	case *valueObject:
		return v, nil
	default:
		return nil, i.typeErrorSpecific(val, &valueObject{})
	}
}

func (i *interpreter) evaluateObject(pv potentialValue) (*valueObject, error) {
	v, err := i.evaluatePV(pv)
	if err != nil {
		return nil, err
	}
	return i.getObject(v)
}

func buildStdObject(i *interpreter) (*valueObject, error) {
	objVal, err := evaluateStd(i)
	if err != nil {
		return nil, err
	}
	obj := objVal.(*valueObject).uncached.(*simpleObject)
	builtinFields := map[string]unboundField{}
	for key, ec := range funcBuiltins {
		function := valueFunction{ec: ec} // TODO(sbarzowski) better way to build function value
		builtinFields[key] = &readyValue{&function}
	}

	for name, value := range builtinFields {
		obj.fields[name] = simpleObjectField{ast.ObjectFieldHidden, value}
	}
	return objVal.(*valueObject), nil
}

func evaluateStd(i *interpreter) (value, error) {
	beforeStdEnv := makeEnvironment(
		bindingFrame{},
		makeUnboundSelfBinding(),
	)
	evalLoc := ast.MakeLocationRangeMessage("During evaluation of std")
	evalTrace := traceElement{loc: &evalLoc}
	node := astgen.StdAst
	i.stack.setCurrentTrace(evalTrace)
	defer i.stack.clearCurrentTrace()
	return i.EvalInCleanEnv(&beforeStdEnv, node, false)
}

func prepareExtVars(i *interpreter, ext vmExtMap, kind string) map[string]*cachedThunk {
	result := make(map[string]*cachedThunk)
	for name, content := range ext {
		if content.isCode {
			result[name] = codeToPV(i, "<"+kind+":"+name+">", content.value)
		} else {
			result[name] = readyThunk(makeValueString(content.value))
		}
	}
	return result
}

func buildObject(hide ast.ObjectFieldHide, fields map[string]value) *valueObject {
	fieldMap := simpleObjectFieldMap{}
	for name, v := range fields {
		fieldMap[name] = simpleObjectField{hide, &readyValue{v}}
	}
	return makeValueSimpleObject(bindingFrame{}, fieldMap, nil, nil)
}

func buildInterpreter(ext vmExtMap, nativeFuncs map[string]*NativeFunction, maxStack int, ic *importCache) (*interpreter, error) {
	i := interpreter{
		stack:       makeCallStack(maxStack),
		importCache: ic,
		nativeFuncs: nativeFuncs,
	}

	stdObj, err := buildStdObject(&i)
	if err != nil {
		return nil, err
	}

	i.baseStd = stdObj

	i.extVars = prepareExtVars(&i, ext, "extvar")

	return &i, nil
}

func makeInitialEnv(filename string, baseStd *valueObject) environment {
	fileSpecific := buildObject(ast.ObjectFieldHidden, map[string]value{
		"thisFile": makeValueString(filename),
	})
	return makeEnvironment(
		bindingFrame{
			"std": readyThunk(makeValueExtendedObject(baseStd, fileSpecific)),
		},
		makeUnboundSelfBinding(),
	)
}

func evaluateAux(i *interpreter, node ast.Node, tla vmExtMap) (value, traceElement, error) {
	evalLoc := ast.MakeLocationRangeMessage("During evaluation")
	evalTrace := traceElement{
		loc: &evalLoc,
	}
	env := makeInitialEnv(node.Loc().FileName, i.baseStd)
	i.stack.setCurrentTrace(evalTrace)
	result, err := i.EvalInCleanEnv(&env, node, false)
	i.stack.clearCurrentTrace()
	if err != nil {
		return nil, traceElement{}, err
	}
	// If it's not a function, ignore TLA
	if f, ok := result.(*valueFunction); ok {
		toplevelArgMap := prepareExtVars(i, tla, "top-level-arg")
		args := callArguments{}
		for argName, pv := range toplevelArgMap {
			args.named = append(args.named, namedCallArgument{name: ast.Identifier(argName), pv: pv})
		}
		funcLoc := ast.MakeLocationRangeMessage("Top-level function call")
		funcTrace := traceElement{
			loc: &funcLoc,
		}
		i.stack.setCurrentTrace(funcTrace)
		result, err = f.call(i, args)
		i.stack.clearCurrentTrace()
		if err != nil {
			return nil, traceElement{}, err
		}
	}
	manifestationLoc := ast.MakeLocationRangeMessage("During manifestation")
	manifestationTrace := traceElement{
		loc: &manifestationLoc,
	}
	return result, manifestationTrace, nil
}

// TODO(sbarzowski) this function takes far too many arguments - build interpreter in vm instead
func evaluate(node ast.Node, ext vmExtMap, tla vmExtMap, nativeFuncs map[string]*NativeFunction,
	maxStack int, ic *importCache, stringOutputMode bool) (string, error) {

	i, err := buildInterpreter(ext, nativeFuncs, maxStack, ic)
	if err != nil {
		return "", err
	}

	result, manifestationTrace, err := evaluateAux(i, node, tla)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	i.stack.setCurrentTrace(manifestationTrace)
	if stringOutputMode {
		err = i.manifestString(&buf, result)
	} else {
		err = i.manifestAndSerializeJSON(&buf, result, true, "")
	}
	i.stack.clearCurrentTrace()
	if err != nil {
		return "", err
	}
	buf.WriteString("\n")
	return buf.String(), nil
}

// TODO(sbarzowski) this function takes far too many arguments - build interpreter in vm instead
func evaluateMulti(node ast.Node, ext vmExtMap, tla vmExtMap, nativeFuncs map[string]*NativeFunction,
	maxStack int, ic *importCache, stringOutputMode bool) (map[string]string, error) {

	i, err := buildInterpreter(ext, nativeFuncs, maxStack, ic)
	if err != nil {
		return nil, err
	}

	result, manifestationTrace, err := evaluateAux(i, node, tla)
	if err != nil {
		return nil, err
	}

	i.stack.setCurrentTrace(manifestationTrace)
	manifested, err := i.manifestAndSerializeMulti(result, stringOutputMode)
	i.stack.clearCurrentTrace()
	return manifested, err
}

// TODO(sbarzowski) this function takes far too many arguments - build interpreter in vm instead
func evaluateStream(node ast.Node, ext vmExtMap, tla vmExtMap, nativeFuncs map[string]*NativeFunction,
	maxStack int, ic *importCache) ([]string, error) {

	i, err := buildInterpreter(ext, nativeFuncs, maxStack, ic)
	if err != nil {
		return nil, err
	}

	result, manifestationTrace, err := evaluateAux(i, node, tla)
	if err != nil {
		return nil, err
	}

	i.stack.setCurrentTrace(manifestationTrace)
	manifested, err := i.manifestAndSerializeYAMLStream(result)
	i.stack.clearCurrentTrace()
	return manifested, err
}
