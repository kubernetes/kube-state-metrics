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

package store

import (
	"context"
	"reflect"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	admissionregistration "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	certv1beta1 "k8s.io/api/certificates/v1beta1"
	v1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	policy "k8s.io/api/policy/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	vpaautoscaling "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"

	"k8s.io/kube-state-metrics/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
	"k8s.io/kube-state-metrics/pkg/options"
	"k8s.io/kube-state-metrics/pkg/sharding"
	"k8s.io/kube-state-metrics/pkg/watch"
)

type whiteBlackLister interface {
	IsIncluded(string) bool
	IsExcluded(string) bool
}

// Builder helps to build store. It follows the builder pattern
// (https://en.wikipedia.org/wiki/Builder_pattern).
type Builder struct {
	kubeClient       clientset.Interface
	vpaClient        vpaclientset.Interface
	namespaces       options.NamespaceList
	ctx              context.Context
	enabledResources []string
	whiteBlackList   whiteBlackLister
	metrics          *watch.ListWatchMetrics
	shard            int32
	totalShards      int
}

// NewBuilder returns a new builder.
func NewBuilder() *Builder { return &Builder{} }

// WithMetrics sets the metrics property of a Builder.
func (b *Builder) WithMetrics(r *prometheus.Registry) {
	b.metrics = watch.NewListWatchMetrics(r)
}

// WithEnabledResources sets the enabledResources property of a Builder.
func (b *Builder) WithEnabledResources(c []string) error {
	for _, col := range c {
		if !collectorExists(col) {
			return errors.Errorf("collector %s does not exist. Available collectors: %s", col, strings.Join(availableCollectors(), ","))
		}
	}

	var copy []string
	copy = append(copy, c...)

	sort.Strings(copy)

	b.enabledResources = copy
	return nil
}

// WithNamespaces sets the namespaces property of a Builder.
func (b *Builder) WithNamespaces(n options.NamespaceList) {
	b.namespaces = n
}

// WithSharding sets the shard and totalShards property of a Builder.
func (b *Builder) WithSharding(shard int32, totalShards int) {
	b.shard = shard
	b.totalShards = totalShards
}

// WithContext sets the ctx property of a Builder.
func (b *Builder) WithContext(ctx context.Context) {
	b.ctx = ctx
}

// WithKubeClient sets the kubeClient property of a Builder.
func (b *Builder) WithKubeClient(c clientset.Interface) {
	b.kubeClient = c
}

// WithVPAClient sets the vpaClient property of a Builder so that the verticalpodautoscaler collector can query VPA objects.
func (b *Builder) WithVPAClient(c vpaclientset.Interface) {
	b.vpaClient = c
}

// WithWhiteBlackList configures the white or blacklisted metric to be exposed
// by the store build by the Builder.
func (b *Builder) WithWhiteBlackList(l whiteBlackLister) {
	b.whiteBlackList = l
}

// Build initializes and registers all enabled stores.
func (b *Builder) Build() []*metricsstore.MetricsStore {
	if b.whiteBlackList == nil {
		panic("whiteBlackList should not be nil")
	}

	stores := []*metricsstore.MetricsStore{}
	activeStoreNames := []string{}

	for _, c := range b.enabledResources {
		constructor, ok := availableStores[c]
		if ok {
			store := constructor(b)
			activeStoreNames = append(activeStoreNames, c)
			stores = append(stores, store)
		}
	}

	klog.Infof("Active collectors: %s", strings.Join(activeStoreNames, ","))

	return stores
}

