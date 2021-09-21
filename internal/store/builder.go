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
	autoscaling "k8s.io/api/autoscaling/v2beta2"
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
	kubeClient           clientset.Interface
	vpaClient            vpaclientset.Interface
	namespaces           options.NamespaceList
	ctx                  context.Context
	enabledResources     []string
	allowDenyList        ksmtypes.AllowDenyLister
	listWatchMetrics     *watch.ListWatchMetrics
	shardingMetrics      *sharding.Metrics
	shard                int32
	totalShards          int
	buildStoresFunc      ksmtypes.BuildStoresFunc
	allowAnnotationsList map[string][]string
	allowLabelsList      map[string][]string
	useAPIServerCache    bool
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

// WithGenerateStoresFunc configures a custom generate store function
func (b *Builder) WithGenerateStoresFunc(f ksmtypes.BuildStoresFunc, u bool) {
	b.buildStoresFunc = f
	b.useAPIServerCache = u
}

// DefaultGenerateStoresFunc returns default buildStores function
func (b *Builder) DefaultGenerateStoresFunc() ksmtypes.BuildStoresFunc {
	return b.buildStores
}

// WithAllowAnnotations configures which annotations can be returned for metrics
func (b *Builder) WithAllowAnnotations(annotations map[string][]string) {
	if len(annotations) > 0 {
		b.allowAnnotationsList = annotations
	}
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
	"certificatesigningrequests":      func(b *Builder) []*metricsstore.MetricsStore { return b.buildCsrStores() },
	"configmaps":                      func(b *Builder) []*metricsstore.MetricsStore { return b.buildConfigMapStores() },
	"cronjobs":                        func(b *Builder) []*metricsstore.MetricsStore { return b.buildCronJobStores() },
	"daemonsets":                      func(b *Builder) []*metricsstore.MetricsStore { return b.buildDaemonSetStores() },
	"deployments":                     func(b *Builder) []*metricsstore.MetricsStore { return b.buildDeploymentStores() },
	"endpoints":                       func(b *Builder) []*metricsstore.MetricsStore { return b.buildEndpointsStores() },
	"horizontalpodautoscalers":        func(b *Builder) []*metricsstore.MetricsStore { return b.buildHPAStores() },
	"ingresses":                       func(b *Builder) []*metricsstore.MetricsStore { return b.buildIngressStores() },
	"jobs":                            func(b *Builder) []*metricsstore.MetricsStore { return b.buildJobStores() },
	"leases":                          func(b *Builder) []*metricsstore.MetricsStore { return b.buildLeasesStores() },
	"limitranges":                     func(b *Builder) []*metricsstore.MetricsStore { return b.buildLimitRangeStores() },
	"mutatingwebhookconfigurations":   func(b *Builder) []*metricsstore.MetricsStore { return b.buildMutatingWebhookConfigurationStores() },
	"namespaces":                      func(b *Builder) []*metricsstore.MetricsStore { return b.buildNamespaceStores() },
	"networkpolicies":                 func(b *Builder) []*metricsstore.MetricsStore { return b.buildNetworkPolicyStores() },
	"nodes":                           func(b *Builder) []*metricsstore.MetricsStore { return b.buildNodeStores() },
	"persistentvolumeclaims":          func(b *Builder) []*metricsstore.MetricsStore { return b.buildPersistentVolumeClaimStores() },
	"persistentvolumes":               func(b *Builder) []*metricsstore.MetricsStore { return b.buildPersistentVolumeStores() },
	"poddisruptionbudgets":            func(b *Builder) []*metricsstore.MetricsStore { return b.buildPodDisruptionBudgetStores() },
	"pods":                            func(b *Builder) []*metricsstore.MetricsStore { return b.buildPodStores() },
	"replicasets":                     func(b *Builder) []*metricsstore.MetricsStore { return b.buildReplicaSetStores() },
	"replicationcontrollers":          func(b *Builder) []*metricsstore.MetricsStore { return b.buildReplicationControllerStores() },
	"resourcequotas":                  func(b *Builder) []*metricsstore.MetricsStore { return b.buildResourceQuotaStores() },
	"secrets":                         func(b *Builder) []*metricsstore.MetricsStore { return b.buildSecretStores() },
	"services":                        func(b *Builder) []*metricsstore.MetricsStore { return b.buildServiceStores() },
	"statefulsets":                    func(b *Builder) []*metricsstore.MetricsStore { return b.buildStatefulSetStores() },
	"storageclasses":                  func(b *Builder) []*metricsstore.MetricsStore { return b.buildStorageClassStores() },
	"validatingwebhookconfigurations": func(b *Builder) []*metricsstore.MetricsStore { return b.buildValidatingWebhookConfigurationStores() },
	"volumeattachments":               func(b *Builder) []*metricsstore.MetricsStore { return b.buildVolumeAttachmentStores() },
	"verticalpodautoscalers":          func(b *Builder) []*metricsstore.MetricsStore { return b.buildVPAStores() },
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

func (b *Builder) buildConfigMapStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(configMapMetricFamilies(b.allowAnnotationsList["configmaps"], b.allowLabelsList["configmaps"]), &v1.ConfigMap{}, createConfigMapListWatch, b.useAPIServerCache)
}

