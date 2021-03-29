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

// Package parser reads Jsonnet files and parses them into AST nodes.
package parser

import (
	"fmt"

	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/errors"
)

type precedence int

const (
	applyPrecedence precedence = 2  // ast.Function calls and indexing.
	unaryPrecedence precedence = 4  // Logical and bitwise negation, unary + -
	maxPrecedence   precedence = 16 // ast.Local, If, ast.Import, ast.Function, Error
)

var bopPrecedence = map[ast.BinaryOp]precedence{
	ast.BopMult:            5,
	ast.BopDiv:             5,
	ast.BopPercent:         5,
	ast.BopPlus:            6,
	ast.BopMinus:           6,
	ast.BopShiftL:          7,
	ast.BopShiftR:          7,
	ast.BopGreater:         8,
	ast.BopGreaterEq:       8,
	ast.BopLess:            8,
	ast.BopLessEq:          8,
	ast.BopIn:              8,
	ast.BopManifestEqual:   9,
	ast.BopManifestUnequal: 9,
	ast.BopBitwiseAnd:      10,
	ast.BopBitwiseXor:      11,
	ast.BopBitwiseOr:       12,
	ast.BopAnd:             13,
	ast.BopOr:              14,
}

// ---------------------------------------------------------------------------

func makeUnexpectedError(t *token, while string) errors.StaticError {
	return errors.MakeStaticError(
		fmt.Sprintf("Unexpected: %v", t), t.loc).WithContext(while)
}

func locFromTokens(begin, end *token) ast.LocationRange {
	return ast.LocationRangeBetween(&begin.loc, &end.loc)
}

func locFromTokenAST(begin *token, end ast.Node) ast.LocationRange {
	return ast.LocationRangeBetween(&begin.loc, end.Loc())
}

// ---------------------------------------------------------------------------

type parser struct {
	t     Tokens
	currT int
}

func makeParser(t Tokens) *parser {
	return &parser{
		t: t,
	}
}

func (p *parser) pop() *token {
	t := &p.t[p.currT]
	p.currT++
	return t
}

func (p *parser) unexpectedTokenError(tk tokenKind, t *token) errors.StaticError {
	if tk == t.kind {
		panic("Unexpectedly expected token kind")
	}
	return errors.MakeStaticError(fmt.Sprintf("Expected token %v but got %v", tk, t), t.loc)
}

func (p *parser) popExpect(tk tokenKind) (*token, errors.StaticError) {
	t := p.pop()
	if t.kind != tk {
		return nil, p.unexpectedTokenError(tk, t)
	}
	return t, nil
}

func (p *parser) popExpectOp(op string) (*token, errors.StaticError) {
	t := p.pop()
	if t.kind != tokenOperator || t.data != op {
		return nil, errors.MakeStaticError(
			fmt.Sprintf("Expected operator %v but got %v", op, t), t.loc)
	}
	return t, nil
}

func (p *parser) peek() *token {
	return &p.t[p.currT]
}

func (p *parser) doublePeek() *token {
	return &p.t[p.currT+1]
}

// parseArgument parses either <f1> id <f2> = expr or just expr.
// It returns either (<f1>, id, <f2>, expr) or (nil, nil, nil, expr)
// respectively.
func (p *parser) parseArgument() (ast.Fodder, *ast.Identifier, ast.Fodder, ast.Node, errors.StaticError) {
	var idFodder ast.Fodder
	var id *ast.Identifier
	var eqFodder ast.Fodder
	if p.peek().kind == tokenIdentifier && p.doublePeek().kind == tokenOperator && p.doublePeek().data == "=" {
		ident := p.pop()
		var tmpID = ast.Identifier(ident.data)
		id = &tmpID
		idFodder = ident.fodder
		eq := p.pop()
		eqFodder = eq.fodder
	}
	expr, err := p.parse(maxPrecedence)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return idFodder, id, eqFodder, expr, nil
}

// TODO(sbarzowski) - this returned bool is weird
func (p *parser) parseArguments(elementKind string) (*token, *ast.Arguments, bool, errors.StaticError) {
	args := &ast.Arguments{}
	gotComma := false
	namedArgumentAdded := false
	first := true
	for {
		commaFodder := ast.Fodder{}
		next := p.peek()

		if next.kind == tokenParenR {
			// gotComma can be true or false here.
			return p.pop(), args, gotComma, nil
		}

		if !first && !gotComma {
			return nil, nil, false, errors.MakeStaticError(fmt.Sprintf("Expected a comma before next %s, got %s", elementKind, next), next.loc)
		}

		idFodder, id, eqFodder, expr, err := p.parseArgument()
		if err != nil {
			return nil, nil, false, err
		}

		if p.peek().kind == tokenComma {
			comma := p.pop()
			gotComma = true
			commaFodder = comma.fodder
		} else {
			gotComma = false
		}

		if id == nil {
			if namedArgumentAdded {
				return nil, nil, false, errors.MakeStaticError("Positional argument after a named argument is not allowed", next.loc)
			}
			el := ast.CommaSeparatedExpr{Expr: expr}
			if gotComma {
				el.CommaFodder = commaFodder
			}
			args.Positional = append(args.Positional, el)
		} else {
			namedArgumentAdded = true
			args.Named = append(args.Named, ast.NamedArgument{
				NameFodder:  idFodder,
				Name:        *id,
				EqFodder:    eqFodder,
				Arg:         expr,
				CommaFodder: commaFodder,
			})
		}

		first = false
	}
}