var availableStores = map[string]func(f *Builder) *metricsstore.MetricsStore{
	"certificatesigningrequests":      func(b *Builder) *metricsstore.MetricsStore { return b.buildCsrStore() },
	"configmaps":                      func(b *Builder) *metricsstore.MetricsStore { return b.buildConfigMapStore() },
	"cronjobs":                        func(b *Builder) *metricsstore.MetricsStore { return b.buildCronJobStore() },
	"daemonsets":                      func(b *Builder) *metricsstore.MetricsStore { return b.buildDaemonSetStore() },
	"deployments":                     func(b *Builder) *metricsstore.MetricsStore { return b.buildDeploymentStore() },
	"endpoints":                       func(b *Builder) *metricsstore.MetricsStore { return b.buildEndpointsStore() },
	"horizontalpodautoscalers":        func(b *Builder) *metricsstore.MetricsStore { return b.buildHPAStore() },
	"ingresses":                       func(b *Builder) *metricsstore.MetricsStore { return b.buildIngressStore() },
	"jobs":                            func(b *Builder) *metricsstore.MetricsStore { return b.buildJobStore() },
	"limitranges":                     func(b *Builder) *metricsstore.MetricsStore { return b.buildLimitRangeStore() },
	"mutatingwebhookconfigurations":   func(b *Builder) *metricsstore.MetricsStore { return b.buildMutatingWebhookConfigurationStore() },
	"namespaces":                      func(b *Builder) *metricsstore.MetricsStore { return b.buildNamespaceStore() },
	"nodes":                           func(b *Builder) *metricsstore.MetricsStore { return b.buildNodeStore() },
	"persistentvolumeclaims":          func(b *Builder) *metricsstore.MetricsStore { return b.buildPersistentVolumeClaimStore() },
	"persistentvolumes":               func(b *Builder) *metricsstore.MetricsStore { return b.buildPersistentVolumeStore() },
	"poddisruptionbudgets":            func(b *Builder) *metricsstore.MetricsStore { return b.buildPodDisruptionBudgetStore() },
	"pods":                            func(b *Builder) *metricsstore.MetricsStore { return b.buildPodStore() },
	"replicasets":                     func(b *Builder) *metricsstore.MetricsStore { return b.buildReplicaSetStore() },
	"replicationcontrollers":          func(b *Builder) *metricsstore.MetricsStore { return b.buildReplicationControllerStore() },
	"resourcequotas":                  func(b *Builder) *metricsstore.MetricsStore { return b.buildResourceQuotaStore() },
	"secrets":                         func(b *Builder) *metricsstore.MetricsStore { return b.buildSecretStore() },
	"services":                        func(b *Builder) *metricsstore.MetricsStore { return b.buildServiceStore() },
	"statefulsets":                    func(b *Builder) *metricsstore.MetricsStore { return b.buildStatefulSetStore() },
	"storageclasses":                  func(b *Builder) *metricsstore.MetricsStore { return b.buildStorageClassStore() },
	"validatingwebhookconfigurations": func(b *Builder) *metricsstore.MetricsStore { return b.buildValidatingWebhookConfigurationStore() },
	"verticalpodautoscalers":          func(b *Builder) *metricsstore.MetricsStore { return b.buildVPAStore() },
}

func collectorExists(name string) bool {
	_, ok := availableStores[name]
	return ok
}

func availableCollectors() []string {
	c := []string{}
	for name := range availableStores {
		c = append(c, name)
	}
	return c
}

func (b *Builder) buildConfigMapStore() *metricsstore.MetricsStore {
	return b.buildStore(configMapMetricFamilies, &v1.ConfigMap{}, createConfigMapListWatch)
}

func (b *Builder) buildCronJobStore() *metricsstore.MetricsStore {
	return b.buildStore(cronJobMetricFamilies, &batchv1beta1.CronJob{}, createCronJobListWatch)
}

func (b *Builder) buildDaemonSetStore() *metricsstore.MetricsStore {
	return b.buildStore(daemonSetMetricFamilies, &appsv1.DaemonSet{}, createDaemonSetListWatch)
}

func (b *Builder) buildDeploymentStore() *metricsstore.MetricsStore {
	return b.buildStore(deploymentMetricFamilies, &appsv1.Deployment{}, createDeploymentListWatch)
}

func (b *Builder) buildEndpointsStore() *metricsstore.MetricsStore {
	return b.buildStore(endpointMetricFamilies, &v1.Endpoints{}, createEndpointsListWatch)
}

func (b *Builder) buildHPAStore() *metricsstore.MetricsStore {
	return b.buildStore(hpaMetricFamilies, &autoscaling.HorizontalPodAutoscaler{}, createHPAListWatch)
}

func (b *Builder) buildIngressStore() *metricsstore.MetricsStore {
	return b.buildStore(ingressMetricFamilies, &extensions.Ingress{}, createIngressListWatch)
}

func (b *Builder) buildJobStore() *metricsstore.MetricsStore {
	return b.buildStore(jobMetricFamilies, &batchv1.Job{}, createJobListWatch)
}

func (b *Builder) buildLimitRangeStore() *metricsstore.MetricsStore {
	return b.buildStore(limitRangeMetricFamilies, &v1.LimitRange{}, createLimitRangeListWatch)
}

func (b *Builder) buildMutatingWebhookConfigurationStore() *metricsstore.MetricsStore {
	return b.buildStore(mutatingWebhookConfigurationMetricFamilies, &admissionregistration.MutatingWebhookConfiguration{}, createMutatingWebhookConfigurationListWatch)
}

func (b *Builder) buildNamespaceStore() *metricsstore.MetricsStore {
	return b.buildStore(namespaceMetricFamilies, &v1.Namespace{}, createNamespaceListWatch)
}

func (b *Builder) buildNodeStore() *metricsstore.MetricsStore {
	return b.buildStore(nodeMetricFamilies, &v1.Node{}, createNodeListWatch)
}

