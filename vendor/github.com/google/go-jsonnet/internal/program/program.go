package program

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/parser"
)

// SnippetToAST converts a Jsonnet code snippet to a desugared and analyzed AST.
func SnippetToAST(filename string, snippet string) (ast.Node, error) {
	node, err := parser.SnippetToRawAST(filename, snippet)
	if err != nil {
		return nil, err
	}
	err = desugarAST(&node)
	if err != nil {
		return nil, err
	}
	err = analyze(node)
	if err != nil {
		return nil, err
	}
	return node, nil
}
