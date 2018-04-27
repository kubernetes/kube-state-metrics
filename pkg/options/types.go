/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package options

import (
	"sort"
	"strings"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CollectorSet map[string]struct{}

func (c *CollectorSet) String() string {
	s := *c
	ss := s.asSlice()
	sort.Strings(ss)
	return strings.Join(ss, ",")
}

func (c *CollectorSet) Set(value string) error {
	s := *c
	cols := strings.Split(value, ",")
	for _, col := range cols {
		col = strings.TrimSpace(col)
		if len(col) != 0 {
			_, ok := AvailableCollectors[col]
			if !ok {
				glog.Fatalf("Collector \"%s\" does not exist", col)
			}
			s[col] = struct{}{}
		}
	}
	return nil
}

func (c CollectorSet) asSlice() []string {
	cols := []string{}
	for col := range c {
		cols = append(cols, col)
	}
	return cols
}

func (c CollectorSet) isEmpty() bool {
	return len(c.asSlice()) == 0
}

func (c *CollectorSet) Type() string {
	return "string"
}

type NamespaceList []string

func (n *NamespaceList) String() string {
	return strings.Join(*n, ",")
}

func (n *NamespaceList) IsAllNamespaces() bool {
	return len(*n) == 1 && (*n)[0] == metav1.NamespaceAll
}

func (n *NamespaceList) Set(value string) error {
	splittedNamespaces := strings.Split(value, ",")
	for _, ns := range splittedNamespaces {
		ns = strings.TrimSpace(ns)
		if len(ns) != 0 {
			*n = append(*n, ns)
		}
	}
	return nil
}

func (n *NamespaceList) Type() string {
	return "string"
}
