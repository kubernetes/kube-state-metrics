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

package parser

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
)

var topLevelContext = "$"

const anonymous = "anonymous"

// TODO(sbarzowski) The children functions should definitely be moved to ast
// package or a separate internal astutils package. The only reason I'm not doing it
// right now is that it's a pretty invasive change that deserves a separate PR.

// directChildren are children of AST node that are executed in the same context
// and environment as their parent. It supports ASTs before and after desugaring.
//
// They must satisfy the following rules:
// * (no-delayed-evaluation) They are evaluated when their parent is evaluated or never.
// * (no-indirect-evaluation) They cannot be evaluated during evaluation of any non-direct children
// * (same-environment) They must be evaluated in the same environment as their parent
func directChildren(node ast.Node) []ast.Node {
	switch node := node.(type) {
	case *ast.Apply:
		return []ast.Node{node.Target}
		// TODO(sbarzowski) tailstrict call arguments (once we have tailstrict)
	case *ast.ApplyBrace:
		return []ast.Node{node.Left, node.Right}
	case *ast.Array:
		return nil
	case *ast.Assert:
		if node.Message != nil {
			return []ast.Node{node.Cond, node.Message, node.Rest}
		}
		return []ast.Node{node.Cond, node.Rest}
	case *ast.Binary:
		return []ast.Node{node.Left, node.Right}
	case *ast.Conditional:
		if node.BranchFalse != nil {
			return []ast.Node{node.Cond, node.BranchTrue, node.BranchFalse}
		}
		return []ast.Node{node.Cond, node.BranchTrue}
	case *ast.Dollar:
		return nil
	case *ast.Error:
		return []ast.Node{node.Expr}
	case *ast.Function:
		return nil
	case *ast.Import:
		return nil
	case *ast.ImportStr:
		return nil
	case *ast.Index:
		if node.Id != nil {
			return nil // non-desugared dot reference
		}
		return []ast.Node{node.Target, node.Index}
	case *ast.Slice:
		var params []ast.Node
		if node.Target != nil {
			params = append(params, node.Target)
		}
		if node.BeginIndex != nil {
			params = append(params, node.BeginIndex)
		}
		if node.EndIndex != nil {
			params = append(params, node.EndIndex)
		}
		return params
	case *ast.Local:
		return nil
	case *ast.LiteralBoolean:
		return nil
	case *ast.LiteralNull:
		return nil
	case *ast.LiteralNumber:
		return nil
	case *ast.LiteralString:
		return nil
	case *ast.Object:
		return objectFieldsDirectChildren(node.Fields)
	case *ast.DesugaredObject:
		return desugaredObjectDirectChildren(node)
	case *ast.ArrayComp:
		result := []ast.Node{}
		spec := &node.Spec
		for spec != nil {
			result = append(result, spec.Expr)
			for _, ifspec := range spec.Conditions {
				result = append(result, ifspec.Expr)
			}
			spec = spec.Outer
		}
		return result
	case *ast.ObjectComp:
		result := objectFieldsDirectChildren(node.Fields)
		spec := &node.Spec
		for spec != nil {
			result = append(result, spec.Expr)
			for _, ifspec := range spec.Conditions {
				result = append(result, ifspec.Expr)
			}
			spec = spec.Outer
		}
		return result
	case *ast.Parens:
		return []ast.Node{node.Inner}
	case *ast.Self:
		return nil
	case *ast.SuperIndex:
		if node.Id != nil {
			return nil
		}
		return []ast.Node{node.Index}
	case *ast.InSuper:
		return []ast.Node{node.Index}
	case *ast.Unary:
		return []ast.Node{node.Expr}
	case *ast.Var:
		return nil
	}
	panic(fmt.Sprintf("directChildren: Unknown node %#v", node))
}

