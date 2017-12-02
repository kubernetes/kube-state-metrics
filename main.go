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
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/golang/glog"
	"github.com/openshift/origin/pkg/util/proc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"net/http/pprof"

	"k8s.io/kube-state-metrics/collectors"
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

var (
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
		"persistentvolumeclaims":   struct{}{},
		"namespaces":               struct{}{},
		"horizontalpodautoscalers": struct{}{},
	}
	availableCollectors = map[string]func(registry prometheus.Registerer, kubeClient clientset.Interface, namespace string){
		"cronjobs":                 collectors.RegisterCronJobCollector,
		"daemonsets":               collectors.RegisterDaemonSetCollector,
		"deployments":              collectors.RegisterDeploymentCollector,
		"jobs":                     collectors.RegisterJobCollector,
		"limitranges":              collectors.RegisterLimitRangeCollector,
		"nodes":                    collectors.RegisterNodeCollector,
		"pods":                     collectors.RegisterPodCollector,
		"replicasets":              collectors.RegisterReplicaSetCollector,
		"replicationcontrollers":   collectors.RegisterReplicationControllerCollector,
		"resourcequotas":           collectors.RegisterResourceQuotaCollector,
		"services":                 collectors.RegisterServiceCollector,
		"statefulsets":             collectors.RegisterStatefulSetCollector,
		"persistentvolumeclaims":   collectors.RegisterPersistentVolumeClaimCollector,
		"namespaces":               collectors.RegisterNamespaceCollector,
		"horizontalpodautoscalers": collectors.RegisterHorizontalPodAutoScalerCollector,
	}
)

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
		_, ok := availableCollectors[col]
		if !ok {
			glog.Fatalf("Collector \"%s\" does not exist", col)
		}
		s[col] = struct{}{}
	}
	return nil
}

func (c collectorSet) asSlice() []string {
	cols := []string{}
	for col, _ := range c {
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

type options struct {
	inCluster  bool
	apiserver  string
	kubeconfig string
	help       bool
	port       int
	collectors collectorSet
	namespace  string
}

func main() {
	options := &options{collectors: make(collectorSet)}
	flags := pflag.NewFlagSet("", pflag.ExitOnError)
	// add glog flags
	flags.AddGoFlagSet(flag.CommandLine)
	flags.Lookup("logtostderr").Value.Set("true")
	flags.Lookup("logtostderr").DefValue = "true"
	flags.Lookup("logtostderr").NoOptDefVal = "true"
	flags.BoolVar(&options.inCluster, "in-cluster", true, `If true, use the built in kubernetes cluster for creating the client`)
	flags.StringVar(&options.apiserver, "apiserver", "", `The URL of the apiserver to use as a master`)
	flags.StringVar(&options.kubeconfig, "kubeconfig", "", "Absolute path to the kubeconfig file")
	flags.BoolVarP(&options.help, "help", "h", false, "Print help text")
	flags.IntVar(&options.port, "port", 80, `Port to expose metrics on.`)
	flags.Var(&options.collectors, "collectors", fmt.Sprintf("Comma-separated list of collectors to be enabled. Defaults to %q", &defaultCollectors))
	flags.StringVar(&options.namespace, "namespace", metav1.NamespaceAll, "namespace to be enabled for collecting resources")

	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flags.PrintDefaults()
	}

	err := flags.Parse(os.Args)
	if err != nil {
		glog.Fatalf("Error: %s", err)
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

	if options.namespace == metav1.NamespaceAll {
		glog.Info("Using all namespace")
	} else {
		glog.Infof("Using %s namespace", options.namespace)
	}

	if isNotExists(options.kubeconfig) && !(options.inCluster) {
		glog.Fatalf("kubeconfig invalid and --in-cluster is false; kubeconfig must be set to a valid file(kubeconfig default file name: $HOME/.kube/config)")
	}
	if options.apiserver != "" {
		glog.Infof("apiserver set to: %v", options.apiserver)
	}

	proc.StartReaper()

	kubeClient, err := createKubeClient(options.inCluster, options.apiserver, options.kubeconfig)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	registry := prometheus.NewRegistry()
	registerCollectors(registry, kubeClient, collectors, options.namespace)
	metricsServer(registry, options.port)
}

func isNotExists(file string) bool {
	if file == "" {
		file = clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
	}
	_, err := os.Stat(file)
	return os.IsNotExist(err)
}

func createKubeClient(inCluster bool, apiserver string, kubeconfig string) (kubeClient clientset.Interface, err error) {
	if inCluster {
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		// Allow overriding of apiserver even if using inClusterConfig
		// (necessary if kube-proxy isn't properly set up).
		if apiserver != "" {
			config.Host = apiserver
		}
		tokenPresent := false
		if len(config.BearerToken) > 0 {
			tokenPresent = true
		}
		glog.Infof("service account token present: %v", tokenPresent)
		glog.Infof("service host: %s", config.Host)
		if kubeClient, err = clientset.NewForConfig(config); err != nil {
			return nil, err
		}
	} else {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		// if you want to change the loading rules (which files in which order), you can do so here
		loadingRules.ExplicitPath = kubeconfig
		configOverrides := &clientcmd.ConfigOverrides{}
		// if you want to change override values or bind them to flags, there are methods to help you
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err := kubeConfig.ClientConfig()
		//config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		//config, err := clientcmd.DefaultClientConfig.ClientConfig()
		if err != nil {
			return nil, err
		}
		kubeClient, err = clientset.NewForConfig(config)
		if err != nil {
			return nil, err
		}
	}

	// Informers don't seem to do a good job logging error messages when it
	// can't reach the server, making debugging hard. This makes it easier to
	// figure out if apiserver is configured incorrectly.
	glog.Infof("Testing communication with server")
	v, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("ERROR communicating with apiserver: %v", err)
	}
	glog.Infof("Running in Kubernetes Cluster version v%v.%v (%v) - git (%v) commit %v - platform %v",
		v.Major, v.Minor, v.GitVersion, v.GitTreeState, v.GitCommit, v.Platform)
	glog.Infof("Communication with server successful")

	return kubeClient, nil
}

func metricsServer(registry prometheus.Gatherer, port int) {
	// Address to listen on for web interface and telemetry
	listenAddress := fmt.Sprintf(":%d", port)

	glog.Infof("Starting metrics server: %s", listenAddress)

	mux := http.NewServeMux()

	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	// Add metricsPath
	mux.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
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
func registerCollectors(registry prometheus.Registerer, kubeClient clientset.Interface, enabledCollectors collectorSet, namespace string) {
	activeCollectors := []string{}
	for c := range enabledCollectors {
		f, ok := availableCollectors[c]
		if ok {
			f(registry, kubeClient, namespace)
			activeCollectors = append(activeCollectors, c)
		}
	}

	glog.Infof("Active collectors: %s", strings.Join(activeCollectors, ","))
}
