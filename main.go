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
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/openshift/origin/pkg/util/proc"
	flag "github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/client/cache"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/controller/framework"
	kubectl_util "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	resyncPeriod = 5 * time.Minute
	metricsPath  = "/metrics"
	healthzPath  = "/healthz"
)

type metricsContainer struct {
	deploymentReplicas          *prometheus.GaugeVec
	deploymentReplicasAvailable *prometheus.GaugeVec
	nodes                       *prometheus.GaugeVec
	containerRestarts           *prometheus.GaugeVec
}

var (
	metrics metricsContainer

	// Error used to indicate that a sync is deferred because the controller isn't ready yet
	errDeferredSync = fmt.Errorf("Deferring sync until all controllers have synced.")

	flags = flag.NewFlagSet("", flag.ContinueOnError)

	inCluster = flags.Bool("in-cluster", true, `If true, use the built in kubernetes
		cluster for creating the client`)

	apiserver = flags.String("apiserver", "", `The URL of the apiserver to use as a master`)

	port = flags.Int("port", 80, `Port to expose metrics on.`)

	dryRun = flags.Bool("dry-run", false, `if set, a single dry run of configuration
		parsing is executed. Results written to stdout.`)
)

func main() {
	// Create kubernetes client.
	clientConfig := kubectl_util.DefaultClientConfig(flags)
	flags.Parse(os.Args)

	if *apiserver == "" && !(*inCluster) {
		glog.Fatalf("--apiserver not set and --in-cluster is false; apiserver must be set to a valid URL")
	}
	glog.Infof("apiServer set to: %v", *apiserver)

	proc.StartReaper()

	kubeClient, err := createKubeClient(clientConfig)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	initializeMetrics()

	// Run metrics server.
	go metricsServer()

	r := &metricsRegistryImpl{}

	mc := newMetricsController(kubeClient)
	if *dryRun {
		// Wait for the initial informer sync.
		time.Sleep(100 * time.Millisecond)
		mc.updateMetrics(r)
		err := dumpMetrics()
		if err != nil {
			glog.Fatalf("%v", err)
		}
	} else {
		// Update metrics every 10 seconds.
		wait.Until(func() {
			err := mc.updateMetrics(r)
			if err != nil {
				if err == errDeferredSync {
					glog.Infof("%v", err)
				} else {
					glog.Fatalf("%v", err)
				}
			}
		}, 10*time.Second, wait.NeverStop)
	}
}

func createKubeClient(clientConfig clientcmd.ClientConfig) (kubeClient clientset.Interface, err error) {
	glog.Infof("Creating client")
	if *inCluster {
		config, err := restclient.InClusterConfig()
		if err != nil {
			return nil, err
		}
		// Allow overriding of apiserver even if using inClusterConfig
		// (necessary if kube-proxy isn't properly set up).
		if *apiserver != "" {
			config.Host = *apiserver
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
		config, err := clientConfig.ClientConfig()
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
	glog.Infof("testing communication with server")
	_, err = kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("ERROR communicating with apiserver: %v", err)
	}

	return kubeClient, nil
}

func initializeMetrics() {
	metrics.nodes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nodes",
		Help: "Number of nodes",
	},
		[]string{
			// Whether they are reporting ready status
			"ready",
		},
	)

	metrics.containerRestarts = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "container_restarts",
		Help: "Number of container restarts per container",
	},
		[]string{
			// Name of the container
			"name",
			// Name of the pod the container is in
			"pod_name",
			// Namespace of the container/pod
			"namespace",
		},
	)

	prometheus.MustRegister(metrics.nodes)
	prometheus.MustRegister(metrics.containerRestarts)
}

