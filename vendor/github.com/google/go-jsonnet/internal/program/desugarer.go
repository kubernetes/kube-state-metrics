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

package program

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"unicode/utf8"

	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/errors"
)

var desugaredBop = map[ast.BinaryOp]ast.Identifier{
	ast.BopPercent: "mod",
	ast.BopIn:      "objectHasAll",
}

func makeStr(s string) *ast.LiteralString {
	return &ast.LiteralString{
		NodeBase:    ast.NodeBase{},
		Value:       s,
		Kind:        ast.StringDouble,
		BlockIndent: "",
	}
}

func stringUnescape(loc *ast.LocationRange, s string) (string, error) {
	var buf bytes.Buffer
	// read one rune at a time
	for i := 0; i < len(s); {
		r, w := utf8.DecodeRuneInString(s[i:])
		i += w
		switch r {
		case '\\':
			if i >= len(s) {
				return "", errors.MakeStaticError("Truncated escape sequence in string literal.", *loc)
			}
			r2, w := utf8.DecodeRuneInString(s[i:])
			i += w
			switch r2 {
			case '"':
				buf.WriteRune('"')
			case '\'':
				buf.WriteRune('\'')
			case '\\':
				buf.WriteRune('\\')
			case '/':
				buf.WriteRune('/') // See json.org, \/ is a valid escape.
			case 'b':
				buf.WriteRune('\b')
			case 'f':
				buf.WriteRune('\f')
			case 'n':
				buf.WriteRune('\n')
			case 'r':
				buf.WriteRune('\r')
			case 't':
				buf.WriteRune('\t')
			case 'u':
				if i+4 > len(s) {
					return "", errors.MakeStaticError("Truncated unicode escape sequence in string literal.", *loc)
				}
				codeBytes, err := hex.DecodeString(s[i : i+4])
				if err != nil {
					return "", errors.MakeStaticError(fmt.Sprintf("Unicode escape sequence was malformed: %s", s[0:4]), *loc)
				}
				code := int(codeBytes[0])*256 + int(codeBytes[1])
				buf.WriteRune(rune(code))
				i += 4
			default:
				return "", errors.MakeStaticError(fmt.Sprintf("Unknown escape sequence in string literal: \\%c", r2), *loc)
			}

		default:
			buf.WriteRune(r)
		}
	}
	return buf.String(), nil
}

func desugarFields(nodeBase ast.NodeBase, fields *ast.ObjectFields, objLevel int) (*ast.DesugaredObject, error) {
	for i := range *fields {
		field := &((*fields)[i])
		if field.Method == nil {
			continue
		}
		field.Expr2 = field.Method
		field.Method = nil
		// Body of the function already desugared through expr2
	}

	asserts := ast.Nodes{}
	locals := ast.LocalBinds{}
	desugaredFields := ast.DesugaredObjectFields{}

	for i := range *fields {
		field := &(*fields)[i]
		switch field.Kind {
		case ast.ObjectAssert:
			msg := field.Expr3
			if msg == nil {
				msg = buildLiteralString("Object assertion failed.")
			}
			onFailure := &ast.Error{Expr: msg}
			asserts = append(asserts, &ast.Conditional{
				Cond:        field.Expr2,
				BranchTrue:  &ast.LiteralBoolean{Value: true}, // ignored anyway
				BranchFalse: onFailure,
			})
		case ast.ObjectFieldID:
			desugaredFields = append(desugaredFields, ast.DesugaredObjectField{
				Hide:      field.Hide,
				Name:      makeStr(string(*field.Id)),
				Body:      field.Expr2,
				PlusSuper: field.SuperSugar,
			})

		case ast.ObjectFieldExpr, ast.ObjectFieldStr:
			desugaredFields = append(desugaredFields, ast.DesugaredObjectField{
				Hide:      field.Hide,
				Name:      field.Expr1,
				Body:      field.Expr2,
				PlusSuper: field.SuperSugar,
			})

		case ast.ObjectLocal:
			locals = append(locals, ast.LocalBind{
				Variable: *field.Id,
				Body:     ast.Clone(field.Expr2), // TODO(sbarzowski) not sure if clone is needed
			})
		default:
			panic(fmt.Sprintf("Unexpected object field kind %v", field.Kind))
		}

	}

	// Hidden variable to allow $ binding.
	if objLevel == 0 {
		locals = append(locals, ast.LocalBind{
			Variable: ast.Identifier("$"),
			Body:     &ast.Self{},
		})
	}

	// Desugar stuff inside
	for i := range asserts {
		assert := &(asserts[i])
		err := desugar(assert, objLevel+1)
		if err != nil {
			return nil, err
		}
	}
	desugarLocalBinds(locals, objLevel+1)
	for i := range desugaredFields {
		field := &(desugaredFields[i])
		if field.Name != nil {
			err := desugar(&field.Name, objLevel)
			if err != nil {
				return nil, err
			}
		}
		err := desugar(&field.Body, objLevel+1)
		if err != nil {
			return nil, err
		}
	}

	return &ast.DesugaredObject{
		NodeBase: nodeBase,
		Asserts:  asserts,
		Locals:   locals,
		Fields:   desugaredFields,
	}, nil
}

