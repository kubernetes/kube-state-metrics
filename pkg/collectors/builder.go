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
	"sort"
	"strings"

	"k8s.io/kube-state-metrics/pkg/metrics"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
	"k8s.io/kube-state-metrics/pkg/options"

	apps "k8s.io/api/apps/v1beta1"
	autoscaling "k8s.io/api/autoscaling/v2beta1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	policy "k8s.io/api/policy/v1beta1"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type whiteBlackLister interface {
	IsIncluded(string) bool
	IsExcluded(string) bool
}

// Builder helps to build collectors. It follows the builder pattern
// (https://en.wikipedia.org/wiki/Builder_pattern).
type Builder struct {
	kubeClient        clientset.Interface
	namespaces        options.NamespaceList
	ctx               context.Context
	enabledCollectors []string
	whiteBlackList    whiteBlackLister
}

// NewBuilder returns a new builder.
func NewBuilder(
	ctx context.Context,
) *Builder {
	return &Builder{
		ctx: ctx,
	}
}

// WithEnabledCollectors sets the enabledCollectors property of a Builder.
func (b *Builder) WithEnabledCollectors(c []string) {
	copy := []string{}
	for _, s := range c {
		copy = append(copy, s)
	}

	sort.Strings(copy)

	b.enabledCollectors = copy
}

// WithNamespaces sets the namespaces property of a Builder.
func (b *Builder) WithNamespaces(n options.NamespaceList) {
	b.namespaces = n
}

// WithKubeClient sets the kubeClient property of a Builder.
func (b *Builder) WithKubeClient(c clientset.Interface) {
	b.kubeClient = c
}

// WithWhiteBlackList configures the white or blacklisted metrics to be exposed
// by the collectors build by the Builder
func (b *Builder) WithWhiteBlackList(l whiteBlackLister) {
	b.whiteBlackList = l
}

// Build initializes and registers all enabled collectors.
func (b *Builder) Build() []*Collector {
	if b.whiteBlackList == nil {
		panic("whiteBlackList should not be nil")
	}

	collectors := []*Collector{}
	activeCollectorNames := []string{}

	for _, c := range b.enabledCollectors {
		constructor, ok := availableCollectors[c]
		if ok {
			collector := constructor(b)
			activeCollectorNames = append(activeCollectorNames, c)
			collectors = append(collectors, collector)
		}
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
	"jobs":                     func(b *Builder) *Collector { return b.buildJobCollector() },
	"limitranges":              func(b *Builder) *Collector { return b.buildLimitRangeCollector() },
	"namespaces":               func(b *Builder) *Collector { return b.buildNamespaceCollector() },
	"nodes":                    func(b *Builder) *Collector { return b.buildNodeCollector() },
	"persistentvolumeclaims":   func(b *Builder) *Collector { return b.buildPersistentVolumeClaimCollector() },
	"persistentvolumes":        func(b *Builder) *Collector { return b.buildPersistentVolumeCollector() },
	"poddisruptionbudgets":     func(b *Builder) *Collector { return b.buildPodDisruptionBudgetCollector() },
	"pods":                     func(b *Builder) *Collector { return b.buildPodCollector() },
	"replicasets":              func(b *Builder) *Collector { return b.buildReplicaSetCollector() },
	"replicationcontrollers":   func(b *Builder) *Collector { return b.buildReplicationControllerCollector() },
	"resourcequotas":           func(b *Builder) *Collector { return b.buildResourceQuotaCollector() },
	"secrets":                  func(b *Builder) *Collector { return b.buildSecretCollector() },
	"services":                 func(b *Builder) *Collector { return b.buildServiceCollector() },
	"statefulsets":             func(b *Builder) *Collector { return b.buildStatefulSetCollector() },
}

func (b *Builder) buildConfigMapCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, configMapMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.ConfigMap{}, store, b.namespaces, createConfigMapListWatch)

	return NewCollector(store)
}

func (b *Builder) buildCronJobCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, cronJobMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &batchv1beta1.CronJob{}, store, b.namespaces, createCronJobListWatch)

	return NewCollector(store)
}

func (b *Builder) buildDaemonSetCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, daemonSetMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &extensions.DaemonSet{}, store, b.namespaces, createDaemonSetListWatch)

	return NewCollector(store)
}

func (b *Builder) buildDeploymentCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, deploymentMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &extensions.Deployment{}, store, b.namespaces, createDeploymentListWatch)

	return NewCollector(store)
}

func (b *Builder) buildEndpointsCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, endpointMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Endpoints{}, store, b.namespaces, createEndpointsListWatch)

	return NewCollector(store)
}