// parseParameter parses either <f1> id <f2> = expr or just <f1> id.
// It returns either (<f1>, id, <f2>, expr) or (<f1>, id, nil, nil)
// respectively.
func (p *parser) parseParameter() (ast.Parameter, errors.StaticError) {
	ret := ast.Parameter{}
	ident, err := p.popExpect(tokenIdentifier)
	if err != nil {
		return ret, err.WithContext("parsing parameter")
	}
	ret.Name = ast.Identifier(ident.data)
	ret.NameFodder = ident.fodder
	ret.LocRange = ident.loc
	if p.peek().kind == tokenOperator && p.peek().data == "=" {
		eq := p.pop()
		ret.EqFodder = eq.fodder
		ret.DefaultArg, err = p.parse(maxPrecedence)
		if err != nil {
			return ret, err
		}
		ret.LocRange = locFromTokenAST(ident, ret.DefaultArg)
	}
	return ret, nil
}

// TODO(sbarzowski) - this returned bool is weird
func (p *parser) parseParameters(elementKind string) (*token, []ast.Parameter, bool, errors.StaticError) {

	var parenR *token
	var params []ast.Parameter
	gotComma := false
	first := true
	for {
		next := p.peek()

		if next.kind == tokenParenR {
			// gotComma can be true or false here.
			parenR = p.pop()
			break
		}

		if !first && !gotComma {
			return nil, nil, false, errors.MakeStaticError(fmt.Sprintf("Expected a comma before next %s, got %s", elementKind, next), next.loc)
		}

		param, err := p.parseParameter()
		if err != nil {
			return nil, nil, false, err
		}

		if p.peek().kind == tokenComma {
			comma := p.pop()
			param.CommaFodder = comma.fodder
			gotComma = true
		} else {
			gotComma = false
		}
		params = append(params, param)

		first = false
	}

	return parenR, params, gotComma, nil
}

// TODO(sbarzowski) add location to all individual binds
func (p *parser) parseBind(binds *ast.LocalBinds) (*token, errors.StaticError) {
	varID, popErr := p.popExpect(tokenIdentifier)
	if popErr != nil {
		return nil, popErr
	}
	for _, b := range *binds {
		if b.Variable == ast.Identifier(varID.data) {
			return nil, errors.MakeStaticError(fmt.Sprintf("Duplicate local var: %v", varID.data), varID.loc)
		}
	}

	var fun *ast.Function
	if p.peek().kind == tokenParenL {
		parenL := p.pop()
		parenR, params, gotComma, err := p.parseParameters("function parameter")
		if err != nil {
			return nil, err
		}
		fun = &ast.Function{
			ParenLeftFodder:  parenL.fodder,
			Parameters:       params,
			TrailingComma:    gotComma,
			ParenRightFodder: parenR.fodder,
			// Body gets filled in later.
		}
	}

	eqToken, popErr := p.popExpectOp("=")
	if popErr != nil {
		return nil, popErr
	}
	body, err := p.parse(maxPrecedence)
	if err != nil {
		return nil, err
	}

	delim := p.pop()
	if delim.kind != tokenSemicolon && delim.kind != tokenComma {
		return nil, errors.MakeStaticError(fmt.Sprintf("Expected , or ; but got %v", delim), delim.loc)
	}

	if fun != nil {
		fun.NodeBase = ast.NewNodeBaseLoc(locFromTokenAST(varID, body), nil)
		fun.Body = body
		*binds = append(*binds, ast.LocalBind{
			VarFodder:   varID.fodder,
			Variable:    ast.Identifier(varID.data),
			EqFodder:    eqToken.fodder,
			Body:        body,
			Fun:         fun,
			CloseFodder: delim.fodder,
			LocRange:    locFromTokenAST(varID, body),
		})
	} else {
		*binds = append(*binds, ast.LocalBind{
			VarFodder:   varID.fodder,
			Variable:    ast.Identifier(varID.data),
			EqFodder:    eqToken.fodder,
			Body:        body,
			CloseFodder: delim.fodder,
			LocRange:    locFromTokenAST(varID, body),
		})
	}

	return delim, nil
}

func (p *parser) parseObjectAssignmentOp() (opFodder ast.Fodder, plusSugar bool, hide ast.ObjectFieldHide, err errors.StaticError) {
	op, err := p.popExpect(tokenOperator)
	if err != nil {
		return
	}
	opFodder = op.fodder
	opStr := op.data
	if opStr[0] == '+' {
		plusSugar = true
		opStr = opStr[1:]
	}

	numColons := 0
	for len(opStr) > 0 {
		if opStr[0] != ':' {
			err = errors.MakeStaticError(
				fmt.Sprintf("Expected one of :, ::, :::, +:, +::, +:::, got: %v", op.data), op.loc)
			return
		}
		opStr = opStr[1:]
		numColons++
	}

	switch numColons {
	case 1:
		hide = ast.ObjectFieldInherit
	case 2:
		hide = ast.ObjectFieldHidden
	case 3:
		hide = ast.ObjectFieldVisible
	default:
		err = errors.MakeStaticError(
			fmt.Sprintf("Expected one of :, ::, :::, +:, +::, +:::, got: %v", op.data), op.loc)
		return
	}

	return
}