// thunkChildren are children of AST node that are executed in a new context
// and capture environment from parent (thunked).
//
// It supports ASTs before and after desugaring.
//
// TODO(sbarzowski) Make sure it works well with boundary cases like tailstrict arguments,
//					make it more precise.
// Rules:
// * (same-environment) They must be evaluated in the same environment as their parent
// * (not-direct) If they can be direct children, they should (and cannot be thunked).
func thunkChildren(node ast.Node) []ast.Node {
	switch node := node.(type) {
	case *ast.Apply:
		var nodes []ast.Node
		for _, arg := range node.Arguments.Positional {
			nodes = append(nodes, arg)
		}
		for _, arg := range node.Arguments.Named {
			nodes = append(nodes, arg.Arg)
		}
		return nodes
	case *ast.ApplyBrace:
		return nil
	case *ast.Array:
		return node.Elements
	case *ast.Assert:
		return nil
	case *ast.Binary:
		return nil
	case *ast.Conditional:
		return nil
	case *ast.Dollar:
		return nil
	case *ast.Error:
		return nil
	case *ast.Function:
		return nil
	case *ast.Import:
		return nil
	case *ast.ImportStr:
		return nil
	case *ast.Index:
		return nil
	case *ast.Slice:
		return nil
	case *ast.Local:
		// TODO(sbarzowski) complicated
		return nil
	case *ast.LiteralBoolean:
		return nil
	case *ast.LiteralNull:
		return nil
	case *ast.LiteralNumber:
		return nil
	case *ast.LiteralString:
		return nil
	case *ast.DesugaredObject:
		return nil
	case *ast.Object:
		return nil
	case *ast.ArrayComp:
		return []ast.Node{node.Body}
	case *ast.ObjectComp:
		return nil
	case *ast.Parens:
		return nil
	case *ast.Self:
		return nil
	case *ast.SuperIndex:
		return nil
	case *ast.InSuper:
		return nil
	case *ast.Unary:
		return nil
	case *ast.Var:
		return nil
	}
	panic(fmt.Sprintf("thunkChildren: Unknown node %#v", node))
}

func objectFieldsDirectChildren(fields ast.ObjectFields) ast.Nodes {
	result := ast.Nodes{}
	for _, field := range fields {
		if field.Expr1 != nil {
			result = append(result, field.Expr1)
		}
	}
	return result
}

func inObjectFieldsChildren(fields ast.ObjectFields) ast.Nodes {
	result := ast.Nodes{}
	for _, field := range fields {
		if field.MethodSugar {
			result = append(result, field.Method)
		} else {
			if field.Expr2 != nil {
				result = append(result, field.Expr2)
			}
			if field.Expr3 != nil {
				result = append(result, field.Expr3)
			}
		}
	}
	return result
}

func desugaredObjectDirectChildren(obj *ast.DesugaredObject) ast.Nodes {
	result := ast.Nodes{}
	for _, field := range obj.Fields {
		if field.Name == nil {
			panic("Name cannot be nil")
		}
		result = append(result, field.Name)
	}
	return result
}

func inDesugaredObjectSpecialChildren(obj *ast.DesugaredObject) ast.Nodes {
	result := make([]ast.Node, 0, len(obj.Fields)+len(obj.Locals))
	for _, field := range obj.Fields {
		result = append(result, field.Body)
	}
	for _, local := range obj.Locals {
		result = append(result, local.Body)
	}
	return result
}

