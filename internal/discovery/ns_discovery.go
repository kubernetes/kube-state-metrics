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

package discovery

import (
	"context"
	"errors"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type NamespaceDiscoverer struct {
	labelSelector string
	fieldSelector string

	namespaces           map[string]struct{}
	mtx                  *sync.RWMutex
	shouldRebuildMetrics bool
}

type Opt func(*NamespaceDiscoverer)

func NewNamespaceDiscoverer(opts ...Opt) NamespaceDiscoverer {
	d := NamespaceDiscoverer{
		namespaces: make(map[string]struct{}),
		mtx:        &sync.RWMutex{},
	}
	for _, opt := range opts {
		opt(&d)
	}
	return d
}

func WithLabelSelector(s string) Opt {
	return func(d *NamespaceDiscoverer) {
		d.labelSelector = s
	}
}

func WithFieldSelector(s string) Opt {
	return func(d *NamespaceDiscoverer) {
		d.fieldSelector = s
	}
}

func (d *NamespaceDiscoverer) Start(ctx context.Context, kubeClient clientset.Interface) ([]string, error) {
	informer := cache.NewSharedInformer(&cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			if d.fieldSelector != "" {
				opts.FieldSelector = d.fieldSelector
			}
			if d.labelSelector != "" {
				opts.LabelSelector = d.labelSelector
			}
			return kubeClient.CoreV1().Namespaces().List(ctx, opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			if d.fieldSelector != "" {
				opts.FieldSelector = d.fieldSelector
			}
			if d.labelSelector != "" {
				opts.LabelSelector = d.labelSelector
			}
			return kubeClient.CoreV1().Namespaces().Watch(ctx, opts)
		},
	}, &corev1.Namespace{}, 0)

	// TODO: add transform to only return name of namespace to avoid RAM usage

	handler, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			name := obj.(*corev1.Namespace).ObjectMeta.Name

			d.safeWrite(func() {
				d.namespaces[name] = struct{}{}
				d.shouldRebuildMetrics = true
			})
		},
		DeleteFunc: func(obj interface{}) {
			name := obj.(*corev1.Namespace).ObjectMeta.Name

			d.safeWrite(func() {
				delete(d.namespaces, name)
				d.shouldRebuildMetrics = true
			})
		},
	})
	if err != nil {
		return []string{}, err
	}

	go informer.RunWithContext(ctx)

	if !cache.WaitForCacheSync(ctx.Done(), handler.HasSynced) {
		return []string{}, errors.New("waiting for initial pre-sync events to be delivered failed")
	}

	var namespaces []string

	d.safeWrite(func() {
		d.shouldRebuildMetrics = false

		// TODO: refactor in its own function d.namespacesAsList or something
		namespaces = make([]string, len(d.namespaces))
		i := 0
		for namespace := range d.namespaces {
			namespaces[i] = namespace
			i++
		}
	})

	return namespaces, nil
}

func (d *NamespaceDiscoverer) PollForCacheUpdates(ctx context.Context, interval time.Duration) <-chan []string {
	notifyChan := make(chan []string)

	// The interval at which we will check the cache for updates.
	t := time.NewTicker(interval)

	go func() {
		for range t.C {
			select {
			case <-ctx.Done():
				klog.InfoS("context cancelled")
				close(notifyChan)
				t.Stop()
				return
			default:
				var namespaces []string
				shouldRebuildMetrics := false

				d.safeRead(func() {
					shouldRebuildMetrics = d.shouldRebuildMetrics

					if shouldRebuildMetrics {
						namespaces = make([]string, len(d.namespaces))
						i := 0
						for namespace := range d.namespaces {
							namespaces[i] = namespace
							i++
						}
					}
				})

				if shouldRebuildMetrics {
					d.safeWrite(func() {
						d.shouldRebuildMetrics = false
					})

					notifyChan <- namespaces
				}
			}
		}
	}()

	return notifyChan
}

// safeRead executes the given function while holding a read lock.
func (d *NamespaceDiscoverer) safeRead(f func()) {
	d.mtx.RLock()
	defer d.mtx.RUnlock()
	f()
}

// safeWrite executes the given function while holding a write lock.
func (d *NamespaceDiscoverer) safeWrite(f func()) {
	d.mtx.Lock()
	defer d.mtx.Unlock()
	f()
}
