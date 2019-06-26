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

		if !blacklist.IsIncluded(item1) {
			t.Fatal("expected included item to be included")
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

		if blacklist.IsIncluded(item1) {
			t.Fatal("expected excluded item to be excluded")
		}
	})
}
