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
	"strconv"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	vpaclientset "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Initialize common client auth plugins.
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/allowdenylist"
	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	"k8s.io/kube-state-metrics/v2/pkg/metricshandler"
	"k8s.io/kube-state-metrics/v2/pkg/optin"
	"k8s.io/kube-state-metrics/v2/pkg/options"
	"k8s.io/kube-state-metrics/v2/pkg/util/proc"
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

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
func RunKubeStateMetricsWrapper(ctx context.Context, opts *options.Options, factories ...customresource.RegistryFactory) error {
	err := RunKubeStateMetrics(ctx, opts, factories...)
	if ctx.Err() == context.Canceled {
		klog.Infoln("Restarting: kube-state-metrics, metrics will be reset")
		return nil
	}
	return err
}

// RunKubeStateMetrics will build and run the kube-state-metrics.
// Any out-of-tree custom resource metrics could be registered by newing a registry factory
// which implements customresource.RegistryFactory and pass all factories into this function.
func RunKubeStateMetrics(ctx context.Context, opts *options.Options, factories ...customresource.RegistryFactory) error {
	promLogger := promLogger{}

	storeBuilder := store.NewBuilder()
	storeBuilder.WithCustomResourceStoreFactories(factories...)

	ksmMetricsRegistry := prometheus.NewRegistry()
	ksmMetricsRegistry.MustRegister(version.NewCollector("kube_state_metrics"))
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
			klog.Warningf("failed to unmarshal opts config file: %v", err)
			// Wait for the next reload.
			klog.Infof("misconfigured config detected, KSM will automatically reload on next write to the config")
			klog.Infof("waiting for config to be fixed")
			configSuccess.WithLabelValues("config", filepath.Clean(got)).Set(0)
			<-ctx.Done()
		} else {
			configSuccess.WithLabelValues("config", filepath.Clean(got)).Set(1)
			configSuccessTime.WithLabelValues("config", filepath.Clean(got)).SetToCurrentTime()
			hash := md5HashAsMetricValue(configFile)
			configHash.WithLabelValues("config", filepath.Clean(got)).Set(hash)
		}
	}

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

	var resources []string
	switch {
	case len(opts.Resources) == 0 && !opts.CustomResourcesOnly:
		klog.InfoS("Used default resources")
		resources = options.DefaultResources.AsSlice()
		// enable custom resource
		for _, factory := range factories {
			resources = append(resources, factory.Name())
		}
	case opts.CustomResourcesOnly:
		// enable custom resource only
		for _, factory := range factories {
			resources = append(resources, factory.Name())
		}
		klog.InfoS("Used CRD resources only", "resources", resources)
	default:
		klog.InfoS("Used resources", "resources", opts.Resources.String())
		resources = opts.Resources.AsSlice()
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
	storeBuilder.WithGenerateCustomResourceStoresFunc(storeBuilder.DefaultGenerateCustomResourceStoresFunc())

	proc.StartReaper()

	kubeClient, vpaClient, customResourceClients, err := createKubeClient(opts.Apiserver, opts.Kubeconfig, factories...)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}
	storeBuilder.WithKubeClient(kubeClient)
	storeBuilder.WithVPAClient(vpaClient)
	storeBuilder.WithCustomResourceClients(customResourceClients)
	storeBuilder.WithSharding(opts.Shard, opts.TotalShards)
	storeBuilder.WithAllowAnnotations(opts.AnnotationsAllowList)
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
		storeBuilder,
		opts.EnableGZIPEncoding,
	)
	// Run MetricsHandler
	{
		ctxMetricsHandler, cancel := context.WithCancel(ctx)
		g.Add(func() error {
			return m.Run(ctxMetricsHandler)
		}, func(error) {
			cancel()
		})
	}

	tlsConfig := opts.TLSConfig

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
		ReadHeaderTimeout: 5 * time.Second}

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

func createKubeClient(apiserver string, kubeconfig string, factories ...customresource.RegistryFactory) (clientset.Interface, vpaclientset.Interface, map[string]interface{}, error) {
	config, err := clientcmd.BuildConfigFromFlags(apiserver, kubeconfig)
	if err != nil {
		return nil, nil, nil, err
	}

	config.UserAgent = version.Version
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"

	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	vpaClient, err := vpaclientset.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, err
	}

	customResourceClients := make(map[string]interface{}, len(factories))
	for _, f := range factories {
		customResourceClient, err := f.CreateClient(config)
		if err != nil {
			return nil, nil, nil, err
		}
		customResourceClients[f.Name()] = customResourceClient
	}

	// Informers don't seem to do a good job logging error messages when it
	// can't reach the server, making debugging hard. This makes it easier to
	// figure out if apiserver is configured incorrectly.
	klog.InfoS("Tested communication with server")
	v, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error while trying to communicate with apiserver: %w", err)
	}
	klog.InfoS("Run with Kubernetes cluster version", "major", v.Major, "minor", v.Minor, "gitVersion", v.GitVersion, "gitTreeState", v.GitTreeState, "gitCommit", v.GitCommit, "platform", v.Platform)
	klog.InfoS("Communication with server successful")

	return kubeClient, vpaClient, customResourceClients, nil
}

func buildTelemetryServer(registry prometheus.Gatherer) *http.ServeMux {
	mux := http.NewServeMux()

	// Add metricsPath
	mux.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorLog: promLogger{}}))
	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Kube-State-Metrics Metrics Server</title></head>
             <body>
             <h1>Kube-State-Metrics Metrics</h1>
			 <ul>
             <li><a href='` + metricsPath + `'>metrics</a></li>
			 </ul>
             </body>
             </html>`))
	})
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
	mux.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
	})
	// Add index
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Kube Metrics Server</title></head>
             <body>
             <h1>Kube Metrics</h1>
			 <ul>
             <li><a href='` + metricsPath + `'>metrics</a></li>
             <li><a href='` + healthzPath + `'>healthz</a></li>
			 </ul>
             </body>
             </html>`))
	})
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