// A LiteralField is a field of an object or object comprehension.
// +gen set
type LiteralField string

func (p *parser) parseObjectRemainderComp(fields ast.ObjectFields, gotComma bool, tok *token, next *token) (ast.Node, *token, errors.StaticError) {
	numFields := 0
	numAsserts := 0
	var field ast.ObjectField
	for _, f := range fields {
		if f.Kind == ast.ObjectLocal {
			continue
		}
		if f.Kind == ast.ObjectAssert {
			numAsserts++
			continue
		}
		numFields++
		field = f
	}

	if numAsserts > 0 {
		return nil, nil, errors.MakeStaticError("Object comprehension cannot have asserts", next.loc)
	}
	if numFields != 1 {
		return nil, nil, errors.MakeStaticError("Object comprehension can only have one field", next.loc)
	}
	if field.Hide != ast.ObjectFieldInherit {
		return nil, nil, errors.MakeStaticError("Object comprehensions cannot have hidden fields", next.loc)
	}
	if field.Kind != ast.ObjectFieldExpr {
		return nil, nil, errors.MakeStaticError("Object comprehensions can only have [e] fields", next.loc)
	}
	spec, last, err := p.parseComprehensionSpecs(next, tokenBraceR)
	if err != nil {
		return nil, nil, err
	}
	return &ast.ObjectComp{
		NodeBase:      ast.NewNodeBaseLoc(locFromTokens(tok, last), tok.fodder),
		Fields:        fields,
		TrailingComma: gotComma,
		Spec:          *spec,
		CloseFodder:   last.fodder,
	}, last, nil
}

func (p *parser) parseObjectRemainderField(literalFields *LiteralFieldSet, tok *token, next *token) (*ast.ObjectField, errors.StaticError) {
	var kind ast.ObjectFieldKind
	var fodder1 ast.Fodder
	var expr1 ast.Node
	var id *ast.Identifier
	var fodder2 ast.Fodder
	switch next.kind {
	case tokenIdentifier:
		kind = ast.ObjectFieldID
		id = (*ast.Identifier)(&next.data)
		fodder1 = next.fodder
	case tokenStringDouble, tokenStringSingle,
		tokenStringBlock, tokenVerbatimStringDouble, tokenVerbatimStringSingle:
		kind = ast.ObjectFieldStr
		expr1 = tokenStringToAst(next)
	default:
		fodder1 = next.fodder
		kind = ast.ObjectFieldExpr
		var err errors.StaticError
		expr1, err = p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		bracketR, err := p.popExpect(tokenBracketR)
		if err != nil {
			return nil, err
		}
		fodder2 = bracketR.fodder
	}

	isMethod := false
	methComma := false
	var parenL *token
	var parenR *token
	var params []ast.Parameter
	if p.peek().kind == tokenParenL {
		parenL = p.pop()
		var err errors.StaticError
		parenR, params, methComma, err = p.parseParameters("method parameter")
		if err != nil {
			return nil, err
		}
		isMethod = true
	}

	opFodder, plusSugar, hide, err := p.parseObjectAssignmentOp()
	if err != nil {
		return nil, err
	}

	if plusSugar && isMethod {
		return nil, errors.MakeStaticError(
			fmt.Sprintf("Cannot use +: syntax sugar in a method: %v", next.data), next.loc)
	}

	if kind != ast.ObjectFieldExpr {
		if !literalFields.Add(LiteralField(next.data)) {
			return nil, errors.MakeStaticError(
				fmt.Sprintf("Duplicate field: %v", next.data), next.loc)
		}
	}

	body, err := p.parse(maxPrecedence)
	if err != nil {
		return nil, err
	}

	var method *ast.Function
	if isMethod {
		method = &ast.Function{
			ParenLeftFodder:  parenL.fodder,
			Parameters:       params,
			TrailingComma:    methComma,
			ParenRightFodder: parenR.fodder,
			Body:             body,
		}
	}

	var commaFodder ast.Fodder
	if p.peek().kind == tokenComma {
		commaFodder = p.peek().fodder
	}

	return &ast.ObjectField{
		Kind:        kind,
		Hide:        hide,
		SuperSugar:  plusSugar,
		Method:      method,
		Fodder1:     fodder1,
		Expr1:       expr1,
		Id:          id,
		Fodder2:     fodder2,
		OpFodder:    opFodder,
		Expr2:       body,
		CommaFodder: commaFodder,
		LocRange:    locFromTokenAST(next, body),
	}, nil
}

