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
	"go/token"
	"strings"

	"k8s.io/component-base/metrics"
)

func decodeMetricCalls(fs []*ast.CallExpr, metricsImportName string, variables map[string]ast.Expr) ([]metric, []error) {
	finder := metricDecoder{
		kubeMetricsImportName: metricsImportName,
		variables:             variables,
	}
	ms := make([]metric, 0, len(fs))
	errors := []error{}
	for _, f := range fs {
		m, err := finder.decodeNewMetricCall(f)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if m != nil {
			ms = append(ms, *m)
		}
	}
	return ms, errors
}

type metricDecoder struct {
	kubeMetricsImportName string
	variables             map[string]ast.Expr
}

func (c *metricDecoder) decodeNewMetricCall(fc *ast.CallExpr) (*metric, error) {
	var m metric
	var err error

	_, ok := fc.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, newDecodeErrorf(fc, errNotDirectCall)
	}
	m, err = c.decodeDesc(fc)
	return &m, err
}

func (c *metricDecoder) decodeDesc(ce *ast.CallExpr) (metric, error) {
	m := &metric{}

	name, err := c.decodeString(ce.Args[0])
	if err != nil {
		return *m, newDecodeErrorf(ce, errorDecodingString)
	}
	m.Name = *name

	help, err := c.decodeString(ce.Args[1])
	if err != nil {
		return *m, newDecodeErrorf(ce, errorDecodingString)
	}
	m.Help = *help

	metricType, err := decodeStabilityLevel(ce.Args[2], "metric")
	if err != nil {
		return *m, newDecodeErrorf(ce, errorDecodingString)
	}
	m.Type = string(*metricType)

	sl, err := decodeStabilityLevel(ce.Args[3], "basemetrics")
	if err != nil {
		return *m, newDecodeErrorf(ce, "can't decode stability level")
	}

	if sl != nil {
		m.StabilityLevel = string(*sl)
	}

	labels, err := c.decodeLabels(ce.Args[5])
	if err != nil {
		return *m, newDecodeErrorf(ce, errorDecodingLabels)
	}
	m.Labels = labels

	return *m, nil
}

func (c *metricDecoder) decodeString(expr ast.Expr) (*string, error) {

	switch e := expr.(type) {
	case *ast.BasicLit:
		value, err := stringValue(e)
		if err != nil {
			return nil, err
		}
		return &value, nil
	}
	return nil, newDecodeErrorf(expr, errorDecodingString)
}

func (c *metricDecoder) decodeLabelsFromArray(exprs []ast.Expr) ([]string, error) {
	retval := []string{}
	for _, e := range exprs {
		v, err := c.decodeString(e)
		if err != nil || v == nil {
			return nil, newDecodeErrorf(e, errNonStringAttribute)
		}
		retval = append(retval, *v)
	}

	return retval, nil
}

func (c *metricDecoder) decodeLabels(expr ast.Expr) ([]string, error) {
	cl, ok := expr.(*ast.CompositeLit)
	if !ok {
		switch e := expr.(type) {
		case *ast.Ident:
			if e.Name == "nil" {
				return []string{}, nil
			}
			variableExpr, found := c.variables[e.Name]
			if !found {
				return nil, newDecodeErrorf(expr, errorFindingVariableForLabels)
			}
			cl2, ok := variableExpr.(*ast.CompositeLit)
			if !ok {
				return nil, newDecodeErrorf(expr, errorFindingVariableForLabels)
			}
			cl = cl2
		}
	}
	return c.decodeLabelsFromArray(cl.Elts)
}

func stringValue(bl *ast.BasicLit) (string, error) {
	if bl.Kind != token.STRING {
		return "", newDecodeErrorf(bl, errNonStringAttribute)
	}
	return strings.Trim(bl.Value, `"`), nil
}

func decodeStabilityLevel(expr ast.Expr, metricsFrameworkImportName string) (*metrics.StabilityLevel, error) {
	se, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return nil, newDecodeErrorf(expr, errStabilityLevel)
	}
	s, ok := se.X.(*ast.Ident)
	if !ok {
		return nil, newDecodeErrorf(expr, errStabilityLevel)
	}
	if s.String() != metricsFrameworkImportName {
		return nil, newDecodeErrorf(expr, errStabilityLevel)
	}

	stability := metrics.StabilityLevel(se.Sel.Name)
	return &stability, nil
}
