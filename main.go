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
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	appsv1beta1 "k8s.io/api/apps/v1beta1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	kcollectors "k8s.io/kube-state-metrics/collectors"
	cronjobbatchv1beta1 "k8s.io/kube-state-metrics/collectors/cronjob/batch/v1beta1"
	daemonsetextensionsv1beta1 "k8s.io/kube-state-metrics/collectors/daemonset/extensions/v1beta1"
	deploymentextensionsv1beta1 "k8s.io/kube-state-metrics/collectors/deployment/extensions/v1beta1"
	endpointcorev1 "k8s.io/kube-state-metrics/collectors/endpoint/core/v1"
	hpaautoscalingv1 "k8s.io/kube-state-metrics/collectors/hpa/autoscaling/v1"
	jobbatchv1 "k8s.io/kube-state-metrics/collectors/job/batch/v1"
	limitrangecorev1 "k8s.io/kube-state-metrics/collectors/limitrange/core/v1"
	namespacecorev1 "k8s.io/kube-state-metrics/collectors/namespace/core/v1"
	nodecorev1 "k8s.io/kube-state-metrics/collectors/node/core/v1"
	persistentvolumecorev1 "k8s.io/kube-state-metrics/collectors/persistentvolume/core/v1"
	persistentvolumeclaimcorev1 "k8s.io/kube-state-metrics/collectors/persistentvolumeclaim/core/v1"
	podcorev1 "k8s.io/kube-state-metrics/collectors/pod/core/v1"
	replicasetextensionsv1beta1 "k8s.io/kube-state-metrics/collectors/replicaset/extensions/v1beta1"
	replicationcontrollercorev1 "k8s.io/kube-state-metrics/collectors/replicationcontroller/core/v1"
	resourcequotacorev1 "k8s.io/kube-state-metrics/collectors/resourcequota/core/v1"
	servicecorev1 "k8s.io/kube-state-metrics/collectors/service/core/v1"
	statefulsetappsv1beta1 "k8s.io/kube-state-metrics/collectors/statefulset/apps/v1beta1"
	"k8s.io/kube-state-metrics/version"
)

const (
	metricsPath = "/metrics"
	healthzPath = "/healthz"
)

var (
	requiredVerbs = []string{"list", "watch"}
)