// specialChildren returns children that are neither direct nor thunked,
// e.g. object field body.
// These nodes are evaluated in a different environment from their parent.
//
// It supports ASTs before and after desugaring.
func specialChildren(node ast.Node) []ast.Node {
	switch node := node.(type) {
	case *ast.Apply:
		return nil
	case *ast.ApplyBrace:
		return nil
	case *ast.Array:
		return nil
	case *ast.Assert:
		return nil
	case *ast.Binary:
		return nil
	case *ast.Conditional:
		return nil
	case *ast.Dollar:
		return nil
	case *ast.Error:
		return nil
	case *ast.Function:
		children := []ast.Node{node.Body}
		for _, child := range node.Parameters.Optional {
			children = append(children, child.DefaultArg)
		}
		return children
	case *ast.Import:
		return nil
	case *ast.ImportStr:
		return nil
	case *ast.Index:
		return nil
	case *ast.Slice:
		return nil
	case *ast.Local:
		children := make([]ast.Node, 1, len(node.Binds)+1)
		children[0] = node.Body
		for _, bind := range node.Binds {
			children = append(children, bind.Body)
		}
		return children
	case *ast.LiteralBoolean:
		return nil
	case *ast.LiteralNull:
		return nil
	case *ast.LiteralNumber:
		return nil
	case *ast.LiteralString:
		return nil
	case *ast.DesugaredObject:
		return inDesugaredObjectSpecialChildren(node)
	case *ast.Object:
		return inObjectFieldsChildren(node.Fields)
	case *ast.ArrayComp:
		return []ast.Node{node.Body}
	case *ast.ObjectComp:
		return inObjectFieldsChildren(node.Fields)
	case *ast.Self:
		return nil
	case *ast.SuperIndex:
		return nil
	case *ast.InSuper:
		return nil
	case *ast.Unary:
		return nil
	case *ast.Var:
		return nil
	case *ast.Parens:
		return nil
	}
	panic(fmt.Sprintf("specialChildren: Unknown node %#v", node))
}

// Children returns all children of a node. It supports ASTs before and after desugaring.
func Children(node ast.Node) []ast.Node {
	var result []ast.Node
	result = append(result, directChildren(node)...)
	result = append(result, thunkChildren(node)...)
	result = append(result, specialChildren(node)...)
	return result
}

func functionContext(funcName string) *string {
	r := "function <" + funcName + ">"
	return &r
}

func objectContext(objName string) *string {
	r := "object <" + objName + ">"
	return &r
}

// addContext adds context to a node and its whole subtree.
//
// context is the surrounding context of a node (e.g. a function it's in)
//
// bind is a name that the node is bound to, i.e. if node is a local bind body
// then bind is its name. For nodes that are not bound to variables `anonymous`
// should be passed. For example:
// local x = 2 + 2; x
// In such case bind for binary node 2 + 2 is "x" and for every other node,
// including its children, its anonymous.
func addContext(node ast.Node, context *string, bind string) {
	if node == nil {
		return
	}

	node.SetContext(context)

	switch node := node.(type) {
	case *ast.Function:
		funContext := functionContext(bind)
		addContext(node.Body, funContext, anonymous)
		for i := range node.Parameters.Optional {
			// Default arguments have the same context as the function body.
			addContext(node.Parameters.Optional[i].DefaultArg, funContext, anonymous)
		}
	case *ast.Object:
		// TODO(sbarzowski) include fieldname, maybe even chains

		outOfObject := directChildren(node)
		for _, f := range outOfObject {
			// This actually is evaluated outside of object
			addContext(f, context, anonymous)
		}

		objContext := objectContext(bind)
		inObject := inObjectFieldsChildren(node.Fields)
		for _, f := range inObject {
			// This actually is evaluated outside of object
			addContext(f, objContext, anonymous)
		}

	case *ast.ObjectComp:
		outOfObject := directChildren(node)
		for _, f := range outOfObject {
			// This actually is evaluated outside of object
			addContext(f, context, anonymous)
		}

		objContext := objectContext(bind)
		inObject := inObjectFieldsChildren(node.Fields)
		for _, f := range inObject {
			// This actually is evaluated outside of object
			addContext(f, objContext, anonymous)
		}

	case *ast.Local:
		for _, bind := range node.Binds {
			namedThunkContext := "thunk <" + string(bind.Variable) + "> from <" + *context + ">"
			if bind.Fun != nil {
				addContext(bind.Fun, &namedThunkContext, string(bind.Variable))
			} else {
				addContext(bind.Body, &namedThunkContext, string(bind.Variable))
			}
		}
		addContext(node.Body, context, bind)
	default:
		for _, child := range directChildren(node) {
			addContext(child, context, anonymous)
		}

		// TODO(sbarzowski) avoid "thunk from <thunk from..."
		thunkContext := "thunk from <" + *context + ">"
		for _, child := range thunkChildren(node) {
			addContext(child, &thunkContext, anonymous)
		}
	}
}
