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

// TODO: rename collector
package collectors

import (
	"strings"

	apps "k8s.io/api/apps/v1beta1"
	autoscaling "k8s.io/api/autoscaling/v2beta1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	extensions "k8s.io/api/extensions/v1beta1"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-state-metrics/pkg/metrics"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
	"k8s.io/kube-state-metrics/pkg/options"
)

// Builder helps to build collectors. It follows the builder pattern
// (https://en.wikipedia.org/wiki/Builder_pattern).
type Builder struct {
	kubeClient        clientset.Interface
	namespaces        options.NamespaceList
	opts              *options.Options
	ctx               context.Context
	enabledCollectors options.CollectorSet
}

// NewBuilder returns a new builder.
func NewBuilder(
	ctx context.Context,
	opts *options.Options,
) *Builder {
	return &Builder{
		opts: opts,
		ctx:  ctx,
	}
}

// WithEnabledCollectors sets the enabledCollectors property of a Builder.
func (b *Builder) WithEnabledCollectors(c options.CollectorSet) {
	b.enabledCollectors = c
}

// WithNamespaces sets the namespaces property of a Builder.
func (b *Builder) WithNamespaces(n options.NamespaceList) {
	b.namespaces = n
}

// WithKubeClient sets the kubeClient property of a Builder.
func (b *Builder) WithKubeClient(c clientset.Interface) {
	b.kubeClient = c
}

// Build initializes and registers all enabled collectors.
func (b *Builder) Build() []*Collector {

	collectors := []*Collector{}
	activeCollectorNames := []string{}

	for c := range b.enabledCollectors {
		constructor, ok := availableCollectors[c]
		if ok {
			collector := constructor(b)
			activeCollectorNames = append(activeCollectorNames, c)
			collectors = append(collectors, collector)
		}
		// TODO: What if not ok?
	}

	glog.Infof("Active collectors: %s", strings.Join(activeCollectorNames, ","))

	return collectors
}

var availableCollectors = map[string]func(f *Builder) *Collector{
	"configmaps":               func(b *Builder) *Collector { return b.buildConfigMapCollector() },
	"cronjobs":                 func(b *Builder) *Collector { return b.buildCronJobCollector() },
	"daemonsets":               func(b *Builder) *Collector { return b.buildDaemonSetCollector() },
	"deployments":              func(b *Builder) *Collector { return b.buildDeploymentCollector() },
	"endpoints":                func(b *Builder) *Collector { return b.buildEndpointsCollector() },
	"horizontalpodautoscalers": func(b *Builder) *Collector { return b.buildHPACollector() },
	"jobs":                   func(b *Builder) *Collector { return b.buildJobCollector() },
	"limitranges":            func(b *Builder) *Collector { return b.buildLimitRangeCollector() },
	"namespaces":             func(b *Builder) *Collector { return b.buildNamespaceCollector() },
	"nodes":                  func(b *Builder) *Collector { return b.buildNodeCollector() },
	"persistentvolumeclaims": func(b *Builder) *Collector { return b.buildPersistentVolumeClaimCollector() },
	"persistentvolumes":      func(b *Builder) *Collector { return b.buildPersistentVolumeCollector() },
	"poddisruptionbudgets":   func(b *Builder) *Collector { return b.buildPodDisruptionBudgetCollector() },
	"pods":                   func(b *Builder) *Collector { return b.buildPodCollector() },
	"replicasets":            func(b *Builder) *Collector { return b.buildReplicaSetCollector() },
	"replicationcontrollers": func(b *Builder) *Collector { return b.buildReplicationControllerCollector() },
	"resourcequotas":         func(b *Builder) *Collector { return b.buildResourceQuotaCollector() },
	"secrets":                func(b *Builder) *Collector { return b.buildSecretCollector() },
	"services":               func(b *Builder) *Collector { return b.buildServiceCollector() },
	"statefulsets":           func(b *Builder) *Collector { return b.buildStatefulSetCollector() },
}

func (b *Builder) buildPodCollector() *Collector {
	genFunc := func(obj interface{}) []*metrics.Metric {
		return generatePodMetrics(b.opts.DisablePodNonGenericResourceMetrics, obj)
	}
	store := metricsstore.NewMetricsStore(genFunc)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Pod{}, store, b.namespaces, createPodListWatch)

	return NewCollector(store)
}

func (b *Builder) buildCronJobCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateCronJobMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &batchv1beta1.CronJob{}, store, b.namespaces, createCronJobListWatch)

	return NewCollector(store)
}

