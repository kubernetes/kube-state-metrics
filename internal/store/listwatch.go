/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

package store

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type labelSelectorListerWatcher struct {
	lw            cache.ListerWatcher
	labelSelector string
}

func withLabelSelector(lw cache.ListerWatcher, labelSelector string) cache.ListerWatcher {
	if labelSelector == "" {
		return lw
	}

	return &labelSelectorListerWatcher{
		lw:            lw,
		labelSelector: labelSelector,
	}
}

func (l *labelSelectorListerWatcher) List(options metav1.ListOptions) (runtime.Object, error) {
	options.LabelSelector = l.labelSelector
	return l.lw.List(options)
}

func (l *labelSelectorListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	options.LabelSelector = l.labelSelector
	return l.lw.Watch(options)
}

func (l *labelSelectorListerWatcher) IsWatchListSemanticsUnSupported() bool {
	type unsupported interface {
		IsWatchListSemanticsUnSupported() bool
	}

	if u, ok := l.lw.(unsupported); ok {
		return u.IsWatchListSemanticsUnSupported()
	}

	return false
}
