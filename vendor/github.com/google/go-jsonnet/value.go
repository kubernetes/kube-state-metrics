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
	"errors"
	"fmt"

	"github.com/google/go-jsonnet/ast"
)

// value represents a concrete jsonnet value of a specific type.
// Various operations on values are allowed, depending on their type.
// All values are of course immutable.
type value interface {
	aValue()

	getType() *valueType
}

type valueType struct {
	name string
}

var stringType = &valueType{"string"}
var numberType = &valueType{"number"}
var functionType = &valueType{"function"}
var objectType = &valueType{"object"}
var booleanType = &valueType{"boolean"}
var nullType = &valueType{"null"}
var arrayType = &valueType{"array"}

// potentialValue is something that may be evaluated to a concrete value.
// The result of the evaluation may *NOT* depend on the current state
// of the interpreter. The evaluation may fail.
//
// It can be used to represent lazy values (e.g. variables values in jsonnet
// are not calculated before they are used). It is also a useful abstraction
// in other cases like error handling.
//
// It may or may not require arbitrary computation when getValue is called the
// first time, but any subsequent calls will immediately return.
//
// TODO(sbarzowski) perhaps call it just "Thunk"?
type potentialValue interface {
	// fromWhere keeps the information from where the evaluation was requested.
	getValue(i *interpreter) (value, error)

	aPotentialValue()
}

// A set of variables with associated thunks.
type bindingFrame map[ast.Identifier]*cachedThunk

type valueBase struct{}

func (v *valueBase) aValue() {}

// Primitive values
// -------------------------------------

type valueString interface {
	value
	length() int
	getRunes() []rune
	getGoString() string
	index(i *interpreter, index int) (value, error)
}

// valueFlatString represents a string value, internally using a []rune for quick
// indexing.
type valueFlatString struct {
	valueBase
	// We use rune slices instead of strings for quick indexing
	value []rune
}

func (s *valueFlatString) index(i *interpreter, index int) (value, error) {
	if 0 <= index && index < s.length() {
		return makeValueString(string(s.value[index])), nil
	}
	return nil, i.Error(fmt.Sprintf("Index %d out of bounds, not within [0, %v)", index, s.length()))
}

func (s *valueFlatString) getRunes() []rune {
	return s.value
}

func (s *valueFlatString) getGoString() string {
	return string(s.value)
}

func (s *valueFlatString) length() int {
	return len(s.value)
}

func (s *valueFlatString) getType() *valueType {
	return stringType
}

// TODO(sbarzowski) this was chosen on a hunch, be more systematic about it
const extendedStringMinLength = 30

// Indirect representation of long strings.
// It consists of two parts, left and right, concatenation of which is the whole string.
type valueStringTree struct {
	valueBase
	left, right valueString
	len         int
}

func buildFullString(s valueString, buf *[]rune) {
	switch s := s.(type) {
	case *valueFlatString:
		*buf = append(*buf, s.value...)
	case *valueStringTree:
		buildFullString(s.left, buf)
		buildFullString(s.right, buf)
	}
}

func (s *valueStringTree) flattenToLeft() {
	if s.right != nil {
		result := make([]rune, 0, s.len)
		buildFullString(s, &result)
		s.left = makeStringFromRunes(result)
		s.right = nil
	}
}

func (s *valueStringTree) index(i *interpreter, index int) (value, error) {
	if 0 <= index && index < s.len {
		s.flattenToLeft()
		return s.left.index(i, index)
	}
	return nil, i.Error(fmt.Sprintf("Index %d out of bounds, not within [0, %v)", index, s.length()))
}

func (s *valueStringTree) getRunes() []rune {
	s.flattenToLeft()
	return s.left.getRunes()
}

func (s *valueStringTree) getGoString() string {
	s.flattenToLeft()
	return s.left.getGoString()
}

func (s *valueStringTree) length() int {
	return s.len
}

func (s *valueStringTree) getType() *valueType {
	return stringType
}

func emptyString() *valueFlatString {
	return &valueFlatString{}
}

func makeStringFromRunes(runes []rune) *valueFlatString {
	return &valueFlatString{value: runes}
}

