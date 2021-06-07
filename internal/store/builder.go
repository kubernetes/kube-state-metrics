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
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	certv1 "k8s.io/api/certificates/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policy "k8s.io/api/policy/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	vpaautoscaling "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	"k8s.io/kube-state-metrics/v2/pkg/options"
	"k8s.io/kube-state-metrics/v2/pkg/sharding"
	"k8s.io/kube-state-metrics/v2/pkg/watch"
)

// Make sure the internal Builder implements the public BuilderInterface.
// New Builder methods should be added to the public BuilderInterface.
var _ ksmtypes.BuilderInterface = &Builder{}

// Builder helps to build store. It follows the builder pattern
// (https://en.wikipedia.org/wiki/Builder_pattern).
type Builder struct {
	kubeClient       clientset.Interface
	vpaClient        vpaclientset.Interface
	namespaces       options.NamespaceList
	ctx              context.Context
	enabledResources []string
	allowDenyList    ksmtypes.AllowDenyLister
	listWatchMetrics *watch.ListWatchMetrics
	shardingMetrics  *sharding.Metrics
	shard            int32
	totalShards      int
	buildStoreFunc   ksmtypes.BuildStoreFunc
	allowLabelsList  map[string][]string
}

// NewBuilder returns a new builder.
func NewBuilder() *Builder {
	b := &Builder{}
	return b
}

// WithMetrics sets the metrics property of a Builder.
func (b *Builder) WithMetrics(r prometheus.Registerer) {
	b.listWatchMetrics = watch.NewListWatchMetrics(r)
	b.shardingMetrics = sharding.NewShardingMetrics(r)
}

