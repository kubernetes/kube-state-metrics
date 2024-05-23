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

package allowdenylist

import (
	"regexp"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("fails with two non empty maps", func(t *testing.T) {
		_, err := New(map[string]struct{}{"not-empty": {}}, map[string]struct{}{"not-empty": {}})
		if err == nil {
			t.Fatal("expected New() to fail with two non-empty maps")
		}
	})

	t.Run("defaults to denylisting", func(t *testing.T) {
		l, err := New(map[string]struct{}{}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		if l.isAllowList {
			t.Fatal("expected allowDenyList to default to denylist")
		}
	})

	t.Run("if allowlist set, should be allowlist", func(t *testing.T) {
		list, err := New(map[string]struct{}{"not-empty": {}}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		if !list.isAllowList {
			t.Fatal("expected list to be allowlist")
		}
	})

	t.Run("if denylist set, should be denylist", func(t *testing.T) {
		list, err := New(map[string]struct{}{}, map[string]struct{}{"not-empty": {}})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		if list.isAllowList {
			t.Fatal("expected list to be denylist")
		}
	})
}

func TestInclude(t *testing.T) {
	t.Run("adds when allowlist", func(t *testing.T) {
		allowlist, err := New(map[string]struct{}{"not-empty": {}}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		allowlist.Include([]string{"item1"})
		err = allowlist.Parse()
		if err != nil {
			t.Fatal("expected Parse() to not fail")
		}

		if !allowlist.IsIncluded("item1") {
			t.Fatal("expected included item to be included")
		}
	})
	t.Run("removes when denylist", func(t *testing.T) {
		item1 := "item1"
		denylist, err := New(map[string]struct{}{}, map[string]struct{}{item1: {}})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		denylist.Include([]string{item1})
		err = denylist.Parse()
		if err != nil {
			t.Fatalf("expected Parse() to not fail, but got error : %v", err)
		}

		if !denylist.IsIncluded(item1) {
			t.Fatal("expected included item to be included")
		}
	})
	t.Run("adds during pattern match when in allowlist mode", func(t *testing.T) {
		allowlist, err := New(map[string]struct{}{"not-empty": {}}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		allowlist.Include([]string{"kube_.*_info"})
		err = allowlist.Parse()
		if err != nil {
			t.Fatalf("expected Parse() to not fail, but got error : %v", err)
		}

		if !allowlist.IsIncluded("kube_secret_info") {
			t.Fatal("expected included item to be included")
		}
	})
	t.Run("removes during pattern match when in denyist mode", func(t *testing.T) {
		item1 := "kube_pod_container_resource_requests_cpu_cores"
		item2 := "kube_pod_container_resource_requests_memory_bytes"
		item3 := "kube_node_status_capacity_cpu_cores"
		item4 := "kube_node_status_capacity_memory_bytes"

		denylist, err := New(map[string]struct{}{}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		denylist.Exclude([]string{"kube_node_.*_cores", "kube_pod_.*_bytes"})
		err = denylist.Parse()
		if err != nil {
			t.Fatalf("expected Parse() to not fail, but got error : %v", err)
		}

		if denylist.IsExcluded(item1) {
			t.Fatalf("expected included %s to be included", item1)
		}
		if denylist.IsIncluded(item2) {
			t.Fatalf("expected included %s to be excluded", item2)
		}
		if denylist.IsIncluded(item3) {
			t.Fatalf("expected included %s to be excluded", item3)
		}
		if denylist.IsExcluded(item4) {
			t.Fatalf("expected included %s to be included", item4)
		}
	})
}

func TestExclude(t *testing.T) {
	t.Run("removes when allowlist", func(t *testing.T) {
		item1 := "item1"
		allowlist, err := New(map[string]struct{}{item1: {}}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		allowlist.Exclude([]string{item1})
		err = allowlist.Parse()
		if err != nil {
			t.Fatalf("expected Parse() to not fail, but got error : %v", err)
		}

		if allowlist.IsIncluded(item1) {
			t.Fatal("expected excluded item to be excluded")
		}
	})
	t.Run("removes when denylist", func(t *testing.T) {
		item1 := "item1"
		denylist, err := New(map[string]struct{}{}, map[string]struct{}{"not-empty": {}})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		denylist.Exclude([]string{item1})
		err = denylist.Parse()
		if err != nil {
			t.Fatalf("expected Parse() to not fail, but got error : %v", err)
		}

		if denylist.IsIncluded(item1) {
			t.Fatal("expected excluded item to be excluded")
		}
	})
}

func TestParse(t *testing.T) {
	t.Run("fails when an unparseable regex is passed", func(t *testing.T) {
		invalidItem := "*_pod_info"
		wb, err := New(map[string]struct{}{invalidItem: {}}, map[string]struct{}{})
		if err != nil {
			t.Fatalf("unexpected error while trying to init a allowDenyList: %v", wb)
		}
		err = wb.Parse()
		if err == nil {
			t.Fatalf("expected Parse() to fail for invalid regex pattern")
		}
	})

	t.Run("parses successfully when the allowDenyList has valid regexes", func(t *testing.T) {
		validItem := "kube_.*_info"
		wb, err := New(map[string]struct{}{validItem: {}}, map[string]struct{}{})
		if err != nil {
			t.Fatalf("unexpected error while trying to init a allowDenyList: %v", wb)
		}
		err = wb.Parse()
		if err != nil {
			t.Errorf("unexpected error while attempting to parse allowDenyList : %v", err)
		}
	})
}

func TestStatus(t *testing.T) {
	t.Run("status when allowlist has single item", func(t *testing.T) {
		item1 := "item1"
		allowlist, _ := New(map[string]struct{}{item1: {}}, map[string]struct{}{})
		actualStatusString := allowlist.Status()
		expectedStatusString := "Including the following lists that were on allowlist: " + item1
		if actualStatusString != expectedStatusString {
			t.Errorf("expected status %q but got %q", expectedStatusString, actualStatusString)
		}
	})
	t.Run("status when allowlist has multiple items", func(t *testing.T) {
		item1 := "item1"
		item2 := "item2"
		allowlist, _ := New(map[string]struct{}{item1: {}, item2: {}}, map[string]struct{}{})
		actualStatusString := allowlist.Status()
		expectedRegexPattern := `^Including the following lists that were on allowlist: (item1|item2), (item2|item1)$`
		matched, _ := regexp.MatchString(expectedRegexPattern, actualStatusString)
		if !matched {
			t.Errorf("expected status %q but got %q", expectedRegexPattern, actualStatusString)
		}
	})
	t.Run("status when denylist has single item", func(t *testing.T) {
		item1 := "not-empty"
		denylist, _ := New(map[string]struct{}{}, map[string]struct{}{item1: {}})
		actualStatusString := denylist.Status()
		expectedStatusString := "Excluding the following lists that were on denylist: " + item1
		if actualStatusString != expectedStatusString {
			t.Errorf("expected status %q but got %q", expectedStatusString, actualStatusString)
		}
	})
	t.Run("status when denylist has multiple items", func(t *testing.T) {
		item1 := "item1"
		item2 := "item2"
		denylist, _ := New(map[string]struct{}{}, map[string]struct{}{item1: {}, item2: {}})
		actualStatusString := denylist.Status()
		expectedRegexPattern := `^Excluding the following lists that were on denylist: (item1|item2), (item2|item1)$`
		matched, _ := regexp.MatchString(expectedRegexPattern, actualStatusString)
		if !matched {
			t.Errorf("expected status %q but got %q", expectedRegexPattern, actualStatusString)
		}
	})
}