func concatStrings(a, b valueString) valueString {
	aLen := a.length()
	bLen := b.length()
	if aLen == 0 {
		return b
	} else if bLen == 0 {
		return a
	} else if aLen+bLen < extendedStringMinLength {
		runesA := a.getRunes()
		runesB := b.getRunes()
		result := make([]rune, 0, aLen+bLen)
		result = append(result, runesA...)
		result = append(result, runesB...)
		return makeStringFromRunes(result)
	} else {
		return &valueStringTree{
			left:  a,
			right: b,
			len:   aLen + bLen,
		}
	}
}

func stringCmp(a, b valueString) int {
	runesA := a.getRunes()
	runesB := b.getRunes()
	var length int
	if len(runesA) < len(runesB) {
		length = len(runesA)
	} else {
		length = len(runesB)
	}
	for i := 0; i < length; i++ {
		if runesA[i] != runesB[i] {
			return runeCmp(runesA[i], runesB[i])
		}
	}
	return intCmp(len(runesA), len(runesB))
}

func stringEqual(a, b valueString) bool {
	runesA := a.getRunes()
	runesB := b.getRunes()
	if len(runesA) != len(runesB) {
		return false
	}
	for i := 0; i < len(runesA); i++ {
		if runesA[i] != runesB[i] {
			return false
		}
	}
	return true
}

func makeValueString(v string) valueString {
	return &valueFlatString{value: []rune(v)}
}

type valueBoolean struct {
	valueBase
	value bool
}

func (*valueBoolean) getType() *valueType {
	return booleanType
}

func makeValueBoolean(v bool) *valueBoolean {
	return &valueBoolean{value: v}
}

type valueNumber struct {
	valueBase
	value float64
}

func (*valueNumber) getType() *valueType {
	return numberType
}

func makeValueNumber(v float64) *valueNumber {
	return &valueNumber{value: v}
}

func intToValue(i int) *valueNumber {
	return makeValueNumber(float64(i))
}

func int64ToValue(i int64) *valueNumber {
	return makeValueNumber(float64(i))
}

type valueNull struct {
	valueBase
}

var nullValue valueNull

func makeValueNull() *valueNull {
	return &nullValue
}

func (*valueNull) getType() *valueType {
	return nullType
}

// ast.Array
// -------------------------------------

type valueArray struct {
	valueBase
	elements []*cachedThunk
}

func (arr *valueArray) index(i *interpreter, index int) (value, error) {
	if 0 <= index && index < arr.length() {
		return i.evaluatePV(arr.elements[index])
	}
	return nil, i.Error(fmt.Sprintf("Index %d out of bounds, not within [0, %v)", index, arr.length()))
}

func (arr *valueArray) length() int {
	return len(arr.elements)
}

func makeValueArray(elements []*cachedThunk) *valueArray {
	// We don't want to keep a bigger array than necessary
	// so we create a new one with minimal capacity
	var arrayElems []*cachedThunk
	if len(elements) == cap(elements) {
		arrayElems = elements
	} else {
		arrayElems = make([]*cachedThunk, len(elements))
		copy(arrayElems, elements)
	}
	return &valueArray{
		elements: arrayElems,
	}
}

func concatArrays(a, b *valueArray) *valueArray {
	result := make([]*cachedThunk, 0, len(a.elements)+len(b.elements))
	result = append(result, a.elements...)
	result = append(result, b.elements...)
	return &valueArray{elements: result}
}

func (*valueArray) getType() *valueType {
	return arrayType
}

// ast.Function
// -------------------------------------

type valueFunction struct {
	valueBase
	ec evalCallable
}

// TODO(sbarzowski) better name?
type evalCallable interface {
	evalCall(args callArguments, i *interpreter) (value, error)
	parameters() []namedParameter
}

func (f *valueFunction) call(i *interpreter, args callArguments) (value, error) {
	err := checkArguments(i, args, f.parameters())
	if err != nil {
		return nil, err
	}
	return f.ec.evalCall(args, i)
}

func (f *valueFunction) parameters() []namedParameter {
	return f.ec.parameters()
}