// WithEnabledResources sets the enabledResources property of a Builder.
func (b *Builder) WithEnabledResources(r []string) error {
	for _, col := range r {
		if !resourceExists(col) {
			return errors.Errorf("resource %s does not exist. Available resources: %s", col, strings.Join(availableResources(), ","))
		}
	}

	var copy []string
	copy = append(copy, r...)

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
	labels := map[string]string{sharding.LabelOrdinal: strconv.Itoa(int(shard))}
	b.shardingMetrics.Ordinal.Reset()
	b.shardingMetrics.Ordinal.With(labels).Set(float64(shard))
	b.totalShards = totalShards
	b.shardingMetrics.Total.Set(float64(totalShards))
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

// WithAllowDenyList configures the allow or denylisted metric to be exposed
// by the store build by the Builder.
func (b *Builder) WithAllowDenyList(l ksmtypes.AllowDenyLister) {
	b.allowDenyList = l
}

// WithGenerateStoreFunc configures a custom generate store function
func (b *Builder) WithGenerateStoreFunc(f ksmtypes.BuildStoreFunc) {
	b.buildStoreFunc = f
}

// DefaultGenerateStoreFunc returns default buildStore function
func (b *Builder) DefaultGenerateStoreFunc() ksmtypes.BuildStoreFunc {
	return b.buildStore
}

// WithAllowLabels configures which labels can be returned for metrics
func (b *Builder) WithAllowLabels(labels map[string][]string) {
	if len(labels) > 0 {
		b.allowLabelsList = labels
	}
}

// Build initializes and registers all enabled stores.
// It returns metrics writers which can be used to write out
// metrics from the stores.
func (b *Builder) Build() []metricsstore.MetricsWriter {
	if b.allowDenyList == nil {
		panic("allowDenyList should not be nil")
	}

	var metricsWriters []metricsstore.MetricsWriter
	var activeStoreNames []string

	for _, c := range b.enabledResources {
		constructor, ok := availableStores[c]
		if ok {
			stores := constructor(b)
			activeStoreNames = append(activeStoreNames, c)
			if len(stores) == 1 {
				metricsWriters = append(metricsWriters, stores[0])
			} else {
				metricsWriters = append(metricsWriters, metricsstore.NewMultiStoreMetricsWriter(stores))
			}
		}
	}

	klog.Infof("Active resources: %s", strings.Join(activeStoreNames, ","))

	return metricsWriters
}

var availableStores = map[string]func(f *Builder) []*metricsstore.MetricsStore{
	"certificatesigningrequests":      func(b *Builder) []*metricsstore.MetricsStore { return b.buildCsrStore() },
	"configmaps":                      func(b *Builder) []*metricsstore.MetricsStore { return b.buildConfigMapStore() },
	"cronjobs":                        func(b *Builder) []*metricsstore.MetricsStore { return b.buildCronJobStore() },
	"daemonsets":                      func(b *Builder) []*metricsstore.MetricsStore { return b.buildDaemonSetStore() },
	"deployments":                     func(b *Builder) []*metricsstore.MetricsStore { return b.buildDeploymentStore() },
	"endpoints":                       func(b *Builder) []*metricsstore.MetricsStore { return b.buildEndpointsStore() },
	"horizontalpodautoscalers":        func(b *Builder) []*metricsstore.MetricsStore { return b.buildHPAStore() },
	"ingresses":                       func(b *Builder) []*metricsstore.MetricsStore { return b.buildIngressStore() },
	"jobs":                            func(b *Builder) []*metricsstore.MetricsStore { return b.buildJobStore() },
	"leases":                          func(b *Builder) []*metricsstore.MetricsStore { return b.buildLeases() },
	"limitranges":                     func(b *Builder) []*metricsstore.MetricsStore { return b.buildLimitRangeStore() },
	"mutatingwebhookconfigurations":   func(b *Builder) []*metricsstore.MetricsStore { return b.buildMutatingWebhookConfigurationStore() },
	"namespaces":                      func(b *Builder) []*metricsstore.MetricsStore { return b.buildNamespaceStore() },
	"networkpolicies":                 func(b *Builder) []*metricsstore.MetricsStore { return b.buildNetworkPolicyStore() },
	"nodes":                           func(b *Builder) []*metricsstore.MetricsStore { return b.buildNodeStore() },
	"persistentvolumeclaims":          func(b *Builder) []*metricsstore.MetricsStore { return b.buildPersistentVolumeClaimStore() },
	"persistentvolumes":               func(b *Builder) []*metricsstore.MetricsStore { return b.buildPersistentVolumeStore() },
	"poddisruptionbudgets":            func(b *Builder) []*metricsstore.MetricsStore { return b.buildPodDisruptionBudgetStore() },
	"pods":                            func(b *Builder) []*metricsstore.MetricsStore { return b.buildPodStore() },
	"replicasets":                     func(b *Builder) []*metricsstore.MetricsStore { return b.buildReplicaSetStore() },
	"replicationcontrollers":          func(b *Builder) []*metricsstore.MetricsStore { return b.buildReplicationControllerStore() },
	"resourcequotas":                  func(b *Builder) []*metricsstore.MetricsStore { return b.buildResourceQuotaStore() },
	"secrets":                         func(b *Builder) []*metricsstore.MetricsStore { return b.buildSecretStore() },
	"services":                        func(b *Builder) []*metricsstore.MetricsStore { return b.buildServiceStore() },
	"statefulsets":                    func(b *Builder) []*metricsstore.MetricsStore { return b.buildStatefulSetStore() },
	"storageclasses":                  func(b *Builder) []*metricsstore.MetricsStore { return b.buildStorageClassStore() },
	"validatingwebhookconfigurations": func(b *Builder) []*metricsstore.MetricsStore { return b.buildValidatingWebhookConfigurationStore() },
	"volumeattachments":               func(b *Builder) []*metricsstore.MetricsStore { return b.buildVolumeAttachmentStore() },
	"verticalpodautoscalers":          func(b *Builder) []*metricsstore.MetricsStore { return b.buildVPAStore() },
}

func resourceExists(name string) bool {
	_, ok := availableStores[name]
	return ok
}

func availableResources() []string {
	c := []string{}
	for name := range availableStores {
		c = append(c, name)
	}
	return c
}

func (b *Builder) buildConfigMapStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(configMapMetricFamilies, &v1.ConfigMap{}, createConfigMapListWatch)
}

