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
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	certv1 "k8s.io/api/certificates/v1"
	coordinationv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	"k8s.io/kube-state-metrics/v2/pkg/options"
	"k8s.io/kube-state-metrics/v2/pkg/sharding"
	"k8s.io/kube-state-metrics/v2/pkg/util"
	"k8s.io/kube-state-metrics/v2/pkg/watch"
)

// ResourceDiscoveryTimeout is the timeout for the resource discovery.
const ResourceDiscoveryTimeout = 10 * time.Second

// ResourceDiscoveryInterval is the interval for the resource discovery.
const ResourceDiscoveryInterval = 100 * time.Millisecond

// Make sure the internal Builder implements the public BuilderInterface.
// New Builder methods should be added to the public BuilderInterface.
var _ ksmtypes.BuilderInterface = &Builder{}

// Builder helps to build store. It follows the builder pattern
// (https://en.wikipedia.org/wiki/Builder_pattern).
type Builder struct {
	kubeClient                    clientset.Interface
	ctx                           context.Context
	familyGeneratorFilter         generator.FamilyGeneratorFilter
	customResourceClients         map[string]interface{}
	listWatchMetrics              *watch.ListWatchMetrics
	shardingMetrics               *sharding.Metrics
	buildStoresFunc               ksmtypes.BuildStoresFunc
	buildCustomResourceStoresFunc ksmtypes.BuildCustomResourceStoresFunc
	allowAnnotationsList          map[string][]string
	allowLabelsList               map[string][]string
	utilOptions                   *options.Options
	// namespaceFilter is inside fieldSelectorFilter
	fieldSelectorFilter string
	namespaces          options.NamespaceList
	enabledResources    []string
	totalShards         int
	shard               int32
	useAPIServerCache   bool
	objectLimit         int64

	GVKToReflectorStopChanMap *map[string]chan struct{}
}

// NewBuilder returns a new builder.
func NewBuilder() *Builder {
	b := &Builder{}
	return b
}

// WithUtilOptions sets the utilOptions property of a Builder.
// These are the options that are necessary for the util package.
func (b *Builder) WithUtilOptions(opts *options.Options) {
	b.utilOptions = options.NewOptions()
	b.utilOptions.Apiserver = opts.Apiserver
	b.utilOptions.Kubeconfig = opts.Kubeconfig
}

// WithMetrics sets the metrics property of a Builder.
func (b *Builder) WithMetrics(r prometheus.Registerer) {
	b.listWatchMetrics = watch.NewListWatchMetrics(r)
	b.shardingMetrics = sharding.NewShardingMetrics(r)
}

// WithEnabledResources sets the enabledResources property of a Builder.
func (b *Builder) WithEnabledResources(r []string) error {
	for _, resource := range r {
		if !resourceExists(resource) {
			return fmt.Errorf("resource %s does not exist. Available resources: %s", resource, strings.Join(availableResources(), ","))
		}
	}

	b.enabledResources = append(b.enabledResources, r...)
	slices.Sort(b.enabledResources)
	b.enabledResources = slices.Compact(b.enabledResources)

	return nil
}

// WithFieldSelectorFilter sets the fieldSelector property of a Builder.
func (b *Builder) WithFieldSelectorFilter(fieldSelectorFilter string) {
	b.fieldSelectorFilter = fieldSelectorFilter
}

// WithNamespaces sets the namespaces property of a Builder.
func (b *Builder) WithNamespaces(n options.NamespaceList) {
	b.namespaces = n
}