var (
	supportedCollectors = collectorsMap{
		"daemonsets":               extensionsv1beta1.SchemeGroupVersion.String(),
		"deployments":              extensionsv1beta1.SchemeGroupVersion.String(),
		"limitranges":              corev1.SchemeGroupVersion.String(),
		"nodes":                    corev1.SchemeGroupVersion.String(),
		"pods":                     corev1.SchemeGroupVersion.String(),
		"replicasets":              extensionsv1beta1.SchemeGroupVersion.String(),
		"replicationcontrollers":   corev1.SchemeGroupVersion.String(),
		"resourcequotas":           corev1.SchemeGroupVersion.String(),
		"services":                 corev1.SchemeGroupVersion.String(),
		"jobs":                     batchv1.SchemeGroupVersion.String(),
		"cronjobs":                 batchv1beta1.SchemeGroupVersion.String(),
		"statefulsets":             appsv1beta1.SchemeGroupVersion.String(),
		"persistentvolumes":        corev1.SchemeGroupVersion.String(),
		"persistentvolumeclaims":   corev1.SchemeGroupVersion.String(),
		"namespaces":               corev1.SchemeGroupVersion.String(),
		"horizontalpodautoscalers": autoscalingv1.SchemeGroupVersion.String(),
		"endpoints":                corev1.SchemeGroupVersion.String(),
	}
	registeredCollectors = map[string]map[string]func(registry prometheus.Registerer, kubeClient clientset.Interface, namespace string){
		"cronjobs": {
			batchv1beta1.SchemeGroupVersion.String(): cronjobbatchv1beta1.RegisterCronJobCollector,
		},
		"daemonsets": {
			extensionsv1beta1.SchemeGroupVersion.String(): daemonsetextensionsv1beta1.RegisterDaemonSetCollector,
		},
		"deployments": {
			extensionsv1beta1.SchemeGroupVersion.String(): deploymentextensionsv1beta1.RegisterDeploymentCollector,
		},
		"jobs": {
			batchv1.SchemeGroupVersion.String(): jobbatchv1.RegisterJobCollector,
		},
		"limitranges": {
			corev1.SchemeGroupVersion.String(): limitrangecorev1.RegisterLimitRangeCollector,
		},
		"nodes": {
			corev1.SchemeGroupVersion.String(): nodecorev1.RegisterNodeCollector,
		},
		"pods": {
			corev1.SchemeGroupVersion.String(): podcorev1.RegisterPodCollector,
		},
		"replicasets": {
			extensionsv1beta1.SchemeGroupVersion.String(): replicasetextensionsv1beta1.RegisterReplicaSetCollector,
		},
		"replicationcontrollers": {
			corev1.SchemeGroupVersion.String(): replicationcontrollercorev1.RegisterReplicationControllerCollector,
		},
		"resourcequotas": {
			corev1.SchemeGroupVersion.String(): resourcequotacorev1.RegisterResourceQuotaCollector,
		},
		"services": {
			corev1.SchemeGroupVersion.String(): servicecorev1.RegisterServiceCollector,
		},
		"statefulsets": {
			appsv1beta1.SchemeGroupVersion.String(): statefulsetappsv1beta1.RegisterStatefulSetCollector,
		},
		"persistentvolumes": {
			corev1.SchemeGroupVersion.String(): persistentvolumecorev1.RegisterPersistentVolumeCollector,
		},
		"persistentvolumeclaims": {
			corev1.SchemeGroupVersion.String(): persistentvolumeclaimcorev1.RegisterPersistentVolumeClaimCollector,
		},
		"namespaces": {
			corev1.SchemeGroupVersion.String(): namespacecorev1.RegisterNamespaceCollector,
		},
		"horizontalpodautoscalers": {
			autoscalingv1.SchemeGroupVersion.String(): hpaautoscalingv1.RegisterHorizontalPodAutoScalerCollector,
		},
		"endpoints": {
			corev1.SchemeGroupVersion.String(): endpointcorev1.RegisterEndpointCollector,
		},
	}
)

// promLogger implements promhttp.Logger
type promLogger struct{}

func (pl promLogger) Println(v ...interface{}) {
	glog.Error(v)
}

type collectorsMap map[string]string

func (c *collectorsMap) String() string {
	s := *c
	ss := s.asSlice()
	sort.Strings(ss)
	return strings.Join(ss, ",")
}

func (c *collectorsMap) Set(value string) error {
	s := *c
	cols := strings.Split(value, ",")
	for _, col := range cols {
		val, ok := supportedCollectors[col]
		if !ok {
			return fmt.Errorf("collector %q does not exist", col)
		}
		s[col] = val
	}
	return nil
}

func (c collectorsMap) asSlice() []string {
	var cols []string
	for col := range c {
		cols = append(cols, col)
	}
	return cols
}

func (c collectorsMap) isEmpty() bool {
	return len(c.asSlice()) == 0
}

func (c *collectorsMap) Type() string {
	return "string"
}

type collectorsConfigMap map[string]string

func (c *collectorsConfigMap) String() string {
	s := *c
	var cC []string
	for key, value := range s {
		cC = append(cC, strings.Join([]string{key, value}, "="))
	}

	return strings.Join(cC, ",")
}

func (c *collectorsConfigMap) Set(value string) error {
	s := *c
	cols := strings.Split(value, ",")
	for _, col := range cols {
		parts := strings.Split(col, "=")
		if len(parts) != 2 {
			return fmt.Errorf("collector config format error for %q. Must be collector=groupVersion", col)
		} else {
			collector, groupVersion := parts[0], parts[1]
			_, err := schema.ParseGroupVersion(groupVersion)
			if err != nil {
				return fmt.Errorf("specified group version %q for collector %q is invalid: %v", groupVersion, c, err)
			}
			s[collector] = groupVersion
		}
	}
	return nil
}

