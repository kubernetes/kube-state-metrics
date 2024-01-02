/*
Copyright 2019 The Kubernetes Authors.

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

package main

import (
	"go/ast"
)

func findStableMetricDeclaration(tree ast.Node, metricsImportName string) ([]*ast.CallExpr, []error) {
	v := stableMetricFinder{
		metricsImportName:          metricsImportName,
		stableMetricsFunctionCalls: []*ast.CallExpr{},
		errors:                     []error{},
	}
	ast.Walk(&v, tree)
	return v.stableMetricsFunctionCalls, v.errors
}

// Implements visitor pattern for ast.Node that collects all stable metric expressions
type stableMetricFinder struct {
	metricsImportName          string
	currentFunctionCall        *ast.CallExpr
	stableMetricsFunctionCalls []*ast.CallExpr
	errors                     []error
}

var _ ast.Visitor = (*stableMetricFinder)(nil)

func (f *stableMetricFinder) Visit(node ast.Node) (w ast.Visitor) {
	switch opts := node.(type) {
	case *ast.CallExpr:
		f.currentFunctionCall = opts
		if se, ok := opts.Fun.(*ast.SelectorExpr); ok {
			if se.Sel.Name == "NewFamilyGeneratorWithStabilityV2" {
				sl, _ := decodeStabilityLevel(opts.Args[3], f.metricsImportName)
				if sl != nil && string(*sl) == "STABLE" {
					f.stableMetricsFunctionCalls = append(f.stableMetricsFunctionCalls, opts)
					f.currentFunctionCall = nil
				}
			}
		}
	default:
		if f.currentFunctionCall == nil || node == nil || node.Pos() < f.currentFunctionCall.Rparen {
			return f
		}
		f.currentFunctionCall = nil
	}
	return f
}
