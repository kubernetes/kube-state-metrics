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
	"errors"
	"strings"
	"sync"
	"time"

	regexp "github.com/dlclark/regexp2"
	"k8s.io/klog/v2"

	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

// Use ECMAScript as the default regexp spec to support lookarounds (#2594).
var (
	once                 sync.Once
	regexpDefaultSpec    regexp.RegexOptions = regexp.ECMAScript
	regexpDefaultTimeout                     = time.Minute
)

// AllowDenyList namespaceencapsulates the logic needed to filter based on a string.
type AllowDenyList struct {
	list        map[string]struct{}
	rList       []*regexp.Regexp
	isAllowList bool
}

// New constructs a new AllowDenyList based on a allow- and a
// denylist. Only one of them can be not empty.
func New(allow, deny map[string]struct{}) (*AllowDenyList, error) {
	once.Do(func() {
		regexp.DefaultMatchTimeout = regexpDefaultTimeout
	})
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
		r, err := regexp.Compile(item, regexpDefaultSpec)
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
func (l *AllowDenyList) IsIncluded(item string) (bool, error) {
	var (
		matched bool
		err     error
	)
	for _, r := range l.rList {
		matched, err = r.MatchString(item)
		if err != nil {
			return false, err
		}
		if matched {
			break
		}
	}

	if l.isAllowList {
		return matched, nil
	}

	return !matched, nil
}

// IsExcluded returns if the given item is excluded.
func (l *AllowDenyList) IsExcluded(item string) (bool, error) {
	isIncluded, err := l.IsIncluded(item)
	if err != nil {
		return false, err
	}

	return !isIncluded, nil
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

// Test returns if the given family generator passes (is included in) the AllowDenyList
func (l *AllowDenyList) Test(generator generator.FamilyGenerator) bool {
	isIncluded, err := l.IsIncluded(generator.Name)
	if err != nil {
		klog.ErrorS(err, "Error while processing allow-deny entries for generator", "generator", generator.Name)
		return false
	}

	return isIncluded
}

func copyList(l map[string]struct{}) map[string]struct{} {
	newList := map[string]struct{}{}
	for k, v := range l {
		newList[k] = v
	}
	return newList
}
