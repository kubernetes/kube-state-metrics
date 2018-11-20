package whiteblacklist

import (
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("fails with two non empty maps", func(t *testing.T) {
		_, err := New(map[string]struct{}{"not-empty": struct{}{}}, map[string]struct{}{"not-empty": struct{}{}})
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
		list, err := New(map[string]struct{}{"not-empty": struct{}{}}, map[string]struct{}{})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		if !list.isWhiteList {
			t.Fatal("expected list to be whitelist")
		}
	})

	t.Run("if blacklist set, should be blacklist", func(t *testing.T) {
		list, err := New(map[string]struct{}{}, map[string]struct{}{"not-empty": struct{}{}})
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
		whitelist, err := New(map[string]struct{}{"not-empty": struct{}{}}, map[string]struct{}{})
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
		blacklist, err := New(map[string]struct{}{}, map[string]struct{}{item1: struct{}{}})
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
		whitelist, err := New(map[string]struct{}{item1: struct{}{}}, map[string]struct{}{})
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
		blacklist, err := New(map[string]struct{}{}, map[string]struct{}{"not-empty": struct{}{}})
		if err != nil {
			t.Fatal("expected New() to not fail")
		}

		blacklist.Exclude([]string{item1})

		if blacklist.IsIncluded(item1) {
			t.Fatal("expected excluded item to be excluded")
		}
	})
}