// MergeFieldSelectors merges multiple fieldSelectors using AND operator.
func (b *Builder) MergeFieldSelectors(selectors []string) (string, error) {
	return options.MergeFieldSelectors(selectors)
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

// WithCustomResourceClients sets the customResourceClients property of a Builder.
func (b *Builder) WithCustomResourceClients(cs map[string]interface{}) {
	b.customResourceClients = cs
}

// WithUsingAPIServerCache configures whether using APIServer cache or not.
func (b *Builder) WithUsingAPIServerCache(u bool) {
	b.useAPIServerCache = u
}

// WithObjectLimit sets a limit on how many objects you can list from the APIServer.
// This is to protect kube-state-metrics from running out of memory if the APIServer has a lot of objects.
func (b *Builder) WithObjectLimit(l int64) {
	b.objectLimit = l
}

// WithFamilyGeneratorFilter configures the family generator filter which decides which
// metrics are to be exposed by the store build by the Builder.
func (b *Builder) WithFamilyGeneratorFilter(l generator.FamilyGeneratorFilter) {
	b.familyGeneratorFilter = l
}

// WithGenerateStoresFunc configures a custom generate store function
func (b *Builder) WithGenerateStoresFunc(f ksmtypes.BuildStoresFunc) {
	b.buildStoresFunc = f
}

// WithGenerateCustomResourceStoresFunc configures a custom generate custom resource store function
func (b *Builder) WithGenerateCustomResourceStoresFunc(f ksmtypes.BuildCustomResourceStoresFunc) {
	b.buildCustomResourceStoresFunc = f
}

// DefaultGenerateStoresFunc returns default buildStores function
func (b *Builder) DefaultGenerateStoresFunc() ksmtypes.BuildStoresFunc {
	return b.buildStores
}

// DefaultGenerateCustomResourceStoresFunc returns default buildCustomResourceStores function
func (b *Builder) DefaultGenerateCustomResourceStoresFunc() ksmtypes.BuildCustomResourceStoresFunc {
	return b.buildCustomResourceStores
}

// WithCustomResourceStoreFactories returns configures a custom resource stores factory
func (b *Builder) WithCustomResourceStoreFactories(fs ...customresource.RegistryFactory) {
	for i := range fs {
		f := fs[i]
		gvr, err := util.GVRFromType(f.Name(), f.ExpectedType())
		if err != nil {
			klog.ErrorS(err, "Failed to get GVR from type", "resourceName", f.Name(), "expectedType", f.ExpectedType())
		}
		var gvrString string
		if gvr != nil {
			gvrString = gvr.String()
		} else {
			gvrString = f.Name()
		}
		if _, ok := availableStores[gvrString]; ok {
			klog.InfoS("Updating store", "GVR", gvrString)
		}
		availableStores[gvrString] = func(b *Builder) []cache.Store {
			return b.buildCustomResourceStoresFunc(
				f.Name(),
				f.MetricFamilyGenerators(),
				f.ExpectedType(),
				f.ListWatch,
				b.useAPIServerCache,
				b.objectLimit,
			)
		}
	}
}

// allowList validates the given map and checks if the resources exists.
// If there is a '*' as key, return new map with all enabled resources.
func (b *Builder) allowList(list map[string][]string) (map[string][]string, error) {
	if len(list) == 0 {
		return nil, nil
	}

	for l := range list {
		if !resourceExists(l) && l != "*" {
			return nil, fmt.Errorf("resource %s does not exist. Available resources: %s", l, strings.Join(availableResources(), ","))
		}
	}

	// "*" takes precedence over other specifications
	allowedList, ok := list["*"]
	if !ok {
		return list, nil
	}
	m := make(map[string][]string)
	for _, resource := range b.enabledResources {
		m[resource] = allowedList
	}
	return m, nil
}

// WithAllowAnnotations configures which annotations can be returned for metrics
func (b *Builder) WithAllowAnnotations(annotations map[string][]string) error {
	var err error
	b.allowAnnotationsList, err = b.allowList(annotations)
	return err
}

// WithAllowLabels configures which labels can be returned for metrics
func (b *Builder) WithAllowLabels(labels map[string][]string) error {
	var err error
	b.allowLabelsList, err = b.allowList(labels)
	return err
}

// Build initializes and registers all enabled stores.
// It returns metrics writers which can be used to write out
// metrics from the stores.
func (b *Builder) Build() metricsstore.MetricsWriterList {
	if b.familyGeneratorFilter == nil {
		panic("familyGeneratorFilter should not be nil")
	}

	var metricsWriters metricsstore.MetricsWriterList
	var activeStoreNames []string

	for _, c := range b.enabledResources {
		constructor, ok := availableStores[c]
		if ok {
			stores := cacheStoresToMetricStores(constructor(b))
			activeStoreNames = append(activeStoreNames, c)
			metricsWriters = append(metricsWriters, metricsstore.NewMetricsWriter(stores...))
		}
	}

	if len(activeStoreNames) > 0 {
		klog.InfoS("Active resources", "activeStoreNames", strings.Join(activeStoreNames, ","))
	}

	return metricsWriters
}

// BuildStores initializes and registers all enabled stores.
// It returns metric stores which can be used to consume
// the generated metrics from the stores.
func (b *Builder) BuildStores() [][]cache.Store {
	if b.familyGeneratorFilter == nil {
		panic("familyGeneratorFilter should not be nil")
	}

	var allStores [][]cache.Store
	var activeStoreNames []string

	for _, c := range b.enabledResources {
		constructor, ok := availableStores[c]
		if ok {
			stores := constructor(b)
			activeStoreNames = append(activeStoreNames, c)
			allStores = append(allStores, stores)
		}
	}

	klog.InfoS("Active resources", "activeStoreNames", strings.Join(activeStoreNames, ","))

	return allStores
}

var availableStores = map[string]func(f *Builder) []cache.Store{
	"certificatesigningrequests":      func(b *Builder) []cache.Store { return b.buildCsrStores() },
	"clusterroles":                    func(b *Builder) []cache.Store { return b.buildClusterRoleStores() },
	"configmaps":                      func(b *Builder) []cache.Store { return b.buildConfigMapStores() },
	"clusterrolebindings":             func(b *Builder) []cache.Store { return b.buildClusterRoleBindingStores() },
	"cronjobs":                        func(b *Builder) []cache.Store { return b.buildCronJobStores() },
	"daemonsets":                      func(b *Builder) []cache.Store { return b.buildDaemonSetStores() },
	"deployments":                     func(b *Builder) []cache.Store { return b.buildDeploymentStores() },
	"endpoints":                       func(b *Builder) []cache.Store { return b.buildEndpointsStores() },
	"endpointslices":                  func(b *Builder) []cache.Store { return b.buildEndpointSlicesStores() },
	"horizontalpodautoscalers":        func(b *Builder) []cache.Store { return b.buildHPAStores() },
	"ingresses":                       func(b *Builder) []cache.Store { return b.buildIngressStores() },
	"ingressclasses":                  func(b *Builder) []cache.Store { return b.buildIngressClassStores() },
	"jobs":                            func(b *Builder) []cache.Store { return b.buildJobStores() },
	"leases":                          func(b *Builder) []cache.Store { return b.buildLeasesStores() },
	"limitranges":                     func(b *Builder) []cache.Store { return b.buildLimitRangeStores() },
	"mutatingwebhookconfigurations":   func(b *Builder) []cache.Store { return b.buildMutatingWebhookConfigurationStores() },
	"namespaces":                      func(b *Builder) []cache.Store { return b.buildNamespaceStores() },
	"networkpolicies":                 func(b *Builder) []cache.Store { return b.buildNetworkPolicyStores() },
	"nodes":                           func(b *Builder) []cache.Store { return b.buildNodeStores() },
	"persistentvolumeclaims":          func(b *Builder) []cache.Store { return b.buildPersistentVolumeClaimStores() },
	"persistentvolumes":               func(b *Builder) []cache.Store { return b.buildPersistentVolumeStores() },
	"poddisruptionbudgets":            func(b *Builder) []cache.Store { return b.buildPodDisruptionBudgetStores() },
	"pods":                            func(b *Builder) []cache.Store { return b.buildPodStores() },
	"replicasets":                     func(b *Builder) []cache.Store { return b.buildReplicaSetStores() },
	"replicationcontrollers":          func(b *Builder) []cache.Store { return b.buildReplicationControllerStores() },
	"resourcequotas":                  func(b *Builder) []cache.Store { return b.buildResourceQuotaStores() },
	"roles":                           func(b *Builder) []cache.Store { return b.buildRoleStores() },
	"rolebindings":                    func(b *Builder) []cache.Store { return b.buildRoleBindingStores() },
	"secrets":                         func(b *Builder) []cache.Store { return b.buildSecretStores() },
	"serviceaccounts":                 func(b *Builder) []cache.Store { return b.buildServiceAccountStores() },
	"services":                        func(b *Builder) []cache.Store { return b.buildServiceStores() },
	"statefulsets":                    func(b *Builder) []cache.Store { return b.buildStatefulSetStores() },
	"storageclasses":                  func(b *Builder) []cache.Store { return b.buildStorageClassStores() },
	"validatingwebhookconfigurations": func(b *Builder) []cache.Store { return b.buildValidatingWebhookConfigurationStores() },
	"volumeattachments":               func(b *Builder) []cache.Store { return b.buildVolumeAttachmentStores() },
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

func (b *Builder) buildConfigMapStores() []cache.Store {
	return b.buildStoresFunc(configMapMetricFamilies(b.allowAnnotationsList["configmaps"], b.allowLabelsList["configmaps"]), &v1.ConfigMap{}, createConfigMapListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildCronJobStores() []cache.Store {
	return b.buildStoresFunc(cronJobMetricFamilies(b.allowAnnotationsList["cronjobs"], b.allowLabelsList["cronjobs"]), &batchv1.CronJob{}, createCronJobListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildDaemonSetStores() []cache.Store {
	return b.buildStoresFunc(daemonSetMetricFamilies(b.allowAnnotationsList["daemonsets"], b.allowLabelsList["daemonsets"]), &appsv1.DaemonSet{}, createDaemonSetListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildDeploymentStores() []cache.Store {
	return b.buildStoresFunc(deploymentMetricFamilies(b.allowAnnotationsList["deployments"], b.allowLabelsList["deployments"]), &appsv1.Deployment{}, createDeploymentListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildEndpointsStores() []cache.Store {
	return b.buildStoresFunc(endpointMetricFamilies(b.allowAnnotationsList["endpoints"], b.allowLabelsList["endpoints"]), &v1.Endpoints{}, createEndpointsListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildEndpointSlicesStores() []cache.Store {
	return b.buildStoresFunc(endpointSliceMetricFamilies(b.allowAnnotationsList["endpointslices"], b.allowLabelsList["endpointslices"]), &discoveryv1.EndpointSlice{}, createEndpointSliceListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildHPAStores() []cache.Store {
	return b.buildStoresFunc(hpaMetricFamilies(b.allowAnnotationsList["horizontalpodautoscalers"], b.allowLabelsList["horizontalpodautoscalers"]), &autoscaling.HorizontalPodAutoscaler{}, createHPAListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildIngressStores() []cache.Store {
	return b.buildStoresFunc(ingressMetricFamilies(b.allowAnnotationsList["ingresses"], b.allowLabelsList["ingresses"]), &networkingv1.Ingress{}, createIngressListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildJobStores() []cache.Store {
	return b.buildStoresFunc(jobMetricFamilies(b.allowAnnotationsList["jobs"], b.allowLabelsList["jobs"]), &batchv1.Job{}, createJobListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildLimitRangeStores() []cache.Store {
	return b.buildStoresFunc(limitRangeMetricFamilies, &v1.LimitRange{}, createLimitRangeListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildMutatingWebhookConfigurationStores() []cache.Store {
	return b.buildStoresFunc(mutatingWebhookConfigurationMetricFamilies, &admissionregistrationv1.MutatingWebhookConfiguration{}, createMutatingWebhookConfigurationListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildNamespaceStores() []cache.Store {
	return b.buildStoresFunc(namespaceMetricFamilies(b.allowAnnotationsList["namespaces"], b.allowLabelsList["namespaces"]), &v1.Namespace{}, createNamespaceListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildNetworkPolicyStores() []cache.Store {
	return b.buildStoresFunc(networkPolicyMetricFamilies(b.allowAnnotationsList["networkpolicies"], b.allowLabelsList["networkpolicies"]), &networkingv1.NetworkPolicy{}, createNetworkPolicyListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildNodeStores() []cache.Store {
	return b.buildStoresFunc(nodeMetricFamilies(b.allowAnnotationsList["nodes"], b.allowLabelsList["nodes"]), &v1.Node{}, createNodeListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildPersistentVolumeClaimStores() []cache.Store {
	return b.buildStoresFunc(persistentVolumeClaimMetricFamilies(b.allowAnnotationsList["persistentvolumeclaims"], b.allowLabelsList["persistentvolumeclaims"]), &v1.PersistentVolumeClaim{}, createPersistentVolumeClaimListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildPersistentVolumeStores() []cache.Store {
	return b.buildStoresFunc(persistentVolumeMetricFamilies(b.allowAnnotationsList["persistentvolumes"], b.allowLabelsList["persistentvolumes"]), &v1.PersistentVolume{}, createPersistentVolumeListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildPodDisruptionBudgetStores() []cache.Store {
	return b.buildStoresFunc(podDisruptionBudgetMetricFamilies(b.allowAnnotationsList["poddisruptionbudgets"], b.allowLabelsList["poddisruptionbudgets"]), &policyv1.PodDisruptionBudget{}, createPodDisruptionBudgetListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildReplicaSetStores() []cache.Store {
	return b.buildStoresFunc(replicaSetMetricFamilies(b.allowAnnotationsList["replicasets"], b.allowLabelsList["replicasets"]), &appsv1.ReplicaSet{}, createReplicaSetListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildReplicationControllerStores() []cache.Store {
	return b.buildStoresFunc(replicationControllerMetricFamilies, &v1.ReplicationController{}, createReplicationControllerListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildResourceQuotaStores() []cache.Store {
	return b.buildStoresFunc(resourceQuotaMetricFamilies(b.allowAnnotationsList["resourcequotas"], b.allowLabelsList["resourcequotas"]), &v1.ResourceQuota{}, createResourceQuotaListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildSecretStores() []cache.Store {
	return b.buildStoresFunc(secretMetricFamilies(b.allowAnnotationsList["secrets"], b.allowLabelsList["secrets"]), &v1.Secret{}, createSecretListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildServiceAccountStores() []cache.Store {
	return b.buildStoresFunc(serviceAccountMetricFamilies(b.allowAnnotationsList["serviceaccounts"], b.allowLabelsList["serviceaccounts"]), &v1.ServiceAccount{}, createServiceAccountListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildServiceStores() []cache.Store {
	return b.buildStoresFunc(serviceMetricFamilies(b.allowAnnotationsList["services"], b.allowLabelsList["services"]), &v1.Service{}, createServiceListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildStatefulSetStores() []cache.Store {
	return b.buildStoresFunc(statefulSetMetricFamilies(b.allowAnnotationsList["statefulsets"], b.allowLabelsList["statefulsets"]), &appsv1.StatefulSet{}, createStatefulSetListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildStorageClassStores() []cache.Store {
	return b.buildStoresFunc(storageClassMetricFamilies(b.allowAnnotationsList["storageclasses"], b.allowLabelsList["storageclasses"]), &storagev1.StorageClass{}, createStorageClassListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildPodStores() []cache.Store {
	return b.buildStoresFunc(podMetricFamilies(b.allowAnnotationsList["pods"], b.allowLabelsList["pods"]), &v1.Pod{}, createPodListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildCsrStores() []cache.Store {
	return b.buildStoresFunc(csrMetricFamilies(b.allowAnnotationsList["certificatesigningrequests"], b.allowLabelsList["certificatesigningrequests"]), &certv1.CertificateSigningRequest{}, createCSRListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildValidatingWebhookConfigurationStores() []cache.Store {
	return b.buildStoresFunc(validatingWebhookConfigurationMetricFamilies, &admissionregistrationv1.ValidatingWebhookConfiguration{}, createValidatingWebhookConfigurationListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildVolumeAttachmentStores() []cache.Store {
	return b.buildStoresFunc(volumeAttachmentMetricFamilies, &storagev1.VolumeAttachment{}, createVolumeAttachmentListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildLeasesStores() []cache.Store {
	return b.buildStoresFunc(leaseMetricFamilies, &coordinationv1.Lease{}, createLeaseListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildClusterRoleStores() []cache.Store {
	return b.buildStoresFunc(clusterRoleMetricFamilies(b.allowAnnotationsList["clusterroles"], b.allowLabelsList["clusterroles"]), &rbacv1.ClusterRole{}, createClusterRoleListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildRoleStores() []cache.Store {
	return b.buildStoresFunc(roleMetricFamilies(b.allowAnnotationsList["roles"], b.allowLabelsList["roles"]), &rbacv1.Role{}, createRoleListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildClusterRoleBindingStores() []cache.Store {
	return b.buildStoresFunc(clusterRoleBindingMetricFamilies(b.allowAnnotationsList["clusterrolebindings"], b.allowLabelsList["clusterrolebindings"]), &rbacv1.ClusterRoleBinding{}, createClusterRoleBindingListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildRoleBindingStores() []cache.Store {
	return b.buildStoresFunc(roleBindingMetricFamilies(b.allowAnnotationsList["rolebindings"], b.allowLabelsList["rolebindings"]), &rbacv1.RoleBinding{}, createRoleBindingListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildIngressClassStores() []cache.Store {
	return b.buildStoresFunc(ingressClassMetricFamilies(b.allowAnnotationsList["ingressclasses"], b.allowLabelsList["ingressclasses"]), &networkingv1.IngressClass{}, createIngressClassListWatch, b.useAPIServerCache, b.objectLimit)
}

func (b *Builder) buildStores(
	metricFamilies []generator.FamilyGenerator,
	expectedType interface{},
	listWatchFunc func(kubeClient clientset.Interface, ns string, fieldSelector string) cache.ListerWatcher,
	useAPIServerCache bool, objectLimit int64,
) []cache.Store {
	metricFamilies = generator.FilterFamilyGenerators(b.familyGeneratorFilter, metricFamilies)
	composedMetricGenFuncs := generator.ComposeMetricGenFuncs(metricFamilies)
	familyHeaders := generator.ExtractMetricFamilyHeaders(metricFamilies)

	if b.namespaces.IsAllNamespaces() {
		store := metricsstore.NewMetricsStore(
			familyHeaders,
			composedMetricGenFuncs,
		)
		if b.fieldSelectorFilter != "" {
			klog.InfoS("FieldSelector is used", "fieldSelector", b.fieldSelectorFilter)
		}
		listWatcher := listWatchFunc(b.kubeClient, v1.NamespaceAll, b.fieldSelectorFilter)
		b.startReflector(expectedType, store, listWatcher, useAPIServerCache, objectLimit)
		return []cache.Store{store}
	}

	stores := make([]cache.Store, 0, len(b.namespaces))
	for _, ns := range b.namespaces {
		store := metricsstore.NewMetricsStore(
			familyHeaders,
			composedMetricGenFuncs,
		)
		if b.fieldSelectorFilter != "" {
			klog.InfoS("FieldSelector is used", "fieldSelector", b.fieldSelectorFilter)
		}
		listWatcher := listWatchFunc(b.kubeClient, ns, b.fieldSelectorFilter)
		b.startReflector(expectedType, store, listWatcher, useAPIServerCache, objectLimit)
		stores = append(stores, store)
	}

	return stores
}

// TODO(Garrybest): Merge `buildStores` and `buildCustomResourceStores`
func (b *Builder) buildCustomResourceStores(resourceName string,
	metricFamilies []generator.FamilyGenerator,
	expectedType interface{},
	listWatchFunc func(customResourceClient interface{}, ns string, fieldSelector string) cache.ListerWatcher,
	useAPIServerCache bool, objectLimit int64,
) []cache.Store {
	metricFamilies = generator.FilterFamilyGenerators(b.familyGeneratorFilter, metricFamilies)
	composedMetricGenFuncs := generator.ComposeMetricGenFuncs(metricFamilies)

	familyHeaders := generator.ExtractMetricFamilyHeaders(metricFamilies)

	gvr, err := util.GVRFromType(resourceName, expectedType)
	if err != nil {
		klog.ErrorS(err, "Failed to get GVR from type", "resourceName", resourceName, "expectedType", expectedType)
	}
	var gvrString string
	if gvr != nil {
		gvrString = gvr.String()
	} else {
		gvrString = resourceName
	}
	customResourceClient, ok := b.customResourceClients[gvrString]
	if !ok {
		klog.InfoS("Custom resource client does not exist", "resourceName", resourceName)
		return []cache.Store{}
	}

	if b.namespaces.IsAllNamespaces() {
		store := metricsstore.NewMetricsStore(
			familyHeaders,
			composedMetricGenFuncs,
		)
		if b.fieldSelectorFilter != "" {
			klog.InfoS("FieldSelector is used", "fieldSelector", b.fieldSelectorFilter)
		}
		listWatcher := listWatchFunc(customResourceClient, v1.NamespaceAll, b.fieldSelectorFilter)
		b.startReflector(expectedType, store, listWatcher, useAPIServerCache, objectLimit)
		return []cache.Store{store}
	}

	stores := make([]cache.Store, 0, len(b.namespaces))
	for _, ns := range b.namespaces {
		store := metricsstore.NewMetricsStore(
			familyHeaders,
			composedMetricGenFuncs,
		)
		klog.InfoS("FieldSelector is used", "fieldSelector", b.fieldSelectorFilter)
		listWatcher := listWatchFunc(customResourceClient, ns, b.fieldSelectorFilter)
		b.startReflector(expectedType, store, listWatcher, useAPIServerCache, objectLimit)
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
	objectLimit int64,
) {
	instrumentedListWatch := watch.NewInstrumentedListerWatcher(listWatcher, b.listWatchMetrics, reflect.TypeOf(expectedType).String(), useAPIServerCache, objectLimit)
	reflector := cache.NewReflectorWithOptions(sharding.NewShardedListWatch(b.shard, b.totalShards, instrumentedListWatch), expectedType, store, cache.ReflectorOptions{ResyncPeriod: 0})
	if cr, ok := expectedType.(*unstructured.Unstructured); ok {
		go reflector.Run((*b.GVKToReflectorStopChanMap)[cr.GroupVersionKind().String()])
	} else {
		go reflector.Run(b.ctx.Done())
	}
}

// cacheStoresToMetricStores converts []cache.Store into []*metricsstore.MetricsStore
func cacheStoresToMetricStores(cStores []cache.Store) []*metricsstore.MetricsStore {
	mStores := make([]*metricsstore.MetricsStore, 0, len(cStores))
	for _, store := range cStores {
		mStores = append(mStores, store.(*metricsstore.MetricsStore))
	}

	return mStores
}
