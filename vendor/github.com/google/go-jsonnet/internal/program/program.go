package program

import (
	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/parser"
)

// SnippetToAST converts a Jsonnet code snippet to a desugared and analyzed AST.
func SnippetToAST(diagnosticFilename ast.DiagnosticFileName, importedFilename, snippet string) (ast.Node, error) {
	node, _, err := parser.SnippetToRawAST(diagnosticFilename, importedFilename, snippet)
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