func (p *parser) parseObjectRemainderLocal(binds *ast.IdentifierSet, tok *token, next *token) (*ast.ObjectField, errors.StaticError) {
	varID, popErr := p.popExpect(tokenIdentifier)
	if popErr != nil {
		return nil, popErr
	}

	id := ast.Identifier(varID.data)

	if binds.Contains(id) {
		return nil, errors.MakeStaticError(fmt.Sprintf("Duplicate local var: %v", id), varID.loc)
	}

	// TODO(sbarzowski) Can we reuse regular local bind parsing here?

	isMethod := false
	funcComma := false
	var parenL *token
	var parenR *token
	var params []ast.Parameter
	if p.peek().kind == tokenParenL {
		parenL = p.pop()
		isMethod = true
		var err errors.StaticError
		parenR, params, funcComma, err = p.parseParameters("function parameter")
		if err != nil {
			return nil, err
		}
	}
	opToken, popErr := p.popExpectOp("=")
	if popErr != nil {
		return nil, popErr
	}

	body, err := p.parse(maxPrecedence)
	if err != nil {
		return nil, err
	}

	var method *ast.Function
	if isMethod {
		method = &ast.Function{
			ParenLeftFodder:  parenL.fodder,
			Parameters:       params,
			ParenRightFodder: parenR.fodder,
			TrailingComma:    funcComma,
			Body:             body,
		}
	}

	binds.Add(id)

	var commaFodder ast.Fodder
	if p.peek().kind == tokenComma {
		commaFodder = p.peek().fodder
	}

	return &ast.ObjectField{
		Kind:        ast.ObjectLocal,
		Hide:        ast.ObjectFieldVisible,
		SuperSugar:  false,
		Method:      method,
		Fodder1:     next.fodder,
		Fodder2:     varID.fodder,
		Id:          &id,
		OpFodder:    opToken.fodder,
		Expr2:       body,
		CommaFodder: commaFodder,
		LocRange:    locFromTokenAST(varID, body),
	}, nil
}

func (p *parser) parseObjectRemainderAssert(tok *token, next *token) (*ast.ObjectField, errors.StaticError) {
	cond, err := p.parse(maxPrecedence)
	if err != nil {
		return nil, err
	}
	lastAST := cond // for determining location
	var msg ast.Node
	var colonFodder ast.Fodder
	if p.peek().kind == tokenOperator && p.peek().data == ":" {
		colonToken := p.pop()
		colonFodder = colonToken.fodder
		msg, err = p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		lastAST = msg
	}

	var commaFodder ast.Fodder
	if p.peek().kind == tokenComma {
		commaFodder = p.peek().fodder
	}

	return &ast.ObjectField{
		Kind:        ast.ObjectAssert,
		Hide:        ast.ObjectFieldVisible,
		Fodder1:     next.fodder,
		Expr2:       cond,
		OpFodder:    colonFodder,
		Expr3:       msg,
		CommaFodder: commaFodder,
		LocRange:    locFromTokenAST(next, lastAST),
	}, nil
}

// Parse object or object comprehension without leading brace
func (p *parser) parseObjectRemainder(tok *token) (ast.Node, *token, errors.StaticError) {
	var fields ast.ObjectFields
	literalFields := make(LiteralFieldSet)
	binds := make(ast.IdentifierSet)

	gotComma := false
	first := true

	next := p.pop()

	for {

		if next.kind == tokenBraceR {
			return &ast.Object{
				NodeBase:      ast.NewNodeBaseLoc(locFromTokens(tok, next), tok.fodder),
				Fields:        fields,
				TrailingComma: gotComma,
				CloseFodder:   next.fodder,
			}, next, nil
		}

		if next.kind == tokenFor {
			// It's a comprehension
			return p.parseObjectRemainderComp(fields, gotComma, tok, next)
		}

		if !gotComma && !first {
			return nil, nil, errors.MakeStaticError("Expected a comma before next field", next.loc)
		}

		var field *ast.ObjectField
		var err errors.StaticError
		switch next.kind {
		case tokenBracketL, tokenIdentifier, tokenStringDouble, tokenStringSingle,
			tokenStringBlock, tokenVerbatimStringDouble, tokenVerbatimStringSingle:
			field, err = p.parseObjectRemainderField(&literalFields, tok, next)
			if err != nil {
				return nil, nil, err
			}

		case tokenLocal:
			field, err = p.parseObjectRemainderLocal(&binds, tok, next)
			if err != nil {
				return nil, nil, err
			}

		case tokenAssert:
			field, err = p.parseObjectRemainderAssert(tok, next)
			if err != nil {
				return nil, nil, err
			}

		default:
			return nil, nil, makeUnexpectedError(next, "parsing field definition")
		}
		fields = append(fields, *field)

		next = p.pop()
		if next.kind == tokenComma {
			gotComma = true
			next = p.pop()
		} else {
			gotComma = false
		}

		first = false
	}
}

