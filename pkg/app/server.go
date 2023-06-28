/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package app

import (
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"k8s.io/client-go/rest"

	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	versionCollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	"gopkg.in/yaml.v3"
	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Initialize common client auth plugins.
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	"k8s.io/kube-state-metrics/v2/pkg/metricshandler"
	"k8s.io/kube-state-metrics/v2/pkg/optin"
	"k8s.io/kube-state-metrics/v2/pkg/options"

	"k8s.io/kube-state-metrics/v2/pkg/util/proc"

	crmonitorclientset "k8s.io/kube-state-metrics/v2/pkg/customresourcemonitor/client/clientset/versioned"

	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

type Reconfigure struct {
}

func (r *Reconfigure) ResolveCustomResourceConfig(opts *options.Options) (customresourcestate.ConfigDecoder, error) {
	return resolveCustomResourceConfig(opts)
}

func (r *Reconfigure) FromConfig(decoder customresourcestate.ConfigDecoder) ([]customresource.RegistryFactory, error) {
	return customresourcestate.FromConfig2(decoder)
}

// promLogger implements promhttp.Logger
type promLogger struct{}

func (pl promLogger) Println(v ...interface{}) {
	klog.Error(v...)
}

// promLogger implements the Logger interface
func (pl promLogger) Log(v ...interface{}) error {
	klog.Info(v...)
	return nil
}

// RunKubeStateMetricsWrapper runs KSM with context cancellation.
func RunKubeStateMetricsWrapper(ctx context.Context, opts *options.Options) error {
	err := RunKubeStateMetrics(ctx, opts)
	if ctx.Err() == context.Canceled {
		klog.Infoln("Restarting: kube-state-metrics, metrics will be reset")
		return nil
	}
	return err
}

// RunKubeStateMetrics will build and run the kube-state-metrics.
// Any out-of-tree custom resource metrics could be registered by newing a registry factory
// which implements customresource.RegistryFactory and pass all factories into this function.
func RunKubeStateMetrics(ctx context.Context, opts *options.Options) error {
	promLogger := promLogger{}
	ksmMetricsRegistry := prometheus.NewRegistry()
	ksmMetricsRegistry.MustRegister(versionCollector.NewCollector("kube_state_metrics"))
	durationVec := promauto.With(ksmMetricsRegistry).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "A histogram of requests for kube-state-metrics metrics handler.",
			Buckets:     prometheus.DefBuckets,
			ConstLabels: prometheus.Labels{"handler": "metrics"},
		}, []string{"method"},
	)
	configHash := promauto.With(ksmMetricsRegistry).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_state_metrics_config_hash",
			Help: "Hash of the currently loaded configuration.",
		}, []string{"type", "filename"})
	configSuccess := promauto.With(ksmMetricsRegistry).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_state_metrics_last_config_reload_successful",
			Help: "Whether the last configuration reload attempt was successful.",
		}, []string{"type", "filename"})
	configSuccessTime := promauto.With(ksmMetricsRegistry).NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kube_state_metrics_last_config_reload_success_timestamp_seconds",
			Help: "Timestamp of the last successful configuration reload.",
		}, []string{"type", "filename"})

	// Register self-metrics to track the state of the cache.
	crdsAddEventsCounter := promauto.With(ksmMetricsRegistry).NewCounter(prometheus.CounterOpts{
		Name: "kube_state_metrics_custom_resource_state_add_events_total",
		Help: "Number of times that the CRD informer triggered the add event.",
	})
	crdsDeleteEventsCounter := promauto.With(ksmMetricsRegistry).NewCounter(prometheus.CounterOpts{
		Name: "kube_state_metrics_custom_resource_state_delete_events_total",
		Help: "Number of times that the CRD informer triggered the remove event.",
	})
	crdsCacheCountGauge := promauto.With(ksmMetricsRegistry).NewGauge(prometheus.GaugeOpts{
		Name: "kube_state_metrics_custom_resource_state_cache",
		Help: "Net amount of CRDs affecting the cache currently.",
	})
	storeBuilder := store.NewBuilder()
	storeBuilder.WithMetrics(ksmMetricsRegistry)

	got := options.GetConfigFile(*opts)
	if got != "" {
		configFile, err := os.ReadFile(filepath.Clean(got))
		if err != nil {
			return fmt.Errorf("failed to read opts config file: %v", err)
		}
		// NOTE: Config value will override default values of intersecting options.
		err = yaml.Unmarshal(configFile, opts)
		if err != nil {
			// DO NOT end the process.
			// We want to allow the user to still be able to fix the misconfigured config (redeploy or edit the configmaps) and reload KSM automatically once that's done.
			klog.ErrorS(err, "failed to unmarshal opts config file")
			// Wait for the next reload.
			klog.InfoS("misconfigured config detected, KSM will automatically reload on next write to the config")
			klog.InfoS("waiting for config to be fixed")
			configSuccess.WithLabelValues("config", filepath.Clean(got)).Set(0)
			<-ctx.Done()
		} else {
			configSuccess.WithLabelValues("config", filepath.Clean(got)).Set(1)
			configSuccessTime.WithLabelValues("config", filepath.Clean(got)).SetToCurrentTime()
			hash := md5HashAsMetricValue(configFile)
			configHash.WithLabelValues("config", filepath.Clean(got)).Set(hash)
		}
	}

	kubeConfig, err := clientcmd.BuildConfigFromFlags(opts.Apiserver, opts.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to build config from flags: %v", err)
	}

	// Loading custom resource state configuration from cli argument or config file
	config, err := resolveCustomResourceConfig(opts)
	if err != nil {
		return err
	}

	var factories []customresource.RegistryFactory

	if opts.CustomResourceConfigFile != "" {
		crcFile, err := os.ReadFile(filepath.Clean(opts.CustomResourceConfigFile))
		if err != nil {
			return fmt.Errorf("failed to read custom resource config file: %v", err)
		}
		configSuccess.WithLabelValues("customresourceconfig", filepath.Clean(opts.CustomResourceConfigFile)).Set(1)
		configSuccessTime.WithLabelValues("customresourceconfig", filepath.Clean(opts.CustomResourceConfigFile)).SetToCurrentTime()
		hash := md5HashAsMetricValue(crcFile)
		configHash.WithLabelValues("customresourceconfig", filepath.Clean(opts.CustomResourceConfigFile)).Set(hash)

	}

	resources := make([]string, len(factories))

	for i, factory := range factories {
		resources[i] = factory.Name()
	}

	switch {
	case len(opts.Resources) == 0 && !opts.CustomResourcesOnly:
		resources = append(resources, options.DefaultResources.AsSlice()...)
		klog.InfoS("Used default resources")
	case opts.CustomResourcesOnly:
		// enable custom resource only, these resources will be populated later on
		klog.InfoS("Used CRD resources only")
	default:
		resources = append(resources, opts.Resources.AsSlice()...)
		klog.InfoS("Used resources", "resources", resources)
	}

	if err := storeBuilder.WithEnabledResources(resources); err != nil {
		return fmt.Errorf("failed to set up resources: %v", err)
	}

	namespaces := opts.Namespaces.GetNamespaces()
	nsFieldSelector := namespaces.GetExcludeNSFieldSelector(opts.NamespacesDenylist)
	nodeFieldSelector := opts.Node.GetNodeFieldSelector()
	merged, err := storeBuilder.MergeFieldSelectors([]string{nsFieldSelector, nodeFieldSelector})
	if err != nil {
		return err
	}
	storeBuilder.WithNamespaces(namespaces)
	storeBuilder.WithFieldSelectorFilter(merged)

	allowDenyList, err := allowdenylist.New(opts.MetricAllowlist, opts.MetricDenylist)
	if err != nil {
		return err
	}

	err = allowDenyList.Parse()
	if err != nil {
		return fmt.Errorf("error initializing the allowdeny list: %v", err)
	}

	klog.InfoS("Metric allow-denylisting", "allowDenyStatus", allowDenyList.Status())

	optInMetricFamilyFilter, err := optin.NewMetricFamilyFilter(opts.MetricOptInList)
	if err != nil {
		return fmt.Errorf("error initializing the opt-in metric list: %v", err)
	}

	if optInMetricFamilyFilter.Count() > 0 {
		klog.InfoS("Metrics which were opted into", "optInMetricsFamilyStatus", optInMetricFamilyFilter.Status())
	}

	storeBuilder.WithFamilyGeneratorFilter(generator.NewCompositeFamilyGeneratorFilter(
		allowDenyList,
		optInMetricFamilyFilter,
	))

	storeBuilder.WithUsingAPIServerCache(opts.UseAPIServerCache)
	storeBuilder.WithGenerateStoresFunc(storeBuilder.DefaultGenerateStoresFunc())
	proc.StartReaper()

	storeBuilder.WithUtilOptions(opts)
	kubeClient, ksmCRClient, err := createKubeClient(opts.Apiserver, opts.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}
	storeBuilder.WithKubeClient(kubeClient)

	storeBuilder.WithSharding(opts.Shard, opts.TotalShards)
	if err := storeBuilder.WithAllowAnnotations(opts.AnnotationsAllowList); err != nil {
		return fmt.Errorf("failed to set up annotations allowlist: %v", err)
	}
	if err := storeBuilder.WithAllowLabels(opts.LabelsAllowList); err != nil {
		return fmt.Errorf("failed to set up labels allowlist: %v", err)
	}

	ksmMetricsRegistry.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)

	var g run.Group

	m := metricshandler.New(
		opts,
		kubeClient,
		ksmCRClient,
		storeBuilder,
		opts.EnableGZIPEncoding,
		&Reconfigure{},
	)
	// Run MetricsHandler
	if config == nil {
		ctxMetricsHandler, cancel := context.WithCancel(ctx)
		g.Add(func() error {
			return m.Run(ctxMetricsHandler)
		}, func(error) {
			cancel()
		})
	}

	tlsConfig := opts.TLSConfig

	// A nil CRS config implies that we need to hold off on all CRS operations.
	if config != nil {
		discovererInstance := &CRDiscoverer{
			CRDsAddEventsCounter:    crdsAddEventsCounter,
			CRDsDeleteEventsCounter: crdsDeleteEventsCounter,
			CRDsCacheCountGauge:     crdsCacheCountGauge,
		}
		// This starts a goroutine that will watch for any new GVKs to extract from CRDs.
		err = discovererInstance.StartDiscovery(ctx, kubeConfig)
		if err != nil {
			return err
		}
		// FromConfig will return different behaviours when a G**-based config is supplied (since that is subject to change based on the resources present in the cluster).
		fn, err := customresourcestate.FromConfig(config, discovererInstance)
		if err != nil {
			return err
		}
		// This starts a goroutine that will keep the cache up to date.
		discovererInstance.PollForCacheUpdates(
			ctx,
			opts,
			storeBuilder,
			m,
			fn,
		)
	}

	telemetryMux := buildTelemetryServer(ksmMetricsRegistry)
	telemetryListenAddress := net.JoinHostPort(opts.TelemetryHost, strconv.Itoa(opts.TelemetryPort))
	telemetryServer := http.Server{
		Handler:           telemetryMux,
		ReadHeaderTimeout: 5 * time.Second}
	telemetryFlags := web.FlagConfig{
		WebListenAddresses: &[]string{telemetryListenAddress},
		WebSystemdSocket:   new(bool),
		WebConfigFile:      &tlsConfig,
	}

	metricsMux := buildMetricsServer(m, durationVec)
	metricsServerListenAddress := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	metricsServer := http.Server{
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	metricsFlags := web.FlagConfig{
		WebListenAddresses: &[]string{metricsServerListenAddress},
		WebSystemdSocket:   new(bool),
		WebConfigFile:      &tlsConfig,
	}

	// Run Telemetry server
	{
		g.Add(func() error {
			klog.InfoS("Started kube-state-metrics self metrics server", "telemetryAddress", telemetryListenAddress)
			return web.ListenAndServe(&telemetryServer, &telemetryFlags, promLogger)
		}, func(error) {
			ctxShutDown, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			telemetryServer.Shutdown(ctxShutDown)
		})
	}
	// Run Metrics server
	{
		g.Add(func() error {
			klog.InfoS("Started metrics server", "metricsServerAddress", metricsServerListenAddress)
			return web.ListenAndServe(&metricsServer, &metricsFlags, promLogger)
		}, func(error) {
			ctxShutDown, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			metricsServer.Shutdown(ctxShutDown)
		})
	}

	if err := g.Run(); err != nil {
		return fmt.Errorf("run server group error: %v", err)
	}

	klog.InfoS("Exited")
	return nil
}

