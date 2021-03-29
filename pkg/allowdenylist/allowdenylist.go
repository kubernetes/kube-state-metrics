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
	"strings"

	"github.com/pkg/errors"
)

// AllowDenyList encapsulates the logic needed to filter based on a string.
type AllowDenyList struct {
	list        map[string]struct{}
	rList       []*regexp.Regexp
	isAllowList bool
}

// New constructs a new AllowDenyList based on a allow- and a
// denylist. Only one of them can be not empty.
func New(allow, deny map[string]struct{}) (*AllowDenyList, error) {
	if len(allow) != 0 && len(deny) != 0 {
		return nil, errors.New(
			"allowlist and denylist are both set, they are mutually exclusive, only one of them can be set",
		)
	}

	var list map[string]struct{}
	var isAllowList bool

	// Default to denylisting
	if len(allow) != 0 {
		list = copyList(allow)
		isAllowList = true
	} else {
		list = copyList(deny)
		isAllowList = false
	}

	return &AllowDenyList{
		list:        list,
		isAllowList: isAllowList,
	}, nil
}

// Parse parses and compiles all of the regexes in the allowDenyList.
func (l *AllowDenyList) Parse() error {
	regexes := make([]*regexp.Regexp, 0, len(l.list))
	for item := range l.list {
		r, err := regexp.Compile(item)
		if err != nil {
			return err
		}
		regexes = append(regexes, r)
	}
	l.rList = regexes
	return nil
}

// Include includes the given items in the list.
func (l *AllowDenyList) Include(items []string) {
	if l.isAllowList {
		for _, item := range items {
			l.list[item] = struct{}{}
		}
	} else {
		for _, item := range items {
			delete(l.list, item)
		}
	}
}

// Exclude excludes the given items from the list.
func (l *AllowDenyList) Exclude(items []string) {
	if l.isAllowList {
		for _, item := range items {
			delete(l.list, item)
		}
	} else {
		for _, item := range items {
			l.list[item] = struct{}{}
		}
	}
}

// IsIncluded returns if the given item is included.
func (l *AllowDenyList) IsIncluded(item string) bool {
	var matched bool
	for _, r := range l.rList {
		matched = r.MatchString(item)
		if matched {
			break
		}
	}

	if l.isAllowList {
		return matched
	}

	return !matched
}

// IsExcluded returns if the given item is excluded.
func (l *AllowDenyList) IsExcluded(item string) bool {
	return !l.IsIncluded(item)
}

// Status returns the status of the AllowDenyList that can e.g. be passed into
// a logger.
func (l *AllowDenyList) Status() string {
	items := make([]string, 0, len(l.list))
	for key := range l.list {
		items = append(items, key)
	}

	if l.isAllowList {
		return "Including the following lists that were on allowlist: " + strings.Join(items, ", ")
	}

	return "Excluding the following lists that were on denylist: " + strings.Join(items, ", ")
}

func copyList(l map[string]struct{}) map[string]struct{} {
	newList := map[string]struct{}{}
	for k, v := range l {
		newList[k] = v
	}
	return newList
}