/* parses for x in expr for y in expr if expr for z in expr ... */
func (p *parser) parseComprehensionSpecs(forToken *token, end tokenKind) (*ast.ForSpec, *token, errors.StaticError) {
	var parseComprehensionSpecsHelper func(forToken *token, outer *ast.ForSpec) (*ast.ForSpec, *token, errors.StaticError)
	parseComprehensionSpecsHelper = func(forToken *token, outer *ast.ForSpec) (*ast.ForSpec, *token, errors.StaticError) {
		var ifSpecs []ast.IfSpec

		varID, popErr := p.popExpect(tokenIdentifier)
		if popErr != nil {
			return nil, nil, popErr
		}
		id := ast.Identifier(varID.data)
		inToken, err := p.popExpect(tokenIn)
		if err != nil {
			return nil, nil, err
		}
		arr, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, nil, err
		}
		forSpec := &ast.ForSpec{
			ForFodder: forToken.fodder,
			VarFodder: varID.fodder,
			VarName:   id,
			InFodder:  inToken.fodder,
			Expr:      arr,
			Outer:     outer,
		}

		maybeIf := p.pop()
		for ; maybeIf.kind == tokenIf; maybeIf = p.pop() {
			cond, err := p.parse(maxPrecedence)
			if err != nil {
				return nil, nil, err
			}
			ifSpecs = append(ifSpecs, ast.IfSpec{
				IfFodder: maybeIf.fodder,
				Expr:     cond,
			})
		}
		forSpec.Conditions = ifSpecs
		if maybeIf.kind == end {
			return forSpec, maybeIf, nil
		}

		if maybeIf.kind != tokenFor {
			return nil, nil, errors.MakeStaticError(
				fmt.Sprintf("Expected for, if or %v after for clause, got: %v", end, maybeIf), maybeIf.loc)
		}

		return parseComprehensionSpecsHelper(maybeIf, forSpec)
	}
	return parseComprehensionSpecsHelper(forToken, nil)
}

// Assumes that the leading '[' has already been consumed and passed as tok.
// Should read up to and consume the trailing ']'
func (p *parser) parseArray(tok *token) (ast.Node, errors.StaticError) {
	if p.peek().kind == tokenBracketR {
		bracketR := p.pop()
		return &ast.Array{
			NodeBase:    ast.NewNodeBaseLoc(locFromTokens(tok, bracketR), tok.fodder),
			CloseFodder: bracketR.fodder,
		}, nil
	}

	first, err := p.parse(maxPrecedence)
	if err != nil {
		return nil, err
	}
	var gotComma bool
	var commaFodder ast.Fodder
	if p.peek().kind == tokenComma {
		comma := p.pop()
		gotComma = true
		commaFodder = comma.fodder
	}

	if p.peek().kind == tokenFor {
		// It's a comprehension
		forToken := p.pop()
		spec, last, err := p.parseComprehensionSpecs(forToken, tokenBracketR)
		if err != nil {
			return nil, err
		}
		return &ast.ArrayComp{
			NodeBase:            ast.NewNodeBaseLoc(locFromTokens(tok, last), tok.fodder),
			Body:                first,
			TrailingComma:       gotComma,
			TrailingCommaFodder: commaFodder,
			Spec:                *spec,
			CloseFodder:         last.fodder,
		}, nil
	}

	// Not a comprehension: It can have more elements.
	elements := []ast.CommaSeparatedExpr{{
		Expr:        first,
		CommaFodder: commaFodder,
	}}

	var bracketR *token
	for {
		next := p.peek()

		if next.kind == tokenBracketR {
			bracketR = p.pop()
			break
		}
		if !gotComma {
			return nil, errors.MakeStaticError("Expected a comma before next array element", next.loc)
		}
		nextElem, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}

		element := ast.CommaSeparatedExpr{
			Expr: nextElem,
		}
		if p.peek().kind == tokenComma {
			comma := p.pop()
			gotComma = true
			element.CommaFodder = comma.fodder
		} else {
			gotComma = false
		}
		elements = append(elements, element)
	}

	return &ast.Array{
		NodeBase: ast.NewNodeBaseLoc(locFromTokens(tok, bracketR),
			tok.fodder),
		Elements:      elements,
		TrailingComma: gotComma,
		CloseFodder:   bracketR.fodder,
	}, nil
}

func tokenStringToAst(tok *token) *ast.LiteralString {
	switch tok.kind {
	case tokenStringSingle:
		return &ast.LiteralString{
			NodeBase: ast.NewNodeBaseLoc(tok.loc, tok.fodder),
			Value:    tok.data,
			Kind:     ast.StringSingle,
		}
	case tokenStringDouble:
		return &ast.LiteralString{
			NodeBase: ast.NewNodeBaseLoc(tok.loc, tok.fodder),
			Value:    tok.data,
			Kind:     ast.StringDouble,
		}
	case tokenStringBlock:
		return &ast.LiteralString{
			NodeBase:        ast.NewNodeBaseLoc(tok.loc, tok.fodder),
			Value:           tok.data,
			Kind:            ast.StringBlock,
			BlockIndent:     tok.stringBlockIndent,
			BlockTermIndent: tok.stringBlockTermIndent,
		}
	case tokenVerbatimStringDouble:
		return &ast.LiteralString{
			NodeBase: ast.NewNodeBaseLoc(tok.loc, tok.fodder),
			Value:    tok.data,
			Kind:     ast.VerbatimStringDouble,
		}
	case tokenVerbatimStringSingle:
		return &ast.LiteralString{
			NodeBase: ast.NewNodeBaseLoc(tok.loc, tok.fodder),
			Value:    tok.data,
			Kind:     ast.VerbatimStringSingle,
		}
	default:
		panic(fmt.Sprintf("Not a string token %#+v", tok))
	}
}

