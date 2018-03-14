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
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift/origin/pkg/util/proc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	kcollectors "k8s.io/kube-state-metrics/collectors"
	"k8s.io/kube-state-metrics/version"
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

var (
	defaultNamespaces = namespaceList{metav1.NamespaceAll}
	defaultCollectors = collectorSet{
		"daemonsets":               struct{}{},
		"deployments":              struct{}{},
		"limitranges":              struct{}{},
		"nodes":                    struct{}{},
		"pods":                     struct{}{},
		"replicasets":              struct{}{},
		"replicationcontrollers":   struct{}{},
		"resourcequotas":           struct{}{},
		"services":                 struct{}{},
		"jobs":                     struct{}{},
		"cronjobs":                 struct{}{},
		"statefulsets":             struct{}{},
		"persistentvolumes":        struct{}{},
		"persistentvolumeclaims":   struct{}{},
		"namespaces":               struct{}{},
		"horizontalpodautoscalers": struct{}{},
		"endpoints":                struct{}{},
		"secrets":                  struct{}{},
		"configmaps":               struct{}{},
	}
	availableCollectors = map[string]func(registry prometheus.Registerer, kubeClient clientset.Interface, namespaces []string){
		"cronjobs":                 kcollectors.RegisterCronJobCollector,
		"daemonsets":               kcollectors.RegisterDaemonSetCollector,
		"deployments":              kcollectors.RegisterDeploymentCollector,
		"jobs":                     kcollectors.RegisterJobCollector,
		"limitranges":              kcollectors.RegisterLimitRangeCollector,
		"nodes":                    kcollectors.RegisterNodeCollector,
		"pods":                     kcollectors.RegisterPodCollector,
		"replicasets":              kcollectors.RegisterReplicaSetCollector,
		"replicationcontrollers":   kcollectors.RegisterReplicationControllerCollector,
		"resourcequotas":           kcollectors.RegisterResourceQuotaCollector,
		"services":                 kcollectors.RegisterServiceCollector,
		"statefulsets":             kcollectors.RegisterStatefulSetCollector,
		"persistentvolumes":        kcollectors.RegisterPersistentVolumeCollector,
		"persistentvolumeclaims":   kcollectors.RegisterPersistentVolumeClaimCollector,
		"namespaces":               kcollectors.RegisterNamespaceCollector,
		"horizontalpodautoscalers": kcollectors.RegisterHorizontalPodAutoScalerCollector,
		"endpoints":                kcollectors.RegisterEndpointCollector,
		"secrets":                  kcollectors.RegisterSecretCollector,
		"configmaps":               kcollectors.RegisterConfigMapCollector,
	}
)

// promLogger implements promhttp.Logger
type promLogger struct{}

func (pl promLogger) Println(v ...interface{}) {
	glog.Error(v)
}

type collectorSet map[string]struct{}

func (c *collectorSet) String() string {
	s := *c
	ss := s.asSlice()
	sort.Strings(ss)
	return strings.Join(ss, ",")
}

func (c *collectorSet) Set(value string) error {
	s := *c
	cols := strings.Split(value, ",")
	for _, col := range cols {
		col = strings.TrimSpace(col)
		if len(col) != 0 {
			_, ok := availableCollectors[col]
			if !ok {
				glog.Fatalf("Collector \"%s\" does not exist", col)
			}
			s[col] = struct{}{}
		}
	}
	return nil
}

func (c collectorSet) asSlice() []string {
	cols := []string{}
	for col := range c {
		cols = append(cols, col)
	}
	return cols
}

func (c collectorSet) isEmpty() bool {
	return len(c.asSlice()) == 0
}

func (c *collectorSet) Type() string {
	return "string"
}

type namespaceList []string

func (n *namespaceList) String() string {
	return strings.Join(*n, ",")
}

func (n *namespaceList) IsAllNamespaces() bool {
	return len(*n) == 1 && (*n)[0] == metav1.NamespaceAll
}

func (n *namespaceList) Set(value string) error {
	splittedNamespaces := strings.Split(value, ",")
	for _, ns := range splittedNamespaces {
		ns = strings.TrimSpace(ns)
		if len(ns) != 0 {
			*n = append(*n, ns)
		}
	}
	return nil
}