// Dumps a call to /metrics to stdout. For development/testing.
func dumpMetrics() error {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", *port))
	if err != nil {
		glog.Fatalf("%v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Fatalf("%v", err)
	}
	glog.Infof("%s", body)
	return nil
}

func metricsServer() {
	// Address to listen on for web interface and telemetry
	listenAddress := fmt.Sprintf(":%d", *port)

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

// All this machinery is for mocking out the prometheus interface for testing
// because promtheus won't let us fetch a metric value after we set it.
type metricsRegistry interface {
	setReadyNodes(float64)
	setUnreadyNodes(float64)
	setContainerRestarts(string, string, string, float64)
}

type metricsRegistryImpl struct{}

func (mr *metricsRegistryImpl) setReadyNodes(count float64) {
	metrics.nodes.With(prometheus.Labels{"ready": "true"}).Set(count)
}

func (mr *metricsRegistryImpl) setUnreadyNodes(count float64) {
	metrics.nodes.With(prometheus.Labels{"ready": "false"}).Set(count)
}

func (mr *metricsRegistryImpl) setContainerRestarts(name, namespace, podName string, count float64) {
	metrics.containerRestarts.With(prometheus.Labels{
		"name": name, "namespace": namespace, "pod_name": podName,
	}).Set(count)
}

// metricsController watches the kubernetes api and adds/removes services
// from the loadbalancer, via loadBalancerConfig.
type metricsController struct {
	client         clientset.Interface
	dplController  *framework.Controller
	dplStore       cache.StoreToDeploymentLister
	podController  *framework.Controller
	podStore       cache.StoreToPodLister
	nodeController *framework.Controller
	nodeStore      cache.StoreToNodeLister
}

// sync all services with the loadbalancer.
func (mc *metricsController) updateMetrics(r metricsRegistry) error {
	if !mc.podController.HasSynced() || !mc.nodeController.HasSynced() {
		time.Sleep(100 * time.Millisecond)
		return errDeferredSync
	}

	nodes, err := mc.nodeStore.List()
	if err != nil {
		return err
	}
	registerNodeMetrics(r, nodes.Items)

	pods, err := mc.podStore.List(labels.Everything())
	if err != nil {
		return err
	}
	registerPodMetrics(r, pods)

	return nil
}

func registerNodeMetrics(r metricsRegistry, nodes []api.Node) {
	var readyNodes float64
	var unreadyNodes float64
	for _, n := range nodes {
		for _, c := range n.Status.Conditions {
			if c.Type == api.NodeReady {
				if c.Status == api.ConditionTrue {
					readyNodes += 1
				} else {
					// Even if status is unknown, call it unready.
					unreadyNodes += 1
				}
			}
		}
	}
	r.setReadyNodes(readyNodes)
	r.setUnreadyNodes(unreadyNodes)
}

func registerPodMetrics(r metricsRegistry, pods []*api.Pod) {
	for _, p := range pods {
		for _, cs := range p.Status.ContainerStatuses {
			r.setContainerRestarts(cs.Name, p.Namespace, p.Name, float64(cs.RestartCount))
		}
	}
}

// newMetricsController creates a new controller from the given config.
func newMetricsController(kubeClient clientset.Interface) *metricsController {
	mc := &metricsController{
		client: kubeClient,
	}

	mc.dplStore.Store, mc.dplController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return mc.client.Extensions().Deployments(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return mc.client.Extensions().Deployments(api.NamespaceAll).Watch(options)
			},
		}, &extensions.Deployment{}, resyncPeriod, framework.ResourceEventHandlerFuncs{})

	mc.podStore.Store, mc.podController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return mc.client.Core().Pods(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return mc.client.Core().Pods(api.NamespaceAll).Watch(options)
			},
		}, &api.Pod{}, resyncPeriod, framework.ResourceEventHandlerFuncs{})

	mc.nodeStore.Store, mc.nodeController = framework.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return mc.client.Core().Nodes().List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return mc.client.Core().Nodes().Watch(options)
			},
		}, &api.Node{}, resyncPeriod, framework.ResourceEventHandlerFuncs{})

	go mc.dplController.Run(wait.NeverStop)
	go mc.podController.Run(wait.NeverStop)
	go mc.nodeController.Run(wait.NeverStop)

	go func() {
		for !mc.dplController.HasSynced() {
			time.Sleep(100 * time.Millisecond)
		}
		prometheus.MustRegister(&deploymentCollector{
			store: &mc.dplStore,
		})
	}()

	return mc
}
