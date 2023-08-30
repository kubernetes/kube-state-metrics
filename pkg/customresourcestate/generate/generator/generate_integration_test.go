/*
Copyright 2023 The Kubernetes Authors All rights reserved.

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
package generator

import (
	"bytes"
	"io"
	"os"
	"path"
	"testing"

	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

func Test_Generate(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	optionsRegistry := &markers.Registry{}

	metricGenerator := CustomResourceConfigGenerator{}
	if err := metricGenerator.RegisterMarkers(optionsRegistry); err != nil {
		t.Error(err)
	}

	out := &outputRule{
		buf: &bytes.Buffer{},
	}

	// Load the passed packages as roots.
	roots, err := loader.LoadRoots(path.Join(cwd, "testdata", "..."))
	if err != nil {
		t.Errorf("loading packages %v", err)
	}

	gen := CustomResourceConfigGenerator{}

	generationContext := &genall.GenerationContext{
		Collector:  &markers.Collector{Registry: optionsRegistry},
		Roots:      roots,
		Checker:    &loader.TypeChecker{},
		OutputRule: out,
	}

	t.Log("Trying to generate a custom resource configuration from the loaded packages")

	if err := gen.Generate(generationContext); err != nil {
		t.Error(err)
	}
	output := out.buf.String()

	t.Log("Comparing output to testdata to check for regressions")

	expectedFile, err := os.ReadFile(path.Clean(path.Join(cwd, "testdata", "foo-config.yaml")))
	if err != nil {
		t.Error(err)
	}

	if string(expectedFile) != output {
		t.Log("output:")
		t.Log(output)
		t.Error("Expected output to match file testdata/foo-config.yaml")
	}
}

type outputRule struct {
	buf *bytes.Buffer
}

func (o *outputRule) Open(_ *loader.Package, _ string) (io.WriteCloser, error) {
	return nopCloser{o.buf}, nil
}

type nopCloser struct {
	io.Writer
}

func (n nopCloser) Close() error {
	return nil
}