func (n *namespaceList) Type() string {
	return "string"
}

type options struct {
	apiserver     string
	kubeconfig    string
	help          bool
	port          int
	host          string
	telemetryPort int
	telemetryHost string
	collectors    collectorSet
	namespaces    namespaceList
	version       bool
}

func main() {
	options := &options{collectors: make(collectorSet)}
	flags := pflag.NewFlagSet("", pflag.ExitOnError)
	// add glog flags
	flags.AddGoFlagSet(flag.CommandLine)
	flags.Lookup("logtostderr").Value.Set("true")
	flags.Lookup("logtostderr").DefValue = "true"
	flags.Lookup("logtostderr").NoOptDefVal = "true"
	flags.StringVar(&options.apiserver, "apiserver", "", `The URL of the apiserver to use as a master`)
	flags.StringVar(&options.kubeconfig, "kubeconfig", "", "Absolute path to the kubeconfig file")
	flags.BoolVarP(&options.help, "help", "h", false, "Print help text")
	flags.IntVar(&options.port, "port", 80, `Port to expose metrics on.`)
	flags.StringVar(&options.host, "host", "0.0.0.0", `Host to expose metrics on.`)
	flags.IntVar(&options.telemetryPort, "telemetry-port", 81, `Port to expose kube-state-metrics self metrics on.`)
	flags.StringVar(&options.telemetryHost, "telemetry-host", "0.0.0.0", `Host to expose kube-state-metrics self metrics on.`)
	flags.Var(&options.collectors, "collectors", fmt.Sprintf("Comma-separated list of collectors to be enabled. Defaults to %q", &defaultCollectors))
	flags.Var(&options.namespaces, "namespace", fmt.Sprintf("Comma-separated list of namespaces to be enabled. Defaults to %q", &defaultNamespaces))
	flags.BoolVarP(&options.version, "version", "", false, "kube-state-metrics build version information")

	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flags.PrintDefaults()
	}

	err := flags.Parse(os.Args)
	if err != nil {
		glog.Fatalf("Error: %s", err)
	}

	if options.version {
		fmt.Printf("%#v\n", version.GetVersion())
		os.Exit(0)
	}

	if options.help {
		flags.Usage()
		os.Exit(0)
	}

	var collectors collectorSet
	if len(options.collectors) == 0 {
		glog.Info("Using default collectors")
		collectors = defaultCollectors
	} else {
		collectors = options.collectors
	}

	var namespaces namespaceList
	if len(options.namespaces) == 0 {
		namespaces = defaultNamespaces
	} else {
		namespaces = options.namespaces
	}

	if namespaces.IsAllNamespaces() {
		glog.Info("Using all namespace")
	} else {
		glog.Infof("Using %s namespaces", namespaces)
	}

	proc.StartReaper()

	kubeClient, err := createKubeClient(options.apiserver, options.kubeconfig)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	ksmMetricsRegistry := prometheus.NewRegistry()
	ksmMetricsRegistry.Register(kcollectors.ResourcesPerScrapeMetric)
	ksmMetricsRegistry.Register(kcollectors.ScrapeErrorTotalMetric)
	ksmMetricsRegistry.Register(prometheus.NewProcessCollector(os.Getpid(), ""))
	ksmMetricsRegistry.Register(prometheus.NewGoCollector())
	go telemetryServer(ksmMetricsRegistry, options.telemetryHost, options.telemetryPort)

	registry := prometheus.NewRegistry()
	registerCollectors(registry, kubeClient, collectors, namespaces)
	metricsServer(registry, options.host, options.port)
}

func createKubeClient(apiserver string, kubeconfig string) (clientset.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags(apiserver, kubeconfig)
	if err != nil {
		return nil, err
	}

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
func registerCollectors(registry prometheus.Registerer, kubeClient clientset.Interface, enabledCollectors collectorSet, namespaces namespaceList) {
	activeCollectors := []string{}
	for c := range enabledCollectors {
		f, ok := availableCollectors[c]
		if ok {
			f(registry, kubeClient, namespaces)
			activeCollectors = append(activeCollectors, c)
		}
	}

	glog.Infof("Active collectors: %s", strings.Join(activeCollectors, ","))
}
