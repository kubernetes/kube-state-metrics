/*
Copyright 2015 The Kubernetes Authors All rights reserved.

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

package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift/origin/pkg/util/proc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"

	kcollectors "k8s.io/kube-state-metrics/pkg/collectors"
	"k8s.io/kube-state-metrics/pkg/metrics"
	"k8s.io/kube-state-metrics/pkg/options"
	"k8s.io/kube-state-metrics/pkg/version"
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

// promLogger implements promhttp.Logger
type promLogger struct{}

func (pl promLogger) Println(v ...interface{}) {
	glog.Error(v)
}

func main() {
	opts := options.NewOptions()
	opts.AddFlags()

	err := opts.Parse()
	if err != nil {
		glog.Fatalf("Error: %s", err)
	}

	if opts.Version {
		fmt.Printf("%#v\n", version.GetVersion())
		os.Exit(0)
	}

	if opts.Help {
		opts.Usage()
		os.Exit(0)
	}

	var collectors options.CollectorSet
	if len(opts.Collectors) == 0 {
		glog.Info("Using default collectors")
		collectors = options.DefaultCollectors
	} else {
		collectors = opts.Collectors
	}

	var namespaces options.NamespaceList
	if len(opts.Namespaces) == 0 {
		namespaces = options.DefaultNamespaces
	} else {
		namespaces = opts.Namespaces
	}

	if namespaces.IsAllNamespaces() {
		glog.Info("Using all namespace")
	} else {
		glog.Infof("Using %s namespaces", namespaces)
	}

	if opts.MetricWhitelist.IsEmpty() && opts.MetricBlacklist.IsEmpty() {
		glog.Info("No metric whitelist or blacklist set. No filtering of metrics will be done.")
	}
	if !opts.MetricWhitelist.IsEmpty() && !opts.MetricBlacklist.IsEmpty() {
		glog.Fatal("Whitelist and blacklist are both set. They are mutually exclusive, only one of them can be set.")
	}
	if !opts.MetricWhitelist.IsEmpty() {
		glog.Infof("A metric whitelist has been configured. Only the following metrics will be exposed: %s.", opts.MetricWhitelist.String())
	}
	if !opts.MetricBlacklist.IsEmpty() {
		glog.Infof("A metric blacklist has been configured. The following metrics will not be exposed: %s.", opts.MetricBlacklist.String())
	}

	proc.StartReaper()

	kubeClient, err := createKubeClient(opts.Apiserver, opts.Kubeconfig)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	ksmMetricsRegistry := prometheus.NewRegistry()
	ksmMetricsRegistry.Register(kcollectors.ResourcesPerScrapeMetric)
	ksmMetricsRegistry.Register(kcollectors.ScrapeErrorTotalMetric)
	ksmMetricsRegistry.Register(prometheus.NewProcessCollector(os.Getpid(), ""))
	ksmMetricsRegistry.Register(prometheus.NewGoCollector())
	go telemetryServer(ksmMetricsRegistry, opts.TelemetryHost, opts.TelemetryPort)

	registry := prometheus.NewRegistry()
	registerCollectors(registry, kubeClient, collectors, namespaces, opts)
	metricsServer(metrics.FilteredGatherer(registry, opts.MetricWhitelist, opts.MetricBlacklist), opts.Host, opts.Port)
}

func createKubeClient(apiserver string, kubeconfig string) (clientset.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags(apiserver, kubeconfig)
	if err != nil {
		return nil, err
	}

	config.UserAgent = version.GetVersion().String()
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"

	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Informers don't seem to do a good job logging error messages when it
	// can't reach the server, making debugging hard. This makes it easier to
	// figure out if apiserver is configured incorrectly.
	glog.Infof("Testing communication with server")
	v, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("ERROR communicating with apiserver: %v", err)
	}
	glog.Infof("Running with Kubernetes cluster version: v%s.%s. git version: %s. git tree state: %s. commit: %s. platform: %s",
		v.Major, v.Minor, v.GitVersion, v.GitTreeState, v.GitCommit, v.Platform)
	glog.Infof("Communication with server successful")

	return kubeClient, nil
}

func telemetryServer(registry prometheus.Gatherer, host string, port int) {
	// Address to listen on for web interface and telemetry
	listenAddress := net.JoinHostPort(host, strconv.Itoa(port))

	glog.Infof("Starting kube-state-metrics self metrics server: %s", listenAddress)

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
	log.Fatal(http.ListenAndServe(listenAddress, mux))
}

func metricsServer(registry prometheus.Gatherer, host string, port int) {
	// Address to listen on for web interface and telemetry
	listenAddress := net.JoinHostPort(host, strconv.Itoa(port))

	glog.Infof("Starting metrics server: %s", listenAddress)

	mux := http.NewServeMux()

	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	// Add metricsPath
	mux.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorLog: promLogger{}}))
	// Add healthzPath
	mux.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
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
	log.Fatal(http.ListenAndServe(listenAddress, mux))
}

// registerCollectors creates and starts informers and initializes and
// registers metrics for collection.
func registerCollectors(registry prometheus.Registerer, kubeClient clientset.Interface, enabledCollectors options.CollectorSet, namespaces options.NamespaceList, opts *options.Options) {
	informerFactories := []informers.SharedInformerFactory{}
	for _, ns := range namespaces {
		informerFactories = append(
			informerFactories,
			informers.NewSharedInformerFactoryWithOptions(
				kubeClient, 0, informers.WithNamespace(ns),
			),
		)
	}
	activeCollectors := []string{}
	for c := range enabledCollectors {
		f, ok := kcollectors.AvailableCollectors[c]
		if ok {
			f(registry, informerFactories, opts)
			activeCollectors = append(activeCollectors, c)
		}
	}

	glog.Infof("Active collectors: %s", strings.Join(activeCollectors, ","))
}