func (b *Builder) buildCronJobStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(cronJobMetricFamilies(b.allowLabelsList["cronjobs"]), &batchv1beta1.CronJob{}, createCronJobListWatch)
}

func (b *Builder) buildDaemonSetStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(daemonSetMetricFamilies(b.allowLabelsList["daemonsets"]), &appsv1.DaemonSet{}, createDaemonSetListWatch)
}

func (b *Builder) buildDeploymentStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(deploymentMetricFamilies(b.allowLabelsList["deployments"]), &appsv1.Deployment{}, createDeploymentListWatch)
}

func (b *Builder) buildEndpointsStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(endpointMetricFamilies(b.allowLabelsList["endpoints"]), &v1.Endpoints{}, createEndpointsListWatch)
}

func (b *Builder) buildHPAStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(hpaMetricFamilies(b.allowLabelsList["horizontalpodautoscalers"]), &autoscaling.HorizontalPodAutoscaler{}, createHPAListWatch)
}

func (b *Builder) buildIngressStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(ingressMetricFamilies(b.allowLabelsList["ingresses"]), &networkingv1.Ingress{}, createIngressListWatch)
}

func (b *Builder) buildJobStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(jobMetricFamilies(b.allowLabelsList["jobs"]), &batchv1.Job{}, createJobListWatch)
}

func (b *Builder) buildLimitRangeStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(limitRangeMetricFamilies, &v1.LimitRange{}, createLimitRangeListWatch)
}

func (b *Builder) buildMutatingWebhookConfigurationStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(mutatingWebhookConfigurationMetricFamilies, &admissionregistrationv1.MutatingWebhookConfiguration{}, createMutatingWebhookConfigurationListWatch)
}

func (b *Builder) buildNamespaceStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(namespaceMetricFamilies(b.allowLabelsList["namespaces"]), &v1.Namespace{}, createNamespaceListWatch)
}

func (b *Builder) buildNetworkPolicyStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(networkPolicyMetricFamilies(b.allowLabelsList["networkpolicies"]), &networkingv1.NetworkPolicy{}, createNetworkPolicyListWatch)
}

func (b *Builder) buildNodeStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(nodeMetricFamilies(b.allowLabelsList["nodes"]), &v1.Node{}, createNodeListWatch)
}

func (b *Builder) buildPersistentVolumeClaimStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(persistentVolumeClaimMetricFamilies(b.allowLabelsList["persistentvolumeclaims"]), &v1.PersistentVolumeClaim{}, createPersistentVolumeClaimListWatch)
}

func (b *Builder) buildPersistentVolumeStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(persistentVolumeMetricFamilies(b.allowLabelsList["persistentvolumes"]), &v1.PersistentVolume{}, createPersistentVolumeListWatch)
}

func (b *Builder) buildPodDisruptionBudgetStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(podDisruptionBudgetMetricFamilies, &policy.PodDisruptionBudget{}, createPodDisruptionBudgetListWatch)
}

func (b *Builder) buildReplicaSetStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(replicaSetMetricFamilies(b.allowLabelsList["replicasets"]), &appsv1.ReplicaSet{}, createReplicaSetListWatch)
}

func (b *Builder) buildReplicationControllerStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(replicationControllerMetricFamilies, &v1.ReplicationController{}, createReplicationControllerListWatch)
}

func (b *Builder) buildResourceQuotaStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(resourceQuotaMetricFamilies, &v1.ResourceQuota{}, createResourceQuotaListWatch)
}

func (b *Builder) buildSecretStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(secretMetricFamilies(b.allowLabelsList["secrets"]), &v1.Secret{}, createSecretListWatch)
}