func simpleLambda(body ast.Node, paramName ast.Identifier) ast.Node {
	return &ast.Function{
		Body:       body,
		Parameters: ast.Parameters{Required: ast.Identifiers{paramName}},
	}
}

func buildAnd(left ast.Node, right ast.Node) ast.Node {
	return &ast.Binary{Op: ast.BopAnd, Left: left, Right: right}
}

// inside is assumed to be already desugared (and cannot be desugared again)
func desugarForSpec(inside ast.Node, forSpec *ast.ForSpec, objLevel int) (ast.Node, error) {
	var body ast.Node
	if len(forSpec.Conditions) > 0 {
		cond := forSpec.Conditions[0].Expr
		for i := 1; i < len(forSpec.Conditions); i++ {
			cond = buildAnd(cond, forSpec.Conditions[i].Expr)
		}
		err := desugar(&cond, objLevel)
		if err != nil {
			return nil, err
		}
		body = &ast.Conditional{
			Cond:        cond,
			BranchTrue:  inside,
			BranchFalse: &ast.Array{},
		}
	} else {
		body = inside
	}
	function := simpleLambda(body, forSpec.VarName)
	err := desugar(&forSpec.Expr, objLevel)
	if err != nil {
		return nil, err
	}
	current := buildStdCall("flatMap", function, forSpec.Expr)
	if forSpec.Outer == nil {
		return current, nil
	}
	return desugarForSpec(current, forSpec.Outer, objLevel)
}

func wrapInArray(inside ast.Node) ast.Node {
	return &ast.Array{Elements: ast.Nodes{inside}}
}

func desugarArrayComp(comp *ast.ArrayComp, objLevel int) (ast.Node, error) {
	err := desugar(&comp.Body, objLevel)
	if err != nil {
		return nil, err
	}
	return desugarForSpec(wrapInArray(comp.Body), &comp.Spec, objLevel)
}

func desugarObjectComp(comp *ast.ObjectComp, objLevel int) (ast.Node, error) {
	obj, err := desugarFields(comp.NodeBase, &comp.Fields, objLevel)
	if err != nil {
		return nil, err
	}

	if len(obj.Fields) != 1 {
		panic("Too many fields in object comprehension, it should have been caught during parsing")
	}

	desugaredArrayComp, err := desugarForSpec(wrapInArray(obj), &comp.Spec, objLevel)
	if err != nil {
		return nil, err
	}

	desugaredComp := buildStdCall("$objectFlatMerge", desugaredArrayComp)
	return desugaredComp, nil
}

func buildLiteralString(value string) ast.Node {
	return &ast.LiteralString{
		Kind:  ast.StringDouble,
		Value: value,
	}
}

func buildSimpleIndex(obj ast.Node, member ast.Identifier) ast.Node {
	return &ast.Index{
		Target: obj,
		Index:  buildLiteralString(string(member)),
	}
}