func (b *Builder) buildConfigMapCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateConfigMapMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.ConfigMap{}, store, b.namespaces, createConfigMapListWatch)

	return NewCollector(store)
}

func (b *Builder) buildDaemonSetCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateDaemonSetMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &extensions.DaemonSet{}, store, b.namespaces, createDaemonSetListWatch)

	return NewCollector(store)
}

func (b *Builder) buildDeploymentCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateDeploymentMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &extensions.Deployment{}, store, b.namespaces, createDeploymentListWatch)

	return NewCollector(store)
}

func (b *Builder) buildEndpointsCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateEndpointsMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Endpoints{}, store, b.namespaces, createEndpointsListWatch)

	return NewCollector(store)
}

func (b *Builder) buildHPACollector() *Collector {
	store := metricsstore.NewMetricsStore(generateHPAMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &autoscaling.HorizontalPodAutoscaler{}, store, b.namespaces, createHPAListWatch)

	return NewCollector(store)
}

func (b *Builder) buildJobCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateJobMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &batchv1.Job{}, store, b.namespaces, createJobListWatch)

	return NewCollector(store)
}

func (b *Builder) buildLimitRangeCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateLimitRangeMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.LimitRange{}, store, b.namespaces, createLimitRangeListWatch)

	return NewCollector(store)
}

func (b *Builder) buildNamespaceCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateNamespaceMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Namespace{}, store, b.namespaces, createNamespaceListWatch)

	return NewCollector(store)
}

func (b *Builder) buildNodeCollector() *Collector {
	genFunc := func(obj interface{}) []*metrics.Metric {
		return generateNodeMetrics(b.opts.DisableNodeNonGenericResourceMetrics, obj)
	}
	store := metricsstore.NewMetricsStore(genFunc)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Node{}, store, b.namespaces, createNodeListWatch)

	return NewCollector(store)
}

func (b *Builder) buildPersistentVolumeCollector() *Collector {
	store := metricsstore.NewMetricsStore(generatePersistentVolumeMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.PersistentVolume{}, store, b.namespaces, createPersistentVolumeListWatch)

	return NewCollector(store)
}

func (b *Builder) buildPersistentVolumeClaimCollector() *Collector {
	store := metricsstore.NewMetricsStore(generatePersistentVolumeClaimMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.PersistentVolumeClaim{}, store, b.namespaces, createPersistentVolumeClaimListWatch)

	return NewCollector(store)
}

func (b *Builder) buildPodDisruptionBudgetCollector() *Collector {
	store := metricsstore.NewMetricsStore(generatePodDisruptionBudgetMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1beta1.PodDisruptionBudget{}, store, b.namespaces, createPodDisruptionBudgetListWatch)

	return NewCollector(store)
}

func (b *Builder) buildReplicaSetCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateReplicaSetMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &extensions.ReplicaSet{}, store, b.namespaces, createReplicaSetListWatch)

	return NewCollector(store)
}

func (b *Builder) buildReplicationControllerCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateReplicationControllerMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.ReplicationController{}, store, b.namespaces, createReplicationControllerListWatch)

	return NewCollector(store)
}

func (b *Builder) buildResourceQuotaCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateResourceQuotaMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.ResourceQuota{}, store, b.namespaces, createResourceQuotaListWatch)

	return NewCollector(store)
}

func (b *Builder) buildSecretCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateSecretMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Secret{}, store, b.namespaces, createSecretListWatch)

	return NewCollector(store)
}

func (b *Builder) buildServiceCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateServiceMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Service{}, store, b.namespaces, createServiceListWatch)

	return NewCollector(store)
}

func (b *Builder) buildStatefulSetCollector() *Collector {
	store := metricsstore.NewMetricsStore(generateStatefulSetMetrics)
	reflectorPerNamespace(b.ctx, b.kubeClient, &apps.StatefulSet{}, store, b.namespaces, createStatefulSetListWatch)

	return NewCollector(store)
}

func reflectorPerNamespace(
	ctx context.Context,
	kubeClient clientset.Interface,
	expectedType interface{},
	store cache.Store,
	namespaces []string,
	listWatchFunc func(kubeClient clientset.Interface, ns string) cache.ListWatch,
) {
	for _, ns := range namespaces {
		lw := listWatchFunc(kubeClient, ns)
		reflector := cache.NewReflector(&lw, expectedType, store, 0)
		go reflector.Run(ctx.Done())
	}
}