func checkArguments(i *interpreter, args callArguments, params []namedParameter) error {

	numPassed := len(args.positional)
	maxExpected := len(params)

	if numPassed > maxExpected {
		return i.Error(fmt.Sprintf("function expected %v positional argument(s), but got %v", maxExpected, numPassed))
	}

	// Parameter names the function will accept.
	accepted := make(map[ast.Identifier]bool)
	for _, param := range params {
		accepted[param.name] = true
	}

	// Parameter names the call will bind.
	received := make(map[ast.Identifier]bool)
	for i := range args.positional {
		received[params[i].name] = true
	}
	for _, arg := range args.named {
		if _, present := received[arg.name]; present {
			return i.Error(fmt.Sprintf("Argument %v already provided", arg.name))
		}
		if _, present := accepted[arg.name]; !present {
			return i.Error(fmt.Sprintf("function has no parameter %v", arg.name))
		}
		received[arg.name] = true
	}

	for _, param := range params {
		if _, present := received[param.name]; !present && param.defaultArg == nil {
			return i.Error(fmt.Sprintf("Missing argument: %v", param.name))
		}
	}

	return nil
}

func (f *valueFunction) getType() *valueType {
	return functionType
}

type namedParameter struct {
	name       ast.Identifier
	defaultArg ast.Node
}

type callArguments struct {
	positional []*cachedThunk
	named      []namedCallArgument
	tailstrict bool
}

type namedCallArgument struct {
	name ast.Identifier
	pv   *cachedThunk
}

func args(xs ...*cachedThunk) callArguments {
	return callArguments{positional: xs}
}

// Objects
// -------------------------------------

// Object is a value that allows indexing (taking a value of a field)
// and combining through mixin inheritence (operator +).

type valueObject struct {
	valueBase
	assertionError error
	cache          map[objectCacheKey]value
	uncached       uncachedObject
}

// Hack - we need to distinguish not-checked-yet and no error situations
// so we have a special value for no error and nil means that we don't know yet.
var errNoErrorInObjectInvariants = errors.New("no error - assertions passed")

type objectCacheKey struct {
	field string
	depth int
}

type selfBinding struct {
	// self is the lexically nearest object we are in, or nil.  Note
	// that this is not the same as context, because we could be inside a function,
	// inside an object and then context would be the function, but self would still point
	// to the object.
	self *valueObject

	// superDepth is the "super" level of self.  Sometimes, we look upwards in the
	// inheritance tree, e.g. via an explicit use of super, or because a given field
	// has been inherited.  When evaluating a field from one of these super objects,
	// we need to bind self to the concrete object (so self must point
	// there) but uses of super should be resolved relative to the object whose
	// field we are evaluating.  Thus, we keep a second field for that.  This is
	// usually 0, unless we are evaluating a super object's field.
	// TODO(sbarzowski) provide some examples
	// TODO(sbarzowski) provide somewhere a complete explanation of the object model
	superDepth int
}

func makeUnboundSelfBinding() selfBinding {
	return selfBinding{
		nil,
		123456789, // poison value
	}
}

func objectBinding(obj *valueObject) selfBinding {
	return selfBinding{self: obj, superDepth: 0}
}

func (sb selfBinding) super() selfBinding {
	return selfBinding{self: sb.self, superDepth: sb.superDepth + 1}
}

// hidden represents wether to include hidden fields in a lookup.
type hidden int

// With/without hidden fields
const (
	withHidden hidden = iota
	withoutHidden
)

func withHiddenFromBool(with bool) hidden {
	if with {
		return withHidden
	}
	return withoutHidden
}

func (*valueObject) getType() *valueType {
	return objectType
}

func (obj *valueObject) index(i *interpreter, field string) (value, error) {
	return objectIndex(i, objectBinding(obj), field)
}

func (obj *valueObject) assertionsChecked() bool {
	// nil - not checked yet
	// errNoErrorInObjectInvariants - we checked and there is no error (or checking in progress)
	return obj.assertionError != nil
}

func (obj *valueObject) setAssertionsCheckResult(err error) {
	if err != nil {
		obj.assertionError = err
	} else {
		obj.assertionError = errNoErrorInObjectInvariants
	}
}

func (obj *valueObject) getAssertionsCheckResult() error {
	if obj.assertionError == nil {
		panic("Assertions not checked yet")
	}
	if obj.assertionError == errNoErrorInObjectInvariants {
		return nil
	}
	return obj.assertionError
}

