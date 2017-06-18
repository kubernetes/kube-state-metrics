package collectors

import (
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/pkg/apis/apps/v1beta1"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"golang.org/x/net/context"
	"k8s.io/client-go/pkg/api"
)

var (
	descStatefulSetStatusReplicas = prometheus.NewDesc(
		"kube_statefulset_status_replicas",
		"The number of replicas per StatefulSet.",
		[]string{"namespace", "statefulset"}, nil,
	)

	descStatefulSetStatusObservedGeneration = prometheus.NewDesc(
		"kube_statefulset_status_observed_generation",
		"The generation observed by the StatefulSet controller.",
		[]string{"namespace", "statefulset"}, nil,
	)

	descStatefulSetSpecReplicas = prometheus.NewDesc(
		"kube_statefulset_replicas",
		"Number of desired pods for a StatefulSet.",
		[]string{"namespace", "statefulset"}, nil,
	)

	descStatefulSetMetadataGeneration = prometheus.NewDesc(
		"kube_statefulset_metadata_generation",
		"Sequence number representing a specific generation of the desired state for the StatefulSet.",
		[]string{"namespace", "statefulset"}, nil,
	)
)

type StatefulSetLister func() ([]v1beta1.StatefulSet, error)

func (l StatefulSetLister) List() ([]v1beta1.StatefulSet, error) {
	return l()
}

func RegisterStatefulSetCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface) {
	client := kubeClient.AppsV1beta1().RESTClient()
	dlw := cache.NewListWatchFromClient(client, "statefulsets", api.NamespaceAll, nil)
	dinf := cache.NewSharedInformer(dlw, &v1beta1.StatefulSet{}, resyncPeriod)

	statefulSetLister := StatefulSetLister(func() (statefulSets []v1beta1.StatefulSet, err error) {
		for _, c := range dinf.GetStore().List() {
			statefulSets = append(statefulSets, *(c.(*v1beta1.StatefulSet)))
		}
		return statefulSets, nil
	})

	registry.MustRegister(&statefulSetCollector{store: statefulSetLister})
	go dinf.Run(context.Background().Done())
}

type statefulSetStore interface {
	List() (statefulSets []v1beta1.StatefulSet, err error)
}

type statefulSetCollector struct {
	store statefulSetStore
}

// Describe implements the prometheus.Collector interface.
func (dc *statefulSetCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descStatefulSetStatusReplicas
	ch <- descStatefulSetStatusObservedGeneration
	ch <- descStatefulSetSpecReplicas
	ch <- descStatefulSetMetadataGeneration
}

// Collect implements the prometheus.Collector interface.
func (sc *statefulSetCollector) Collect(ch chan<- prometheus.Metric) {
	dpls, err := sc.store.List()
	if err != nil {
		glog.Errorf("listing statefulsets failed: %s", err)
		return
	}
	for _, d := range dpls {
		sc.collectStatefulSet(ch, d)
	}
}

func (dc *statefulSetCollector) collectStatefulSet(ch chan<- prometheus.Metric, statefulSet v1beta1.StatefulSet) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{statefulSet.Namespace, statefulSet.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	addGauge(descStatefulSetStatusReplicas, float64(statefulSet.Status.Replicas))
	addGauge(descStatefulSetStatusObservedGeneration, float64(*statefulSet.Status.ObservedGeneration))
	addGauge(descStatefulSetSpecReplicas, float64(*statefulSet.Spec.Replicas))
	addGauge(descStatefulSetMetadataGeneration, float64(statefulSet.ObjectMeta.Generation))
}