func (b *Builder) buildHPACollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, hpaMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &autoscaling.HorizontalPodAutoscaler{}, store, b.namespaces, createHPAListWatch)

	return NewCollector(store)
}

func (b *Builder) buildJobCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, jobMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &batchv1.Job{}, store, b.namespaces, createJobListWatch)

	return NewCollector(store)
}

func (b *Builder) buildLimitRangeCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, limitRangeMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.LimitRange{}, store, b.namespaces, createLimitRangeListWatch)

	return NewCollector(store)
}

func (b *Builder) buildNamespaceCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, namespaceMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Namespace{}, store, b.namespaces, createNamespaceListWatch)

	return NewCollector(store)
}

func (b *Builder) buildNodeCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, nodeMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Node{}, store, b.namespaces, createNodeListWatch)

	return NewCollector(store)
}

func (b *Builder) buildPersistentVolumeClaimCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, persistentVolumeClaimMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.PersistentVolumeClaim{}, store, b.namespaces, createPersistentVolumeClaimListWatch)

	return NewCollector(store)
}

func (b *Builder) buildPersistentVolumeCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, persistentVolumeMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.PersistentVolume{}, store, b.namespaces, createPersistentVolumeListWatch)

	return NewCollector(store)
}

func (b *Builder) buildPodDisruptionBudgetCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, podDisruptionBudgetMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &policy.PodDisruptionBudget{}, store, b.namespaces, createPodDisruptionBudgetListWatch)

	return NewCollector(store)
}

func (b *Builder) buildReplicaSetCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, replicaSetMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &extensions.ReplicaSet{}, store, b.namespaces, createReplicaSetListWatch)

	return NewCollector(store)
}

func (b *Builder) buildReplicationControllerCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, replicationControllerMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.ReplicationController{}, store, b.namespaces, createReplicationControllerListWatch)

	return NewCollector(store)
}

func (b *Builder) buildResourceQuotaCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, resourceQuotaMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.ResourceQuota{}, store, b.namespaces, createResourceQuotaListWatch)

	return NewCollector(store)
}

func (b *Builder) buildSecretCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, secretMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Secret{}, store, b.namespaces, createSecretListWatch)

	return NewCollector(store)
}

func (b *Builder) buildServiceCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, serviceMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Service{}, store, b.namespaces, createServiceListWatch)

	return NewCollector(store)
}

func (b *Builder) buildStatefulSetCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, statefulSetMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &apps.StatefulSet{}, store, b.namespaces, createStatefulSetListWatch)

	return NewCollector(store)
}

func (b *Builder) buildPodCollector() *Collector {
	filteredMetricFamilies := filterMetricFamilies(b.whiteBlackList, podMetricFamilies)
	composedMetricGenFuncs := composeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := extractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	reflectorPerNamespace(b.ctx, b.kubeClient, &v1.Pod{}, store, b.namespaces, createPodListWatch)

	return NewCollector(store)
}

func extractMetricFamilyHeaders(families []metrics.FamilyGenerator) []string {
	headers := make([]string, len(families))

	for i, f := range families {
		header := strings.Builder{}

		header.WriteString("# HELP ")
		header.WriteString(f.Name)
		header.WriteByte(' ')
		header.WriteString(f.Help)
		header.WriteByte('\n')
		header.WriteString("# TYPE ")
		header.WriteString(f.Name)
		header.WriteByte(' ')
		header.WriteString(string(f.Type))

		headers[i] = header.String()
	}

	return headers
}

// composeMetricGenFuncs takes a slice of metric families and returns a function
// that composes their metric generation functions into a single one.
func composeMetricGenFuncs(families []metrics.FamilyGenerator) func(obj interface{}) []metricsstore.FamilyStringer {
	funcs := []func(obj interface{}) metrics.Family{}

	for _, f := range families {
		funcs = append(funcs, f.GenerateFunc)
	}

	return func(obj interface{}) []metricsstore.FamilyStringer {
		families := make([]metricsstore.FamilyStringer, len(funcs))

		for i, f := range funcs {
			families[i] = f(obj)
		}

		return families
	}
}

// filterMetricFamilies takes a white- and a blacklist and a slice of metric
// families and returns a filtered slice.
func filterMetricFamilies(l whiteBlackLister, families []metrics.FamilyGenerator) []metrics.FamilyGenerator {
	filtered := []metrics.FamilyGenerator{}

	for _, f := range families {
		if l.IsIncluded(f.Name) {
			filtered = append(filtered, f)
		}
	}

	return filtered
}

// reflectorPerNamespace creates a Kubernetes client-go reflector with the given
// listWatchFunc for each given namespace and registers it with the given store.
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