type uncachedObject interface {
	inheritanceSize() int
}

type objectLocal struct {
	name ast.Identifier
	// Locals may depend on self and super so they are unbound fields and not simply thunks
	node ast.Node
}

// simpleObject represents a flat object (no inheritance).
// Note that it can be used as part of extended objects
// in inheritance using operator +.
//
// Fields are late bound (to object), so they are not values or potentialValues.
// This is important for inheritance, for example:
// Let a = {x: 42} and b = {y: self.x}. Evaluating b.y is an error,
// but (a+b).y evaluates to 42.
type simpleObject struct {
	upValues bindingFrame
	fields   simpleObjectFieldMap
	asserts  []unboundField
	locals   []objectLocal
}

func checkAssertionsHelper(i *interpreter, obj *valueObject, curr uncachedObject, superDepth int) error {
	switch curr := curr.(type) {
	case *extendedObject:
		err := checkAssertionsHelper(i, obj, curr.right, superDepth)
		if err != nil {
			return err
		}
		err = checkAssertionsHelper(i, obj, curr.left, superDepth+curr.right.inheritanceSize())
		if err != nil {
			return err
		}
		return nil
	case *simpleObject:
		for _, assert := range curr.asserts {
			sb := selfBinding{self: obj, superDepth: superDepth}
			fieldUpValues := prepareFieldUpvalues(sb, curr.upValues, curr.locals)
			_, err := assert.evaluate(i, sb, fieldUpValues, "")
			if err != nil {
				return err
			}
		}
		return nil
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("Unknown object type %#v", curr))
	}
}

func checkAssertions(i *interpreter, obj *valueObject) error {
	if !obj.assertionsChecked() {
		// Assertions may refer to the object that will normally
		// trigger checking of assertions, resulting in an endless recursion.
		// To avoid that, while we check them, we treat them as already passed.
		obj.setAssertionsCheckResult(errNoErrorInObjectInvariants)
		obj.setAssertionsCheckResult(checkAssertionsHelper(i, obj, obj.uncached, 0))
	}
	return obj.getAssertionsCheckResult()
}

func (*simpleObject) inheritanceSize() int {
	return 1
}

func makeValueSimpleObject(b bindingFrame, fields simpleObjectFieldMap, asserts []unboundField, locals []objectLocal) *valueObject {
	return &valueObject{
		cache: make(map[objectCacheKey]value),
		uncached: &simpleObject{
			upValues: b,
			fields:   fields,
			asserts:  asserts,
			locals:   locals,
		},
	}
}

type simpleObjectFieldMap map[string]simpleObjectField

type simpleObjectField struct {
	hide  ast.ObjectFieldHide
	field unboundField
}

// unboundField is a field that doesn't know yet in which object it is.
type unboundField interface {
	evaluate(i *interpreter, sb selfBinding, origBinding bindingFrame, fieldName string) (value, error)
}

// extendedObject represents an object created through inheritance (left + right).
// We represent it as the pair of objects. This results in a tree-like structure.
// Example:
// (A + B) + C
//
//        +
//       / \
//      +   C
//     / \
//    A   B
//
// It is possible to create an arbitrary binary tree.
// Note however, that because + is associative the only thing that matters
// is the order of leafs.
//
// This represenation allows us to implement "+" in O(1),
// but requires going through the tree and trying subsequent leafs for field access.
//
type extendedObject struct {
	left, right          uncachedObject
	totalInheritanceSize int
}

func (o *extendedObject) inheritanceSize() int {
	return o.totalInheritanceSize
}

func makeValueExtendedObject(left, right *valueObject) *valueObject {
	return &valueObject{
		cache: make(map[objectCacheKey]value),
		uncached: &extendedObject{
			left:                 left.uncached,
			right:                right.uncached,
			totalInheritanceSize: left.uncached.inheritanceSize() + right.uncached.inheritanceSize(),
		},
	}
}

