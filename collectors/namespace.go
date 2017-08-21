package collectors

import (
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

var (
	descNamespaceLabelsName          = "kube_namespace_labels"
	descNamespaceLabelsHelp          = "Kubernetes labels converted to Prometheus labels."
	descNamespaceLabelsDefaultLabels = []string{"namespace"}

	descNamespaceCreated = prometheus.NewDesc(
		"kube_namespace_created",
		"Unix creation timestamp",
		[]string{"namespace"}, nil,
	)

	descNamespaceLabels = prometheus.NewDesc(
		descNamespaceLabelsName,
		descNamespaceLabelsHelp,
		descNamespaceLabelsDefaultLabels, nil,
	)
)

type NamespaceLister func() ([]v1.Namespace, error)

func (l NamespaceLister) List() ([]v1.Namespace, error) {
	return l()
}

func RegisterNamespaceCollector(registry prometheus.Registerer, kubeClient kubernetes.Interface, namespace string) {
	client := kubeClient.CoreV1().RESTClient()
	var selector fields.Selector
	if namespace != api.NamespaceAll {
		selector = fields.OneTermEqualSelector("metadata.name", namespace)
	}
	nlw := cache.NewListWatchFromClient(client, "namespaces", api.NamespaceAll, selector)
	ninf := cache.NewSharedInformer(nlw, &v1.Namespace{}, resyncPeriod)

	namespaceLister := NamespaceLister(func() (namespaces []v1.Namespace, err error) {
		for _, n := range ninf.GetStore().List() {
			namespaces = append(namespaces, *(n.(*v1.Namespace)))
		}
		return namespaces, nil
	})

	registry.MustRegister(&namespaceCollector{store: namespaceLister})
	go ninf.Run(context.Background().Done())
}

type namespaceStore interface {
	List() (namespaces []v1.Namespace, err error)
}

type namespaceCollector struct {
	store namespaceStore
}

// Describe implements the prometheus.Collector interface.
func (nc *namespaceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descNamespaceCreated
	ch <- descNamespaceLabels
}

// Collect implements the prometheus.Collector interface.
func (nc *namespaceCollector) Collect(ch chan<- prometheus.Metric) {
	ns, err := nc.store.List()
	if err != nil {
		glog.Errorf("listing namespaces failed: %s", err)
		return
	}
	for _, n := range ns {
		nc.collectNamespace(ch, n)
	}
}

func namespaceLabelsDesc(labelKeys []string) *prometheus.Desc {
	return prometheus.NewDesc(
		descNamespaceLabelsName,
		descNamespaceLabelsHelp,
		append(descNamespaceLabelsDefaultLabels, labelKeys...),
		nil,
	)
}

func (nc *namespaceCollector) collectNamespace(ch chan<- prometheus.Metric, namespace v1.Namespace) {
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		lv = append([]string{namespace.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	if !namespace.CreationTimestamp.IsZero() {
		addGauge(descNamespaceCreated, float64(namespace.CreationTimestamp.Unix()))
	}

	labelKeys, labelValues := kubeLabelsToPrometheusLabels(namespace.Labels)
	addGauge(namespaceLabelsDesc(labelKeys), 1, labelValues...)
}