func (p *parser) parseTerminal() (ast.Node, errors.StaticError) {
	tok := p.pop()
	switch tok.kind {
	case tokenAssert, tokenBraceR, tokenBracketR, tokenComma, tokenDot, tokenElse,
		tokenError, tokenFor, tokenFunction, tokenIf, tokenIn, tokenImport, tokenImportStr,
		tokenLocal, tokenOperator, tokenParenR, tokenSemicolon, tokenTailStrict, tokenThen:
		return nil, makeUnexpectedError(tok, "parsing terminal")

	case tokenEndOfFile:
		return nil, errors.MakeStaticError("Unexpected end of file", tok.loc)

	case tokenBraceL:
		obj, _, err := p.parseObjectRemainder(tok)
		return obj, err

	case tokenBracketL:
		return p.parseArray(tok)

	case tokenParenL:
		inner, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		tokRight, err := p.popExpect(tokenParenR)
		if err != nil {
			return nil, err
		}
		return &ast.Parens{
			NodeBase:    ast.NewNodeBaseLoc(locFromTokens(tok, tokRight), tok.fodder),
			Inner:       inner,
			CloseFodder: tokRight.fodder,
		}, nil

	// Literals
	case tokenNumber:
		return &ast.LiteralNumber{
			NodeBase:       ast.NewNodeBaseLoc(tok.loc, tok.fodder),
			OriginalString: tok.data,
		}, nil
	case tokenStringDouble, tokenStringSingle,
		tokenStringBlock, tokenVerbatimStringDouble, tokenVerbatimStringSingle:
		return tokenStringToAst(tok), nil
	case tokenFalse:
		return &ast.LiteralBoolean{
			NodeBase: ast.NewNodeBaseLoc(tok.loc, tok.fodder),
			Value:    false,
		}, nil
	case tokenTrue:
		return &ast.LiteralBoolean{
			NodeBase: ast.NewNodeBaseLoc(tok.loc, tok.fodder),
			Value:    true,
		}, nil
	case tokenNullLit:
		return &ast.LiteralNull{
			NodeBase: ast.NewNodeBaseLoc(tok.loc, tok.fodder),
		}, nil

	// Variables
	case tokenDollar:
		return &ast.Dollar{
			NodeBase: ast.NewNodeBaseLoc(tok.loc, tok.fodder),
		}, nil
	case tokenIdentifier:
		return &ast.Var{
			NodeBase: ast.NewNodeBaseLoc(tok.loc, tok.fodder),
			Id:       ast.Identifier(tok.data),
		}, nil
	case tokenSelf:
		return &ast.Self{
			NodeBase: ast.NewNodeBaseLoc(tok.loc, tok.fodder),
		}, nil
	case tokenSuper:
		next := p.pop()
		var index ast.Node
		var id *ast.Identifier
		var idFodder ast.Fodder
		switch next.kind {
		case tokenDot:
			fieldID, err := p.popExpect(tokenIdentifier)
			if err != nil {
				return nil, err
			}
			idFodder = fieldID.fodder
			id = (*ast.Identifier)(&fieldID.data)
		case tokenBracketL:
			var err errors.StaticError
			index, err = p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
			bracketR, err := p.popExpect(tokenBracketR)
			if err != nil {
				return nil, err
			}
			idFodder = bracketR.fodder
		default:
			return nil, errors.MakeStaticError("Expected . or [ after super", tok.loc)
		}
		return &ast.SuperIndex{
			NodeBase:  ast.NewNodeBaseLoc(tok.loc, tok.fodder),
			DotFodder: next.fodder,
			Index:     index,
			IDFodder:  idFodder,
			Id:        id,
		}, nil
	}

	return nil, errors.MakeStaticError(fmt.Sprintf("INTERNAL ERROR: Unknown tok kind: %v", tok.kind), tok.loc)
}

func (p *parser) parsingFailure(msg string, tok *token) (ast.Node, errors.StaticError) {
	return nil, errors.MakeStaticError(msg, tok.loc)
}