func (b *Builder) buildCronJobStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(cronJobMetricFamilies(b.allowAnnotationsList["cronjobs"], b.allowLabelsList["cronjobs"]), &batchv1beta1.CronJob{}, createCronJobListWatch, b.useAPIServerCache)
}

func (b *Builder) buildDaemonSetStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(daemonSetMetricFamilies(b.allowAnnotationsList["daemonsets"], b.allowLabelsList["daemonsets"]), &appsv1.DaemonSet{}, createDaemonSetListWatch, b.useAPIServerCache)
}

func (b *Builder) buildDeploymentStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(deploymentMetricFamilies(b.allowAnnotationsList["deployments"], b.allowLabelsList["deployments"]), &appsv1.Deployment{}, createDeploymentListWatch, b.useAPIServerCache)
}

func (b *Builder) buildEndpointsStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(endpointMetricFamilies(b.allowAnnotationsList["endpoints"], b.allowLabelsList["endpoints"]), &v1.Endpoints{}, createEndpointsListWatch, b.useAPIServerCache)
}

func (b *Builder) buildHPAStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(hpaMetricFamilies(b.allowAnnotationsList["horizontalpodautoscalers"], b.allowLabelsList["horizontalpodautoscalers"]), &autoscaling.HorizontalPodAutoscaler{}, createHPAListWatch, b.useAPIServerCache)
}

func (b *Builder) buildIngressStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(ingressMetricFamilies(b.allowAnnotationsList["ingresses"], b.allowLabelsList["ingresses"]), &networkingv1.Ingress{}, createIngressListWatch, b.useAPIServerCache)
}

func (b *Builder) buildJobStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(jobMetricFamilies(b.allowAnnotationsList["jobs"], b.allowLabelsList["jobs"]), &batchv1.Job{}, createJobListWatch, b.useAPIServerCache)
}

func (b *Builder) buildLimitRangeStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(limitRangeMetricFamilies, &v1.LimitRange{}, createLimitRangeListWatch, b.useAPIServerCache)
}

func (b *Builder) buildMutatingWebhookConfigurationStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(mutatingWebhookConfigurationMetricFamilies, &admissionregistrationv1.MutatingWebhookConfiguration{}, createMutatingWebhookConfigurationListWatch, b.useAPIServerCache)
}

func (b *Builder) buildNamespaceStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(namespaceMetricFamilies(b.allowAnnotationsList["namespaces"], b.allowLabelsList["namespaces"]), &v1.Namespace{}, createNamespaceListWatch, b.useAPIServerCache)
}

func (b *Builder) buildNetworkPolicyStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(networkPolicyMetricFamilies(b.allowAnnotationsList["networkpolicies"], b.allowLabelsList["networkpolicies"]), &networkingv1.NetworkPolicy{}, createNetworkPolicyListWatch, b.useAPIServerCache)
}

func (b *Builder) buildNodeStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(nodeMetricFamilies(b.allowAnnotationsList["nodes"], b.allowLabelsList["nodes"]), &v1.Node{}, createNodeListWatch, b.useAPIServerCache)
}

func (b *Builder) buildPersistentVolumeClaimStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(persistentVolumeClaimMetricFamilies(b.allowAnnotationsList["persistentvolumeclaims"], b.allowLabelsList["persistentvolumeclaims"]), &v1.PersistentVolumeClaim{}, createPersistentVolumeClaimListWatch, b.useAPIServerCache)
}

func (b *Builder) buildPersistentVolumeStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(persistentVolumeMetricFamilies(b.allowAnnotationsList["persistentvolumes"], b.allowLabelsList["persistentvolumes"]), &v1.PersistentVolume{}, createPersistentVolumeListWatch, b.useAPIServerCache)
}

func (b *Builder) buildPodDisruptionBudgetStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(podDisruptionBudgetMetricFamilies, &policy.PodDisruptionBudget{}, createPodDisruptionBudgetListWatch, b.useAPIServerCache)
}

