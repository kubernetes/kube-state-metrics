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
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/openshift/origin/pkg/util/proc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/pflag"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	resyncPeriod = 5 * time.Minute
	metricsPath  = "/metrics"
	healthzPath  = "/healthz"
)

var (
	defaultCollectors = collectorSet{
		"daemonsets":             struct{}{},
		"deployments":            struct{}{},
		"pods":                   struct{}{},
		"nodes":                  struct{}{},
		"resourcequotas":         struct{}{},
		"replicasets":            struct{}{},
		"replicationcontrollers": struct{}{},
	}
	availableCollectors = map[string]func(registry prometheus.Registerer, kubeClient clientset.Interface){
		"daemonsets":             RegisterDaemonSetCollector,
		"deployments":            RegisterDeploymentCollector,
		"pods":                   RegisterPodCollector,
		"nodes":                  RegisterNodeCollector,
		"resourcequotas":         RegisterResourceQuotaCollector,
		"replicasets":            RegisterReplicaSetCollector,
		"replicationcontrollers": RegisterReplicationControllerCollector,
	}
)

type collectorSet map[string]struct{}

func (c *collectorSet) String() string {
	s := *c
	return strings.Join(s.asSlice(), ",")
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
	return "map[string]struct{}"
}

type options struct {
	inCluster  bool
	apiserver  string
	kubeconfig string
	help       bool
	port       int
	collectors collectorSet
}

func main() {
	// configure glog
	flag.CommandLine.Parse([]string{})
	flag.Lookup("logtostderr").Value.Set("true")

	options := &options{collectors: make(collectorSet)}
	flags := pflag.NewFlagSet("", pflag.ExitOnError)

	flags.BoolVar(&options.inCluster, "in-cluster", true, `If true, use the built in kubernetes cluster for creating the client`)
	flags.StringVar(&options.apiserver, "apiserver", "", `The URL of the apiserver to use as a master`)
	flags.StringVar(&options.kubeconfig, "kubeconfig", "", "Absolute path to the kubeconfig file")
	flags.BoolVarP(&options.help, "help", "h", false, "Print help text")
	flags.IntVar(&options.port, "port", 80, `Port to expose metrics on.`)
	flags.Var(&options.collectors, "collectors", "Collectors to be enabled")

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
		glog.Infof("Using default collectors")
		collectors = defaultCollectors
	} else {
		collectors = options.collectors
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

	registerCollectors(kubeClient, collectors)
	metricsServer(options.port)
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
	_, err = kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("ERROR communicating with apiserver: %v", err)
	}
	glog.Infof("Communication with server successful")

	return kubeClient, nil
}

func metricsServer(port int) {
	// Address to listen on for web interface and telemetry
	listenAddress := fmt.Sprintf(":%d", port)

	glog.Infof("Starting metrics server: %s", listenAddress)
	// Add metricsPath
	http.Handle(metricsPath, prometheus.UninstrumentedHandler())
	// Add healthzPath
	http.HandleFunc(healthzPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	})
	// Add index
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
	log.Fatal(http.ListenAndServe(listenAddress, nil))
}

// registerCollectors creates and starts informers and initializes and
// registers metrics for collection.
func registerCollectors(kubeClient clientset.Interface, enabledCollectors collectorSet) {
	activeCollectors := []string{}
	for c, _ := range enabledCollectors {
		f, ok := availableCollectors[c]
		if ok {
			f(prometheus.DefaultRegisterer, kubeClient)
			activeCollectors = append(activeCollectors, c)
		}
	}

	glog.Infof("Active collectors: %s", strings.Join(activeCollectors, ","))
}