// findField returns a field in object curr, with superDepth at least minSuperDepth
// It also returns an associated bindingFrame and actual superDepth that the field
// was found at.
func findField(curr uncachedObject, minSuperDepth int, f string) (bool, simpleObjectField, bindingFrame, []objectLocal, int) {
	switch curr := curr.(type) {
	case *extendedObject:
		if curr.right.inheritanceSize() > minSuperDepth {
			found, field, frame, locals, counter := findField(curr.right, minSuperDepth, f)
			if found {
				return true, field, frame, locals, counter
			}
		}
		found, field, frame, locals, counter := findField(curr.left, minSuperDepth-curr.right.inheritanceSize(), f)
		return found, field, frame, locals, counter + curr.right.inheritanceSize()

	case *simpleObject:
		if minSuperDepth <= 0 {
			if field, ok := curr.fields[f]; ok {
				return true, field, curr.upValues, curr.locals, 0
			}
		}
		return false, simpleObjectField{}, nil, nil, 0
	default:
		panic(fmt.Sprintf("Unknown object type %#v", curr))
	}
}

func prepareFieldUpvalues(sb selfBinding, upValues bindingFrame, locals []objectLocal) bindingFrame {
	newUpValues := make(bindingFrame)
	for k, v := range upValues {
		newUpValues[k] = v
	}
	localThunks := make([]*cachedThunk, 0, len(locals))
	for _, l := range locals {
		th := &cachedThunk{
			// We will fill upValues later
			env:  &environment{upValues: nil, selfBinding: sb},
			body: l.node,
		}
		newUpValues[l.name] = th
		localThunks = append(localThunks, th)
	}
	for _, th := range localThunks {
		th.env.upValues = newUpValues
	}
	return newUpValues
}

func objectIndex(i *interpreter, sb selfBinding, fieldName string) (value, error) {
	err := checkAssertions(i, sb.self)
	if err != nil {
		return nil, err
	}
	if sb.superDepth >= sb.self.uncached.inheritanceSize() {
		return nil, i.Error("Attempt to use super when there is no super class.")
	}

	found, field, upValues, locals, foundAt := findField(sb.self.uncached, sb.superDepth, fieldName)
	if !found {
		return nil, i.Error(fmt.Sprintf("Field does not exist: %s", fieldName))
	}

	if val, ok := sb.self.cache[objectCacheKey{field: fieldName, depth: foundAt}]; ok {
		return val, nil
	}

	fieldSelfBinding := selfBinding{self: sb.self, superDepth: foundAt}
	fieldUpValues := prepareFieldUpvalues(fieldSelfBinding, upValues, locals)

	val, err := field.field.evaluate(i, fieldSelfBinding, fieldUpValues, fieldName)

	if err == nil {
		sb.self.cache[objectCacheKey{field: fieldName, depth: foundAt}] = val
	}

	return val, err
}

func objectHasField(sb selfBinding, fieldName string, h hidden) bool {
	found, field, _, _, _ := findField(sb.self.uncached, sb.superDepth, fieldName)
	if !found || (h == withoutHidden && field.hide == ast.ObjectFieldHidden) {
		return false
	}
	return true
}

type fieldHideMap map[string]ast.ObjectFieldHide

func uncachedObjectFieldsVisibility(obj uncachedObject) fieldHideMap {
	r := make(fieldHideMap)
	switch obj := obj.(type) {
	case *extendedObject:
		r = uncachedObjectFieldsVisibility(obj.left)
		rightMap := uncachedObjectFieldsVisibility(obj.right)
		for k, v := range rightMap {
			if v == ast.ObjectFieldInherit {
				if _, alreadyExists := r[k]; !alreadyExists {
					r[k] = v
				}
			} else {
				r[k] = v
			}
		}
		return r

	case *simpleObject:
		for fieldName, field := range obj.fields {
			r[fieldName] = field.hide
		}
	}
	return r
}

func objectFieldsVisibility(obj *valueObject) fieldHideMap {
	return uncachedObjectFieldsVisibility(obj.uncached)
}

// Returns field names of an object. Gotcha: the order of fields is unpredictable.
func objectFields(obj *valueObject, h hidden) []string {
	var r []string
	for fieldName, hide := range objectFieldsVisibility(obj) {
		if h == withHidden || hide != ast.ObjectFieldHidden {
			r = append(r, fieldName)
		}
	}
	return r
}

func duplicateFieldNameErrMsg(fieldName string) string {
	return fmt.Sprintf("Duplicate field name: %s", unparseString(fieldName))
}