func (b *Builder) buildReplicaSetStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(replicaSetMetricFamilies(b.allowAnnotationsList["replicasets"], b.allowLabelsList["replicasets"]), &appsv1.ReplicaSet{}, createReplicaSetListWatch, b.useAPIServerCache)
}

func (b *Builder) buildReplicationControllerStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(replicationControllerMetricFamilies, &v1.ReplicationController{}, createReplicationControllerListWatch, b.useAPIServerCache)
}

func (b *Builder) buildResourceQuotaStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(resourceQuotaMetricFamilies, &v1.ResourceQuota{}, createResourceQuotaListWatch, b.useAPIServerCache)
}

func (b *Builder) buildSecretStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(secretMetricFamilies(b.allowAnnotationsList["secrets"], b.allowLabelsList["secrets"]), &v1.Secret{}, createSecretListWatch, b.useAPIServerCache)
}

func (b *Builder) buildServiceStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(serviceMetricFamilies(b.allowAnnotationsList["services"], b.allowLabelsList["services"]), &v1.Service{}, createServiceListWatch, b.useAPIServerCache)
}

func (b *Builder) buildStatefulSetStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(statefulSetMetricFamilies(b.allowAnnotationsList["statefulsets"], b.allowLabelsList["statefulsets"]), &appsv1.StatefulSet{}, createStatefulSetListWatch, b.useAPIServerCache)
}

func (b *Builder) buildStorageClassStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(storageClassMetricFamilies(b.allowAnnotationsList["storageclasses"], b.allowLabelsList["storageclasses"]), &storagev1.StorageClass{}, createStorageClassListWatch, b.useAPIServerCache)
}

func (b *Builder) buildPodStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(podMetricFamilies(b.allowAnnotationsList["pods"], b.allowLabelsList["pods"]), &v1.Pod{}, createPodListWatch, b.useAPIServerCache)
}

func (b *Builder) buildCsrStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(csrMetricFamilies(b.allowAnnotationsList["certificatesigningrequests"], b.allowLabelsList["certificatesigningrequests"]), &certv1.CertificateSigningRequest{}, createCSRListWatch, b.useAPIServerCache)
}

func (b *Builder) buildValidatingWebhookConfigurationStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(validatingWebhookConfigurationMetricFamilies, &admissionregistrationv1.ValidatingWebhookConfiguration{}, createValidatingWebhookConfigurationListWatch, b.useAPIServerCache)
}

func (b *Builder) buildVolumeAttachmentStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(volumeAttachmentMetricFamilies, &storagev1.VolumeAttachment{}, createVolumeAttachmentListWatch, b.useAPIServerCache)
}

func (b *Builder) buildVPAStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(vpaMetricFamilies(b.allowAnnotationsList["verticalpodautoscalers"], b.allowLabelsList["verticalpodautoscalers"]), &vpaautoscaling.VerticalPodAutoscaler{}, createVPAListWatchFunc(b.vpaClient), b.useAPIServerCache)
}

func (b *Builder) buildLeasesStores() []*metricsstore.MetricsStore {
	return b.buildStoresFunc(leaseMetricFamilies, &coordinationv1.Lease{}, createLeaseListWatch, b.useAPIServerCache)
}

func (b *Builder) buildStores(
	metricFamilies []generator.FamilyGenerator,
	expectedType interface{},
	listWatchFunc func(kubeClient clientset.Interface, ns string) cache.ListerWatcher,
	useAPIServerCache bool,
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
		b.startReflector(expectedType, store, listWatcher, useAPIServerCache)
		return []*metricsstore.MetricsStore{store}
	}

	stores := make([]*metricsstore.MetricsStore, 0, len(b.namespaces))
	for _, ns := range b.namespaces {
		store := metricsstore.NewMetricsStore(
			familyHeaders,
			composedMetricGenFuncs,
		)
		listWatcher := listWatchFunc(b.kubeClient, ns)
		b.startReflector(expectedType, store, listWatcher, useAPIServerCache)
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
	useAPIServerCache bool,
) {
	instrumentedListWatch := watch.NewInstrumentedListerWatcher(listWatcher, b.listWatchMetrics, reflect.TypeOf(expectedType).String(), useAPIServerCache)
	reflector := cache.NewReflector(sharding.NewShardedListWatch(b.shard, b.totalShards, instrumentedListWatch), expectedType, store, 0)
	go reflector.Run(b.ctx.Done())
}

// isAllNamespaces checks if the given slice of namespaces
// contains only v1.NamespaceAll.
func isAllNamespaces(namespaces []string) bool {
	return len(namespaces) == 1 && namespaces[0] == v1.NamespaceAll
}