func (p *parser) parse(prec precedence) (ast.Node, errors.StaticError) {
	begin := p.peek()

	switch begin.kind {
	// These cases have effectively maxPrecedence as the first
	// call to parse will parse them.
	case tokenAssert:
		p.pop()
		cond, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		var msg ast.Node
		var colonFodder ast.Fodder
		if p.peek().kind == tokenOperator && p.peek().data == ":" {
			colon := p.pop()
			colonFodder = colon.fodder
			msg, err = p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
		}
		semicolon, err := p.popExpect(tokenSemicolon)
		if err != nil {
			return nil, err
		}
		rest, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		return &ast.Assert{
			NodeBase:        ast.NewNodeBaseLoc(locFromTokenAST(begin, rest), begin.fodder),
			Cond:            cond,
			ColonFodder:     colonFodder,
			Message:         msg,
			SemicolonFodder: semicolon.fodder,
			Rest:            rest,
		}, nil

	case tokenError:
		p.pop()
		expr, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		return &ast.Error{
			NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(begin, expr), begin.fodder),
			Expr:     expr,
		}, nil

	case tokenIf:
		p.pop()
		cond, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		thenToken, err := p.popExpect(tokenThen)
		if err != nil {
			return nil, err
		}
		branchTrue, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		var branchFalse ast.Node
		var elseFodder ast.Fodder
		lr := locFromTokenAST(begin, branchTrue)
		if p.peek().kind == tokenElse {
			elseToken := p.pop()
			elseFodder = elseToken.fodder
			branchFalse, err = p.parse(maxPrecedence)
			if err != nil {
				return nil, err
			}
			lr = locFromTokenAST(begin, branchFalse)
		}
		return &ast.Conditional{
			NodeBase:    ast.NewNodeBaseLoc(lr, begin.fodder),
			Cond:        cond,
			ThenFodder:  thenToken.fodder,
			BranchTrue:  branchTrue,
			ElseFodder:  elseFodder,
			BranchFalse: branchFalse,
		}, nil

	case tokenFunction:
		p.pop()
		next := p.pop()
		if next.kind != tokenParenL {
			return nil, errors.MakeStaticError(fmt.Sprintf("Expected ( but got %v", next), next.loc)
		}
		parenR, params, gotComma, err := p.parseParameters("function parameter")
		if err != nil {
			return nil, err
		}
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		return &ast.Function{
			NodeBase:         ast.NewNodeBaseLoc(locFromTokenAST(begin, body), begin.fodder),
			ParenLeftFodder:  next.fodder,
			Parameters:       params,
			TrailingComma:    gotComma,
			ParenRightFodder: parenR.fodder,
			Body:             body,
		}, nil

	case tokenImport:
		p.pop()
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		if lit, ok := body.(*ast.LiteralString); ok {
			if lit.Kind == ast.StringBlock {
				return nil, errors.MakeStaticError("Block string literals not allowed in imports", *body.Loc())
			}
			return &ast.Import{
				NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(begin, body), begin.fodder),
				File:     lit,
			}, nil
		}
		return nil, errors.MakeStaticError("Computed imports are not allowed", *body.Loc())

	case tokenImportStr:
		p.pop()
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		if lit, ok := body.(*ast.LiteralString); ok {
			if lit.Kind == ast.StringBlock {
				return nil, errors.MakeStaticError("Block string literals not allowed in imports", *body.Loc())
			}
			return &ast.ImportStr{
				NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(begin, body), begin.fodder),
				File:     lit,
			}, nil
		}
		return nil, errors.MakeStaticError("Computed imports are not allowed", *body.Loc())

	case tokenLocal:
		p.pop()
		var binds ast.LocalBinds
		for {
			delim, err := p.parseBind(&binds)
			if err != nil {
				return nil, err
			}
			if delim.kind == tokenSemicolon {
				break
			}
		}
		body, err := p.parse(maxPrecedence)
		if err != nil {
			return nil, err
		}
		return &ast.Local{
			NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(begin, body), begin.fodder),
			Binds:    binds,
			Body:     body,
		}, nil

	default:
		// ast.Unary operator
		if begin.kind == tokenOperator {
			uop, ok := ast.UopMap[begin.data]
			if !ok {
				return nil, errors.MakeStaticError(fmt.Sprintf("Not a unary operator: %v", begin.data), begin.loc)
			}
			if prec == unaryPrecedence {
				op := p.pop()
				expr, err := p.parse(prec)
				if err != nil {
					return nil, err
				}
				return &ast.Unary{
					NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(op, expr), begin.fodder),
					Op:       uop,
					Expr:     expr,
				}, nil
			}
		}

		// Base case
		if prec == 0 {
			return p.parseTerminal()
		}

		lhs, err := p.parse(prec - 1)
		if err != nil {
			return nil, err
		}

		for {
			// Then next token must be a binary operator.

			var bop ast.BinaryOp

			// Check precedence is correct for this level.  If we're parsing operators
			// with higher precedence, then return lhs and let lower levels deal with
			// the operator.
			switch p.peek().kind {
			case tokenIn:
				bop = ast.BopIn
				if bopPrecedence[bop] != prec {
					return lhs, nil
				}
			case tokenOperator:
				_ = "breakpoint"
				if p.peek().data == ":" {
					// Special case for the colons in assert. Since COLON is no-longer a
					// special token, we have to make sure it does not trip the
					// op_is_binary test below.  It should terminate parsing of the
					// expression here, returning control to the parsing of the actual
					// assert AST.
					return lhs, nil
				}
				if p.peek().data == "::" {
					// Special case for [e::]
					// We need to stop parsing e when we see the :: and
					// avoid tripping the op_is_binary test below.
					return lhs, nil
				}
				var ok bool
				bop, ok = ast.BopMap[p.peek().data]
				if !ok {
					return nil, errors.MakeStaticError(fmt.Sprintf("Not a binary operator: %v", p.peek().data), p.peek().loc)
				}
				if bopPrecedence[bop] != prec {
					return lhs, nil
				}

			case tokenDot, tokenBracketL, tokenParenL, tokenBraceL:
				if applyPrecedence != prec {
					return lhs, nil
				}
			default:
				return lhs, nil
			}

			op := p.pop()
			switch op.kind {
			case tokenBracketL:
				// handle slice
				var indexes [3]ast.Node
				var fodders [3]ast.Fodder
				colonsConsumed := 0

				var end *token
				readyForNextIndex := true
				var rightBracketFodder ast.Fodder
				for colonsConsumed < 3 {
					if p.peek().kind == tokenBracketR {
						end = p.pop()
						rightBracketFodder = end.fodder
						break
					} else if p.peek().data == ":" {
						end = p.pop()
						fodders[colonsConsumed] = end.fodder
						colonsConsumed++
						readyForNextIndex = true
					} else if p.peek().data == "::" {
						end = p.pop()
						fodders[colonsConsumed] = end.fodder
						colonsConsumed += 2
						readyForNextIndex = true
					} else if readyForNextIndex {
						indexes[colonsConsumed], err = p.parse(maxPrecedence)
						if err != nil {
							return nil, err
						}
						readyForNextIndex = false
					} else {
						return nil, p.unexpectedTokenError(tokenBracketR, p.peek())
					}
				}
				if colonsConsumed > 2 {
					// example: target[42:42:42:42]
					return p.parsingFailure("Invalid slice: too many colons", end)
				}
				if colonsConsumed == 0 && readyForNextIndex {
					// example: target[]
					return p.parsingFailure("ast.Index requires an expression", end)
				}
				isSlice := colonsConsumed > 0

				if isSlice {
					lhs = &ast.Slice{
						NodeBase:           ast.NewNodeBaseLoc(locFromTokens(begin, end), ast.Fodder{}),
						Target:             lhs,
						LeftBracketFodder:  op.fodder,
						BeginIndex:         indexes[0],
						EndColonFodder:     fodders[0],
						EndIndex:           indexes[1],
						StepColonFodder:    fodders[1],
						Step:               indexes[2],
						RightBracketFodder: rightBracketFodder,
					}
				} else {
					lhs = &ast.Index{
						NodeBase:           ast.NewNodeBaseLoc(locFromTokens(begin, end), ast.Fodder{}),
						Target:             lhs,
						LeftBracketFodder:  op.fodder,
						Index:              indexes[0],
						RightBracketFodder: rightBracketFodder,
					}
				}
			case tokenDot:
				fieldID, err := p.popExpect(tokenIdentifier)
				if err != nil {
					return nil, err
				}
				id := ast.Identifier(fieldID.data)
				lhs = &ast.Index{
					NodeBase:           ast.NewNodeBaseLoc(locFromTokens(begin, fieldID), ast.Fodder{}),
					Target:             lhs,
					LeftBracketFodder:  op.fodder,
					Id:                 &id,
					RightBracketFodder: fieldID.fodder,
				}
			case tokenParenL:
				end, args, gotComma, err := p.parseArguments("function argument")
				if err != nil {
					return nil, err
				}
				tailStrict := false
				var tailStrictFodder ast.Fodder
				if p.peek().kind == tokenTailStrict {
					tailStrictTok := p.pop()
					tailStrictFodder = tailStrictTok.fodder
					tailStrict = true
				}
				lhs = &ast.Apply{
					NodeBase:         ast.NewNodeBaseLoc(locFromTokens(begin, end), ast.Fodder{}),
					Target:           lhs,
					FodderLeft:       op.fodder,
					Arguments:        *args,
					TrailingComma:    gotComma,
					FodderRight:      end.fodder,
					TailStrict:       tailStrict,
					TailStrictFodder: tailStrictFodder,
				}
			case tokenBraceL:
				obj, end, err := p.parseObjectRemainder(op)
				if err != nil {
					return nil, err
				}
				lhs = &ast.ApplyBrace{
					NodeBase: ast.NewNodeBaseLoc(locFromTokens(begin, end), ast.Fodder{}),
					Left:     lhs,
					Right:    obj,
				}
			default:
				if op.kind == tokenIn && p.peek().kind == tokenSuper {
					super := p.pop()
					lhs = &ast.InSuper{
						NodeBase:    ast.NewNodeBaseLoc(locFromTokens(begin, super), ast.Fodder{}),
						Index:       lhs,
						InFodder:    op.fodder,
						SuperFodder: super.fodder,
					}
				} else {
					rhs, err := p.parse(prec - 1)
					if err != nil {
						return nil, err
					}
					lhs = &ast.Binary{
						NodeBase: ast.NewNodeBaseLoc(locFromTokenAST(begin, rhs), ast.Fodder{}),
						Left:     lhs,
						OpFodder: op.fodder,
						Op:       bop,
						Right:    rhs,
					}
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------

// Parse parses a slice of tokens into a parse tree.  Any fodder after the final token is
// returned as well.
func Parse(t Tokens) (ast.Node, ast.Fodder, errors.StaticError) {
	p := makeParser(t)
	expr, err := p.parse(maxPrecedence)
	if err != nil {
		return nil, nil, err
	}
	eof := p.peek()

	if eof.kind != tokenEndOfFile {
		return nil, nil, errors.MakeStaticError(fmt.Sprintf("Did not expect: %v", eof), eof.loc)
	}

	addContext(expr, &topLevelContext, anonymous)

	return expr, eof.fodder, nil
}

// SnippetToRawAST converts a Jsonnet code snippet to an AST (without any transformations).
// Any fodder after the final token is returned as well.
func SnippetToRawAST(diagnosticFilename ast.DiagnosticFileName, importedFilename, snippet string) (ast.Node, ast.Fodder, error) {
	tokens, err := Lex(diagnosticFilename, importedFilename, snippet)
	if err != nil {
		return nil, nil, err
	}
	return Parse(tokens)
}
