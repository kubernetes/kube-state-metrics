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
package markers

import (
	"fmt"

	"k8s.io/client-go/util/jsonpath"
	"k8s.io/klog/v2"
	ctrlmarkers "sigs.k8s.io/controller-tools/pkg/markers"

	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
)

type markerDefinitionWithHelp struct {
	*ctrlmarkers.Definition
	Help *ctrlmarkers.DefinitionHelp
}

func must(def *ctrlmarkers.Definition, err error) *markerDefinitionWithHelp {
	return &markerDefinitionWithHelp{
		Definition: ctrlmarkers.Must(def, err),
	}
}

func (d *markerDefinitionWithHelp) help(help *ctrlmarkers.DefinitionHelp) *markerDefinitionWithHelp {
	d.Help = help
	return d
}

func (d *markerDefinitionWithHelp) Register(reg *ctrlmarkers.Registry) error {
	if err := reg.Register(d.Definition); err != nil {
		return err
	}
	if d.Help != nil {
		reg.AddHelp(d.Definition, d.Help)
	}
	return nil
}

// jsonPath is a simple JSON path, i.e. without array notation.
type jsonPath string

// Parse is implemented to overwrite how json.Marshal and json.Unmarshal handles
// this type and parses the string to a string array instead. It is inspired by
// `kubectl explain` parsing the json path parameter.
// xref: https://github.com/kubernetes/kubectl/blob/release-1.28/pkg/explain/explain.go#L35
func (j jsonPath) Parse() ([]string, error) {
	ret := []string{}

	jpp, err := jsonpath.Parse("JSONPath", `{`+string(j)+`}`)
	if err != nil {
		return nil, fmt.Errorf("parse JSONPath: %w", err)
	}

	// Because of the way the jsonpath library works, the schema of the parser is [][]NodeList
	// meaning we need to get the outer node list, make sure it's only length 1, then get the inner node
	// list, and only then can we look at the individual nodes themselves.
	outerNodeList := jpp.Root.Nodes
	if len(outerNodeList) > 1 {
		return nil, fmt.Errorf("must pass in 1 jsonpath string, got %d", len(outerNodeList))
	}

	list, ok := outerNodeList[0].(*jsonpath.ListNode)
	if !ok {
		return nil, fmt.Errorf("unable to typecast to jsonpath.ListNode")
	}
	for _, n := range list.Nodes {
		nf, ok := n.(*jsonpath.FieldNode)
		if !ok {
			return nil, fmt.Errorf("unable to typecast to jsonpath.NodeField")
		}
		ret = append(ret, nf.Value)
	}

	return ret, nil
}

func newMetricMeta(basePath []string, j jsonPath, jsonLabelsFromPath map[string]jsonPath) customresourcestate.MetricMeta {
	path := basePath
	if j != "" {
		valueFrom, err := j.Parse()
		if err != nil {
			klog.Fatal(err)
		}
		if len(valueFrom) > 0 {
			path = append(path, valueFrom...)
		}
	}

	labelsFromPath := map[string][]string{}
	for k, v := range jsonLabelsFromPath {
		path := []string{}
		var err error
		if v != "." {
			path, err = v.Parse()
			if err != nil {
				klog.Fatal(err)
			}
		}
		labelsFromPath[k] = path
	}

	return customresourcestate.MetricMeta{
		Path:           path,
		LabelsFromPath: labelsFromPath,
	}
}