func buildTelemetryServer(registry prometheus.Gatherer) *http.ServeMux {
	mux := http.NewServeMux()

	// Add metricsPath
	mux.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorLog: promLogger{}}))

	// Add index
	landingConfig := web.LandingConfig{
		Name:        "kube-state-metrics",
		Description: "Self-metrics for kube-state-metrics",
		Version:     version.Info(),
		Links: []web.LandingLinks{
			{
				Address: metricsPath,
				Text:    "Metrics",
			},
		},
	}
	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		klog.ErrorS(err, "failed to create landing page")
	}
	mux.Handle("/", landingPage)
	return mux
}

func buildMetricsServer(m *metricshandler.MetricsHandler, durationObserver prometheus.ObserverVec) *http.ServeMux {
	mux := http.NewServeMux()

	// TODO: This doesn't belong into serveMetrics
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	mux.Handle(metricsPath, promhttp.InstrumentHandlerDuration(durationObserver, m))

	// Add healthzPath
	mux.HandleFunc(healthzPath, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	})

	// Add index
	landingConfig := web.LandingConfig{
		Name:        "kube-state-metrics",
		Description: "Metrics for Kubernetes' state",
		Version:     version.Info(),
		Links: []web.LandingLinks{
			{
				Address: metricsPath,
				Text:    "Metrics",
			},
			{
				Address: healthzPath,
				Text:    "Healthz",
			},
		},
	}
	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		klog.ErrorS(err, "failed to create landing page")
	}
	mux.Handle("/", landingPage)
	return mux
}