func (c collectorsConfigMap) asSlice() []string {
	var cols []string
	for col := range c {
		cols = append(cols, col)
	}
	return cols
}

func (c collectorsConfigMap) isEmpty() bool {
	return len(c.asSlice()) == 0
}

func (c *collectorsConfigMap) Type() string {
	return "string"
}

type options struct {
	inCluster                bool
	apiserver                string
	kubeconfig               string
	help                     bool
	port                     int
	host                     string
	telemetryPort            int
	telemetryHost            string
	collectors               collectorsMap
	collectorsConfig         collectorsConfigMap
	listRegisteredCollectors bool
	namespace                string
	version                  bool
}

func main() {
	collectorsConfig := collectorsConfigMap(map[string]string(supportedCollectors))

	options := &options{collectors: make(collectorsMap), collectorsConfig: make(collectorsConfigMap)}
	flags := pflag.NewFlagSet("", pflag.ExitOnError)
	// add glog flags
	flags.AddGoFlagSet(flag.CommandLine)
	flags.Lookup("logtostderr").Value.Set("true")
	flags.Lookup("logtostderr").DefValue = "true"
	flags.Lookup("logtostderr").NoOptDefVal = "true"
	flags.BoolVar(&options.inCluster, "in-cluster", true, `If true, use the built in kubernetes cluster for creating the client`)
	flags.BoolVar(&options.listRegisteredCollectors, "list-registered-collectors", false, `If true, list registered collectors with corresponding supported group versions`)
	flags.StringVar(&options.apiserver, "apiserver", "", `The URL of the apiserver to use as a master`)
	flags.StringVar(&options.kubeconfig, "kubeconfig", "", "Absolute path to the kubeconfig file")
	flags.BoolVarP(&options.help, "help", "h", false, "Print help text")
	flags.IntVar(&options.port, "port", 80, `Port to expose metrics on.`)
	flags.StringVar(&options.host, "host", "0.0.0.0", `Host to expose metrics on.`)
	flags.IntVar(&options.telemetryPort, "telemetry-port", 81, `Port to expose kube-state-metrics self metrics on.`)
	flags.StringVar(&options.telemetryHost, "telemetry-host", "0.0.0.0", `Host to expose kube-state-metrics self metrics on.`)
	flags.Var(&options.collectors, "collectors", fmt.Sprintf("Comma-separated list of collectors to be enabled. Defaults to %q", &supportedCollectors))
	flags.Var(&options.collectorsConfig, "collectors-config", fmt.Sprintf("Comma-separated list of collectors group version to be used. Defaults to %q", &collectorsConfig))
	flags.StringVar(&options.namespace, "namespace", metav1.NamespaceAll, "namespace to be enabled for collecting resources")
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

	if options.listRegisteredCollectors {
		for c, val := range registeredCollectors {
			fmt.Printf("%s: ", c)
			for groupVersion := range val {
				fmt.Printf("%s ", groupVersion)
			}
			fmt.Println()
		}
		os.Exit(0)
	}

	if options.help {
		flags.Usage()
		os.Exit(0)
	}

	var collectors collectorsMap
	if len(options.collectors) == 0 {
		glog.Info("Using default collectors")
		collectors = supportedCollectors
	} else {
		collectors = options.collectors
	}

	err = mergeCollectorsConfig(collectors, options.collectorsConfig)
	if err != nil {
		glog.Fatalf("Failed to merge collectors and collectors config: %v", err)
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

	supportedResourceMetaInfo, err := getServerSupportedResourceMetaInfo(kubeClient)
	if err != nil {
		glog.Fatalf("Failed to list server supported resource meta info: %v", err)
	}

	glog.V(4).Info(supportedResourceMetaInfo)

	mergeCollectorsAgainstServer(supportedResourceMetaInfo, collectors)

	ksmMetricsRegistry := prometheus.NewRegistry()
	ksmMetricsRegistry.Register(kcollectors.ResourcesPerScrapeMetric)
	ksmMetricsRegistry.Register(kcollectors.ScrapeErrorTotalMetric)
	ksmMetricsRegistry.Register(prometheus.NewProcessCollector(os.Getpid(), ""))
	ksmMetricsRegistry.Register(prometheus.NewGoCollector())
	go telemetryServer(ksmMetricsRegistry, options.telemetryHost, options.telemetryPort)

	registry := prometheus.NewRegistry()
	registerCollectors(registry, kubeClient, collectors, options.namespace)
	metricsServer(registry, options.host, options.port)
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

type resourceMetaInfo struct {
	GroupVersion schema.GroupVersion
	Verbs        sets.String
}

// getServerSupportedResourceMetaInfo queries server for the supported resource meta info about group version and verbs.
func getServerSupportedResourceMetaInfo(kubeClient clientset.Interface) (supportedResourceMetaInfo map[string][]resourceMetaInfo, err error) {
	apiRLs, err := kubeClient.Discovery().ServerResources()
	if err != nil {
		return supportedResourceMetaInfo, err
	}

	supportedResourceMetaInfo = make(map[string][]resourceMetaInfo)

	for _, arl := range apiRLs {
		groupVersion, err := schema.ParseGroupVersion(arl.GroupVersion)
		if err != nil {
			return map[string][]resourceMetaInfo{}, err
		}
		for _, ar := range arl.APIResources {
			if _, exists := supportedResourceMetaInfo[ar.Name]; !exists {
				supportedResourceMetaInfo[ar.Name] = []resourceMetaInfo{
					{
						GroupVersion: groupVersion,
						Verbs:        sets.NewString(ar.Verbs...),
					},
				}
			} else {
				supportedResourceMetaInfo[ar.Name] = append(supportedResourceMetaInfo[ar.Name], resourceMetaInfo{
					GroupVersion: groupVersion,
					Verbs:        sets.NewString(ar.Verbs...),
				})
			}
		}
	}

	return
}

// mergeCollectorsAgainstServer checks whether the specified collectors are validating to the server.
func mergeCollectorsAgainstServer(supportedResourceMetaInfo map[string][]resourceMetaInfo, enabledCollectors collectorsMap) {
	mergedCollectors := enabledCollectors

	for c, groupVersion := range enabledCollectors {
		if rmis, exists := supportedResourceMetaInfo[c]; !exists {
			glog.Errorf("Collector %q is not supported with the server", c)
			delete(mergedCollectors, c)
		} else {
			for _, rmi := range rmis {
				if rmi.GroupVersion.String() == groupVersion {
					if !rmi.Verbs.HasAll(requiredVerbs...) {
						glog.Errorf("Collector %q does not support all these verbs: %v", c, requiredVerbs)
						delete(mergedCollectors, c)
					}
				}
			}
		}
	}

	enabledCollectors = mergedCollectors
}

func mergeCollectorsConfig(enabledCollectors collectorsMap, collectorsConfig collectorsConfigMap) error {
	for c, groupVersion := range collectorsConfig {
		if _, exists := enabledCollectors[c]; exists {
			enabledCollectors[c] = groupVersion
		} else {
			return fmt.Errorf("collector %q is not enabled", c)
		}
	}

	return nil
}

// registerCollectors creates and starts informers and initializes and
// registers metrics for collection.
func registerCollectors(registry prometheus.Registerer, kubeClient clientset.Interface, enabledCollectors collectorsMap, namespace string) {
	actualCollectors := enabledCollectors

	for c, groupVersion := range enabledCollectors {
		if fns, ok := registeredCollectors[c]; ok {
			if f, exists := fns[groupVersion]; exists {
				f(registry, kubeClient, namespace)
			} else {
				glog.Errorf("Group version %q for collector %q is not registered in kube-state-metrics. Ignore collecting", groupVersion, c)
				delete(actualCollectors, c)
			}
		} else {
			glog.Errorf("Enabled collector %q is not registered in kube-state-metrics. Ignore collecting", c)
			delete(actualCollectors, c)
		}
	}

	glog.Infof("Active collectors: %s", actualCollectors)
}