func buildStdCall(builtinName ast.Identifier, args ...ast.Node) ast.Node {
	std := &ast.Var{Id: "std"}
	builtin := buildSimpleIndex(std, builtinName)
	return &ast.Apply{
		Target:    builtin,
		Arguments: ast.Arguments{Positional: args},
	}
}

func desugarLocalBinds(binds ast.LocalBinds, objLevel int) (err error) {
	for i := range binds {
		if binds[i].Fun != nil {
			binds[i] = ast.LocalBind{
				Variable: binds[i].Variable,
				Body:     binds[i].Fun,
				Fun:      nil,
			}
		}
		err = desugar(&binds[i].Body, objLevel)
		if err != nil {
			return
		}
	}
	return nil
}

func desugar(astPtr *ast.Node, objLevel int) (err error) {
	node := *astPtr

	if node == nil {
		return
	}

	switch node := node.(type) {
	case *ast.Apply:
		err = desugar(&node.Target, objLevel)
		if err != nil {
			return
		}
		for i := range node.Arguments.Positional {
			err = desugar(&node.Arguments.Positional[i], objLevel)
			if err != nil {
				return
			}
		}
		for i := range node.Arguments.Named {
			err = desugar(&node.Arguments.Named[i].Arg, objLevel)
			if err != nil {
				return
			}
		}

	case *ast.ApplyBrace:
		err = desugar(&node.Left, objLevel)
		if err != nil {
			return
		}
		err = desugar(&node.Right, objLevel)
		if err != nil {
			return
		}
		*astPtr = &ast.Binary{
			NodeBase: node.NodeBase,
			Left:     node.Left,
			Op:       ast.BopPlus,
			Right:    node.Right,
		}

	case *ast.Array:
		for i := range node.Elements {
			err = desugar(&node.Elements[i], objLevel)
			if err != nil {
				return
			}
		}

	case *ast.ArrayComp:
		*astPtr, err = desugarArrayComp(node, objLevel)
		if err != nil {
			return err
		}

	case *ast.Assert:
		if node.Message == nil {
			node.Message = buildLiteralString("Assertion failed")
		}
		*astPtr = &ast.Conditional{
			Cond:        node.Cond,
			BranchTrue:  node.Rest,
			BranchFalse: &ast.Error{Expr: node.Message},
		}
		err = desugar(astPtr, objLevel)
		if err != nil {
			return err
		}

	case *ast.Binary:
		// some operators get replaced by stdlib functions
		if funcname, replaced := desugaredBop[node.Op]; replaced {
			if node.Op == ast.BopIn {
				// reversed order of arguments
				*astPtr = buildStdCall(funcname, node.Right, node.Left)
			} else {
				*astPtr = buildStdCall(funcname, node.Left, node.Right)
			}
			return desugar(astPtr, objLevel)
		}

		err = desugar(&node.Left, objLevel)
		if err != nil {
			return
		}
		err = desugar(&node.Right, objLevel)
		if err != nil {
			return
		}

	case *ast.Conditional:
		err = desugar(&node.Cond, objLevel)
		if err != nil {
			return
		}
		err = desugar(&node.BranchTrue, objLevel)
		if err != nil {
			return
		}
		if node.BranchFalse == nil {
			node.BranchFalse = &ast.LiteralNull{}
		}
		err = desugar(&node.BranchFalse, objLevel)
		if err != nil {
			return
		}

	case *ast.Dollar:
		if objLevel == 0 {
			return errors.MakeStaticError("No top-level object found.", *node.Loc())
		}
		*astPtr = &ast.Var{NodeBase: node.NodeBase, Id: ast.Identifier("$")}

	case *ast.Error:
		err = desugar(&node.Expr, objLevel)
		if err != nil {
			return
		}

	case *ast.Function:
		for i := range node.Parameters.Optional {
			param := &node.Parameters.Optional[i]
			err = desugar(&param.DefaultArg, objLevel)
			if err != nil {
				return
			}
		}
		err = desugar(&node.Body, objLevel)
		if err != nil {
			return
		}

	case *ast.Import:
		// desugar() is allowed to update the pointer to point to something else, but will never do
		// this for a LiteralString.  We cannot simply do &node.File because the type is
		// **ast.LiteralString which is not compatible with *ast.Node.
		var file ast.Node = node.File
		err = desugar(&file, objLevel)
		if err != nil {
			return
		}

	case *ast.ImportStr:
		// See comment in ast.Import.
		var file ast.Node = node.File
		err = desugar(&file, objLevel)
		if err != nil {
			return
		}

	case *ast.Index:
		err = desugar(&node.Target, objLevel)
		if err != nil {
			return
		}
		if node.Id != nil {
			if node.Index != nil {
				panic(fmt.Sprintf("Node with both Id and Index: %#+v", node))
			}
			node.Index = makeStr(string(*node.Id))
			node.Id = nil
		}
		err = desugar(&node.Index, objLevel)
		if err != nil {
			return
		}

	case *ast.Slice:
		if node.BeginIndex == nil {
			node.BeginIndex = &ast.LiteralNull{}
		}
		if node.EndIndex == nil {
			node.EndIndex = &ast.LiteralNull{}
		}
		if node.Step == nil {
			node.Step = &ast.LiteralNull{}
		}
		*astPtr = buildStdCall("slice", node.Target, node.BeginIndex, node.EndIndex, node.Step)
		err = desugar(astPtr, objLevel)
		if err != nil {
			return
		}

	case *ast.Local:
		err = desugarLocalBinds(node.Binds, objLevel)
		if err != nil {
			return
		}
		err = desugar(&node.Body, objLevel)
		if err != nil {
			return
		}

	case *ast.LiteralBoolean:
		// Nothing to do.

	case *ast.LiteralNull:
		// Nothing to do.

	case *ast.LiteralNumber:
		// Nothing to do.

	case *ast.LiteralString:
		if node.Kind.FullyEscaped() {
			unescaped, err := stringUnescape(node.Loc(), node.Value)
			if err != nil {
				return err
			}
			node.Value = unescaped
		}
		node.Kind = ast.StringDouble
		node.BlockIndent = ""
	case *ast.Object:
		*astPtr, err = desugarFields(node.NodeBase, &node.Fields, objLevel)
		if err != nil {
			return
		}

	case *ast.DesugaredObject:
		// Desugaring something multiple times is a bad idea.
		// All functions here should desugar nodes in one go.
		panic("Desugaring desugared object")

	case *ast.ObjectComp:
		*astPtr, err = desugarObjectComp(node, objLevel)
		if err != nil {
			return err
		}

	case *ast.Parens:
		*astPtr = node.Inner
		err = desugar(astPtr, objLevel)
		if err != nil {
			return err
		}

	case *ast.Self:
		// Nothing to do.

	case *ast.SuperIndex:
		if node.Id != nil {
			node.Index = &ast.LiteralString{Value: string(*node.Id)}
			node.Id = nil
		}

	case *ast.InSuper:
		err := desugar(&node.Index, objLevel)
		if err != nil {
			return err
		}
	case *ast.Unary:
		err = desugar(&node.Expr, objLevel)
		if err != nil {
			return
		}

	case *ast.Var:
		// Nothing to do.

	default:
		panic(fmt.Sprintf("Desugarer does not recognize ast: %s", reflect.TypeOf(node)))
	}

	return nil
}

// desugarAST desugars Jsonnet expressions to reduce the number of constructs
// the rest of the implementation needs to understand.
//
// Note that despite the name, desugar() is not idempotent.  String literals
// have their escape codes translated to low-level characters during desugaring.
//
// Desugaring should happen immediately after parsing, i.e. before static
// analysis and execution. Temporary variables introduced here should be
// prefixed with $ to ensure they do not clash with variables used in user code.
//
// TODO(sbarzowski) Actually we may want to do some static analysis before
// desugaring, e.g. warning user about dangerous use of constructs that we
// desugar.
func desugarAST(ast *ast.Node) error {
	err := desugar(ast, 0)
	if err != nil {
		return err
	}
	return nil
}