// md5HashAsMetricValue creates an md5 hash and returns the most significant bytes that fit into a float64
// Taken from https://github.com/prometheus/alertmanager/blob/6ef6e6868dbeb7984d2d577dd4bf75c65bf1904f/config/coordinator.go#L149
func md5HashAsMetricValue(data []byte) float64 {
	sum := md5.Sum(data) //nolint:gosec
	// We only want 48 bits as a float64 only has a 53 bit mantissa.
	smallSum := sum[0:6]
	bytes := make([]byte, 8)
	copy(bytes, smallSum)
	return float64(binary.LittleEndian.Uint64(bytes))
}

func resolveCustomResourceConfig(opts *options.Options) (customresourcestate.ConfigDecoder, error) {
	if opts.CustomResourcesKSMCRWatched {
		ksmCRClient, err := createKubeCRClient(opts.Apiserver, opts.Kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("Can not create KSM CR client: %v", err)
		}
		// TODO(): fetch CRs in all namespaces
		crMonitorList, err := ksmCRClient.CustomresourceV1alpha1().CustomResourceMonitors("kube-system").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("Can not list KSM CR: %v", err)
		}
		var crconfig customresourcestate.Metrics
		for _, item := range crMonitorList.Items {
			crconfig.Spec.Resources = append(crconfig.Spec.Resources, item.Spec.Resources...)
		}
		fmt.Printf("Merged custom resource %v \n", crconfig)
		var buf strings.Builder
		encoder := yaml.NewEncoder(&buf)
		err = encoder.Encode(&crconfig)
		if err != nil {
			return nil, fmt.Errorf("Can encode KSM CR: %v", err)
		}
		return yaml.NewDecoder(strings.NewReader(buf.String())), nil
	}

	if s := opts.CustomResourceConfig; s != "" {
		return yaml.NewDecoder(strings.NewReader(s)), nil
	}
	if file := opts.CustomResourceConfigFile; file != "" {
		f, err := os.Open(filepath.Clean(file))
		if err != nil {
			return nil, fmt.Errorf("Custom Resource State Metrics file could not be opened: %v", err)
		}
		return yaml.NewDecoder(f), nil
	}
	return nil, nil
}