func (b *Builder) buildServiceStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(serviceMetricFamilies(b.allowLabelsList["services"]), &v1.Service{}, createServiceListWatch)
}

func (b *Builder) buildStatefulSetStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(statefulSetMetricFamilies(b.allowLabelsList["statefulsets"]), &appsv1.StatefulSet{}, createStatefulSetListWatch)
}

func (b *Builder) buildStorageClassStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(storageClassMetricFamilies(b.allowLabelsList["storageclasses"]), &storagev1.StorageClass{}, createStorageClassListWatch)
}

func (b *Builder) buildPodStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(podMetricFamilies(b.allowLabelsList["pods"]), &v1.Pod{}, createPodListWatch)
}

func (b *Builder) buildCsrStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(csrMetricFamilies(b.allowLabelsList["certificatesigningrequests"]), &certv1.CertificateSigningRequest{}, createCSRListWatch)
}

func (b *Builder) buildValidatingWebhookConfigurationStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(validatingWebhookConfigurationMetricFamilies, &admissionregistrationv1.ValidatingWebhookConfiguration{}, createValidatingWebhookConfigurationListWatch)
}

func (b *Builder) buildVolumeAttachmentStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(volumeAttachmentMetricFamilies, &storagev1.VolumeAttachment{}, createVolumeAttachmentListWatch)
}

func (b *Builder) buildVPAStore() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(vpaMetricFamilies(b.allowLabelsList["verticalpodautoscalers"]), &vpaautoscaling.VerticalPodAutoscaler{}, createVPAListWatchFunc(b.vpaClient))
}

func (b *Builder) buildLeases() []*metricsstore.MetricsStore {
	return b.buildStoreFunc(leaseMetricFamilies, &coordinationv1.Lease{}, createLeaseListWatch)
}

func (b *Builder) buildStore(
	metricFamilies []generator.FamilyGenerator,
	expectedType interface{},
	listWatchFunc func(kubeClient clientset.Interface, ns string) cache.ListerWatcher,
) []*metricsstore.MetricsStore {
	metricFamilies = generator.FilterMetricFamilies(b.allowDenyList, metricFamilies)
	composedMetricGenFuncs := generator.ComposeMetricGenFuncs(metricFamilies)
	familyHeaders := generator.ExtractMetricFamilyHeaders(metricFamilies)

	if isAllNamespaces(b.namespaces) {
		store := metricsstore.NewMetricsStore(
			familyHeaders,
			composedMetricGenFuncs,
		)
		listWatcher := listWatchFunc(b.kubeClient, v1.NamespaceAll)
		b.startReflector(expectedType, store, listWatcher)
		return []*metricsstore.MetricsStore{store}
	}

	stores := make([]*metricsstore.MetricsStore, 0, len(b.namespaces))
	for _, ns := range b.namespaces {
		store := metricsstore.NewMetricsStore(
			familyHeaders,
			composedMetricGenFuncs,
		)
		listWatcher := listWatchFunc(b.kubeClient, ns)
		b.startReflector(expectedType, store, listWatcher)
		stores = append(stores, store)
	}

	return stores
}

// startReflector starts a Kubernetes client-go reflector with the given
// listWatcher and registers it with the given store.
func (b *Builder) startReflector(
	expectedType interface{},
	store cache.Store,
	listWatcher cache.ListerWatcher,
) {
	instrumentedListWatch := watch.NewInstrumentedListerWatcher(listWatcher, b.listWatchMetrics, reflect.TypeOf(expectedType).String())
	reflector := cache.NewReflector(sharding.NewShardedListWatch(b.shard, b.totalShards, instrumentedListWatch), expectedType, store, 0)
	go reflector.Run(b.ctx.Done())
}

// isAllNamespaces checks if the given slice of namespaces
// contains only v1.NamespaceAll.
func isAllNamespaces(namespaces []string) bool {
	return len(namespaces) == 1 && namespaces[0] == v1.NamespaceAll
}
