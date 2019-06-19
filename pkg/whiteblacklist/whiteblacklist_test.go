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

package whiteblacklist

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

	t.Run("defaults to blacklisting", func(t *testing.T) {
		l, err := New(map[string]struct{}{}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		if l.isWhiteList {
			t.Fatal("expected whiteBlackList to default to blacklist")
		}
	})

	t.Run("if whitelist set, should be whitelist", func(t *testing.T) {
		list, err := New(map[string]struct{}{"not-empty": {}}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		if !list.isWhiteList {
			t.Fatal("expected list to be whitelist")
		}
	})

	t.Run("if blacklist set, should be blacklist", func(t *testing.T) {
		list, err := New(map[string]struct{}{}, map[string]struct{}{"not-empty": {}})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		if list.isWhiteList {
			t.Fatal("expected list to be blacklist")
		}
	})
}

func TestInclude(t *testing.T) {
	t.Run("adds when whitelist", func(t *testing.T) {
		whitelist, err := New(map[string]struct{}{"not-empty": {}}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		whitelist.Include([]string{"item1"})
		err = whitelist.Parse()
		if err != nil {
			t.Fatal("expected Parse() to not fail")
		}

		if !whitelist.IsIncluded("item1") {
			t.Fatal("expected included item to be included")
		}
	})
	t.Run("removes when blacklist", func(t *testing.T) {
		item1 := "item1"
		blacklist, err := New(map[string]struct{}{}, map[string]struct{}{item1: {}})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		blacklist.Include([]string{item1})
		err = blacklist.Parse()
		if err != nil {
			t.Fatalf("expected Parse() to not fail, but got error : %v", err)
		}

		if !blacklist.IsIncluded(item1) {
			t.Fatal("expected included item to be included")
		}
	})
	t.Run("adds during pattern match when in whitelist mode", func(t *testing.T) {
		whitelist, err := New(map[string]struct{}{"not-empty": {}}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		whitelist.Include([]string{"kube_.*_info"})
		err = whitelist.Parse()
		if err != nil {
			t.Fatalf("expected Parse() to not fail, but got error : %v", err)
		}

		if !whitelist.IsIncluded("kube_secret_info") {
			t.Fatal("expected included item to be included")
		}
	})
	t.Run("removes during pattern match when in blackist mode", func(t *testing.T) {
		item1 := "kube_pod_container_resource_requests_cpu_cores"
		item2 := "kube_pod_container_resource_requests_memory_bytes"
		item3 := "kube_node_status_capacity_cpu_cores"
		item4 := "kube_node_status_capacity_memory_bytes"

		blacklist, err := New(map[string]struct{}{}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		blacklist.Exclude([]string{"kube_node_.*_cores", "kube_pod_.*_bytes"})
		err = blacklist.Parse()
		if err != nil {
			t.Fatalf("expected Parse() to not fail, but got error : %v", err)
		}

		if blacklist.IsExcluded(item1) {
			t.Fatalf("expected included %s to be included", item1)
		}
		if blacklist.IsIncluded(item2) {
			t.Fatalf("expected included %s to be excluded", item2)
		}
		if blacklist.IsIncluded(item3) {
			t.Fatalf("expected included %s to be excluded", item3)
		}
		if blacklist.IsExcluded(item4) {
			t.Fatalf("expected included %s to be included", item4)
		}
	})
}

func TestExclude(t *testing.T) {
	t.Run("removes when whitelist", func(t *testing.T) {
		item1 := "item1"
		whitelist, err := New(map[string]struct{}{item1: {}}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		whitelist.Exclude([]string{item1})
		err = whitelist.Parse()
		if err != nil {
			t.Fatalf("expected Parse() to not fail, but got error : %v", err)
		}

		if whitelist.IsIncluded(item1) {
			t.Fatal("expected excluded item to be excluded")
		}
	})
	t.Run("removes when blacklist", func(t *testing.T) {
		item1 := "item1"
		blacklist, err := New(map[string]struct{}{}, map[string]struct{}{"not-empty": {}})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		blacklist.Exclude([]string{item1})
		err = blacklist.Parse()
		if err != nil {
			t.Fatalf("expected Parse() to not fail, but got error : %v", err)
		}

		if blacklist.IsIncluded(item1) {
			t.Fatal("expected excluded item to be excluded")
		}
	})
}

func TestParse(t *testing.T) {
	t.Run("fails when an unparseable regex is passed", func(t *testing.T) {
		invalidItem := "*_pod_info"
		wb, err := New(map[string]struct{}{invalidItem: {}}, map[string]struct{}{})
		if err != nil {
			t.Fatalf("unexpected error while trying to init a whiteBlackList: %v", wb)
		}
		err = wb.Parse()
		if err == nil {
			t.Fatalf("expected Parse() to fail for invalid regex pattern")
		}
	})

	t.Run("parses successfully when the whiteBlackList has valid regexes", func(t *testing.T) {
		validItem := "kube_.*_info"
		wb, err := New(map[string]struct{}{validItem: {}}, map[string]struct{}{})
		if err != nil {
			t.Fatalf("unexpected error while trying to init a whiteBlackList: %v", wb)
		}
		err = wb.Parse()
		if err != nil {
			t.Errorf("unexpected error while attempting to parse whiteBlackList : %v", err)
		}
	})
}

func TestStatus(t *testing.T) {
	t.Run("status when whitelist has single item", func(t *testing.T) {
		item1 := "item1"
		whitelist, _ := New(map[string]struct{}{item1: {}}, map[string]struct{}{})
		actualStatusString := whitelist.Status()
		expectedStatusString := "whitelisting the following items: " + item1
		if actualStatusString != expectedStatusString {
			t.Errorf("expected status %q but got %q", expectedStatusString, actualStatusString)
		}
	})
	t.Run("status when whitelist has multiple items", func(t *testing.T) {
		item1 := "item1"
		item2 := "item2"
		whitelist, _ := New(map[string]struct{}{item1: {}, item2: {}}, map[string]struct{}{})
		actualStatusString := whitelist.Status()
		expectedRegexPattern := `^whitelisting the following items: (item1|item2), (item2|item1)$`
		matched, _ := regexp.MatchString(expectedRegexPattern, actualStatusString)
		if !matched {
			t.Errorf("expected status %q but got %q", expectedRegexPattern, actualStatusString)
		}
	})
	t.Run("status when blacklist has single item", func(t *testing.T) {
		item1 := "not-empty"
		blacklist, _ := New(map[string]struct{}{}, map[string]struct{}{item1: {}})
		actualStatusString := blacklist.Status()
		expectedStatusString := "blacklisting the following items: " + item1
		if actualStatusString != expectedStatusString {
			t.Errorf("expected status %q but got %q", expectedStatusString, actualStatusString)
		}
	})
	t.Run("status when blacklist has multiple items", func(t *testing.T) {
		item1 := "item1"
		item2 := "item2"
		blacklist, _ := New(map[string]struct{}{}, map[string]struct{}{item1: {}, item2: {}})
		actualStatusString := blacklist.Status()
		expectedRegexPattern := `^blacklisting the following items: (item1|item2), (item2|item1)$`
		matched, _ := regexp.MatchString(expectedRegexPattern, actualStatusString)
		if !matched {
			t.Errorf("expected status %q but got %q", expectedRegexPattern, actualStatusString)
		}
	})
}