func createKubeClient(apiserver string, kubeconfig string) (clientset.Interface, crmonitorclientset.Interface, error) {
	var config *rest.Config

	var err error

	if config == nil {
		config, err = clientcmd.BuildConfigFromFlags(apiserver, kubeconfig)
		if err != nil {
			return nil, nil, err
		}
	}

	config.UserAgent = fmt.Sprintf("%s/%s (%s/%s) kubernetes/%s", "kube-state-metrics", version.Version, runtime.GOOS, runtime.GOARCH, version.Revision)
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"

	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	// Informers don't seem to do a good job logging error messages when it
	// can't reach the server, making debugging hard. This makes it easier to
	// figure out if apiserver is configured incorrectly.
	klog.InfoS("Tested communication with server")
	v, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, nil, fmt.Errorf("error while trying to communicate with apiserver: %w", err)
	}
	klog.InfoS("Run with Kubernetes cluster version", "major", v.Major, "minor", v.Minor, "gitVersion", v.GitVersion, "gitTreeState", v.GitTreeState, "gitCommit", v.GitCommit, "platform", v.Platform)
	klog.InfoS("Communication with server successful")

	customResourceMonitorClient, err := crmonitorclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	return kubeClient, customResourceMonitorClient, nil
}

func createKubeCRClient(apiserver string, kubeconfig string) (crmonitorclientset.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags(apiserver, kubeconfig)
	if err != nil {
		return nil, err
	}

	config.UserAgent = version.Version
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"

	customResourceMonitorClients, err := crmonitorclientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return customResourceMonitorClients, nil
}
