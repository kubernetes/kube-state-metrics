/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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

package listwatch

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// denylistListerWatcher implements cache.ListerWatcher
// which wraps a cache.ListerWatcher,
// filtering list results and watch events by denied namespaces.
type denylistListerWatcher struct {
	denylist map[string]struct{}
	next     cache.ListerWatcher
}

// newDenylistListerWatcher creates a cache.ListerWatcher
// wrapping the given next cache.ListerWatcher
// filtering lists and watch events by the given namespaces.
func newDenylistListerWatcher(namespaces []string, next cache.ListerWatcher) cache.ListerWatcher {
	if len(namespaces) == 0 {
		return next
	}

	denylist := make(map[string]struct{})

	for _, ns := range namespaces {
		denylist[ns] = struct{}{}
	}

	return &denylistListerWatcher{
		denylist: denylist,
		next:     next,
	}
}

// List lists the wrapped next listerwatcher List result,
// but filtering denied namespaces from the result.
func (w *denylistListerWatcher) List(options metav1.ListOptions) (runtime.Object, error) {
	l := metav1.List{}

	list, err := w.next.List(options)
	if err != nil {
		klog.Errorf("error listing: %v", err)
		return nil, err
	}

	objs, err := meta.ExtractList(list)
	if err != nil {
		klog.Errorf("error extracting list: %v", err)
		return nil, err
	}

	metaObj, err := meta.ListAccessor(list)
	if err != nil {
		klog.Errorf("error getting list accessor: %v", err)
		return nil, err
	}

	for _, obj := range objs {
		acc, err := meta.Accessor(obj)
		if err != nil {
			klog.Errorf("error getting meta accessor accessor for object %s: %v", fmt.Sprintf("%v", obj), err)
			return nil, err
		}

		if _, denied := w.denylist[getNamespace(acc)]; denied {
			klog.V(8).Infof("denied %s", acc.GetSelfLink())
			continue
		}

		klog.V(8).Infof("allowed %s", acc.GetSelfLink())

		l.Items = append(l.Items, runtime.RawExtension{Object: obj.DeepCopyObject()})
	}

	l.ListMeta.ResourceVersion = metaObj.GetResourceVersion()
	return &l, nil
}

// Watch
func (w *denylistListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	nextWatch, err := w.next.Watch(options)
	if err != nil {
		return nil, err
	}

	return newDenylistWatch(w.denylist, nextWatch), nil
}

// newDenylistWatch creates a new watch.Interface,
// wrapping the given next watcher,
// and filtering watch events by the given namespaces.
//
// It starts a new goroutine until either
// a) the result channel of the wrapped next watcher is closed, or
// b) Stop() was invoked on the returned watcher.
func newDenylistWatch(denylist map[string]struct{}, next watch.Interface) watch.Interface {
	var (
		result = make(chan watch.Event)
		proxy  = watch.NewProxyWatcher(result)
	)

	go func() {
		defer func() {
			klog.V(8).Info("stopped denylist watcher")
			// According to watch.Interface the result channel is supposed to be called
			// in case of error or if the listwach is closed, see [1].
			//
			// [1] https://github.com/kubernetes/apimachinery/blob/533d101be9a6450773bb2829bef282b6b7c4ff6d/pkg/watch/watch.go#L34-L37
			close(result)
		}()

		for {
			select {
			case event, ok := <-next.ResultChan():
				if !ok {
					klog.V(8).Info("result channel closed")
					return
				}

				acc, err := meta.Accessor(event.Object)
				if err != nil {
					// ignore this event, it doesn't implement the metav1.Object interface,
					// hence we cannot determine its namespace.
					klog.V(6).Infof("unexpected object type in event (%T): %v", event.Object, event.Object)
					continue
				}

				if _, denied := denylist[getNamespace(acc)]; denied {
					klog.V(8).Infof("denied %s", acc.GetSelfLink())
					continue
				}

				klog.V(8).Infof("allowed %s", acc.GetSelfLink())

				select {
				case result <- event:
					klog.V(8).Infof("dispatched %s", acc.GetSelfLink())
				case <-proxy.StopChan():
					next.Stop()
					return
				}
			case <-proxy.StopChan():
				next.Stop()
				return
			}
		}
	}()

	return proxy
}

// getNamespace returns the namespace of the given object.
// If the object is itself a namespace, it returns the object's
// name.
func getNamespace(obj metav1.Object) string {
	if _, ok := obj.(*v1.Namespace); ok {
		return obj.GetName()
	}
	return obj.GetNamespace()
}