func (b *Builder) buildPersistentVolumeClaimStore() *metricsstore.MetricsStore {
	return b.buildStore(persistentVolumeClaimMetricFamilies, &v1.PersistentVolumeClaim{}, createPersistentVolumeClaimListWatch)
}

func (b *Builder) buildPersistentVolumeStore() *metricsstore.MetricsStore {
	return b.buildStore(persistentVolumeMetricFamilies, &v1.PersistentVolume{}, createPersistentVolumeListWatch)
}

func (b *Builder) buildPodDisruptionBudgetStore() *metricsstore.MetricsStore {
	return b.buildStore(podDisruptionBudgetMetricFamilies, &policy.PodDisruptionBudget{}, createPodDisruptionBudgetListWatch)
}

func (b *Builder) buildReplicaSetStore() *metricsstore.MetricsStore {
	return b.buildStore(replicaSetMetricFamilies, &appsv1.ReplicaSet{}, createReplicaSetListWatch)
}

func (b *Builder) buildReplicationControllerStore() *metricsstore.MetricsStore {
	return b.buildStore(replicationControllerMetricFamilies, &v1.ReplicationController{}, createReplicationControllerListWatch)
}

func (b *Builder) buildResourceQuotaStore() *metricsstore.MetricsStore {
	return b.buildStore(resourceQuotaMetricFamilies, &v1.ResourceQuota{}, createResourceQuotaListWatch)
}

func (b *Builder) buildSecretStore() *metricsstore.MetricsStore {
	return b.buildStore(secretMetricFamilies, &v1.Secret{}, createSecretListWatch)
}

func (b *Builder) buildServiceStore() *metricsstore.MetricsStore {
	return b.buildStore(serviceMetricFamilies, &v1.Service{}, createServiceListWatch)
}

func (b *Builder) buildStatefulSetStore() *metricsstore.MetricsStore {
	return b.buildStore(statefulSetMetricFamilies, &appsv1.StatefulSet{}, createStatefulSetListWatch)
}

func (b *Builder) buildStorageClassStore() *metricsstore.MetricsStore {
	return b.buildStore(storageClassMetricFamilies, &storagev1.StorageClass{}, createStorageClassListWatch)
}

func (b *Builder) buildPodStore() *metricsstore.MetricsStore {
	return b.buildStore(podMetricFamilies, &v1.Pod{}, createPodListWatch)
}

func (b *Builder) buildCsrStore() *metricsstore.MetricsStore {
	return b.buildStore(csrMetricFamilies, &certv1beta1.CertificateSigningRequest{}, createCSRListWatch)
}

func (b *Builder) buildValidatingWebhookConfigurationStore() *metricsstore.MetricsStore {
	return b.buildStore(validatingWebhookConfigurationMetricFamilies, &admissionregistration.ValidatingWebhookConfiguration{}, createValidatingWebhookConfigurationListWatch)
}

func (b *Builder) buildVPAStore() *metricsstore.MetricsStore {
	return b.buildStore(vpaMetricFamilies, &vpaautoscaling.VerticalPodAutoscaler{}, createVPAListWatchFunc(b.vpaClient))
}

func (b *Builder) buildStore(
	metricFamilies []metric.FamilyGenerator,
	expectedType interface{},
	listWatchFunc func(kubeClient clientset.Interface, ns string) cache.ListerWatcher,
) *metricsstore.MetricsStore {
	filteredMetricFamilies := metric.FilterMetricFamilies(b.whiteBlackList, metricFamilies)
	composedMetricGenFuncs := metric.ComposeMetricGenFuncs(filteredMetricFamilies)

	familyHeaders := metric.ExtractMetricFamilyHeaders(filteredMetricFamilies)

	store := metricsstore.NewMetricsStore(
		familyHeaders,
		composedMetricGenFuncs,
	)
	b.reflectorPerNamespace(expectedType, store, listWatchFunc)

	return store
}

// reflectorPerNamespace creates a Kubernetes client-go reflector with the given
// listWatchFunc for each given namespace and registers it with the given store.
func (b *Builder) reflectorPerNamespace(
	expectedType interface{},
	store cache.Store,
	listWatchFunc func(kubeClient clientset.Interface, ns string) cache.ListerWatcher,
) {
	for _, ns := range b.namespaces {
		lw := listWatchFunc(b.kubeClient, ns)
		instrumentedListWatch := watch.NewInstrumentedListerWatcher(lw, b.metrics, reflect.TypeOf(expectedType).String())
		reflector := cache.NewReflector(sharding.NewShardedListWatch(b.shard, b.totalShards, instrumentedListWatch), expectedType, store, 0)
		go reflector.Run(b.ctx.Done())
	}
}
