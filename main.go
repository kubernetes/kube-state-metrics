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
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/openshift/origin/pkg/util/proc"
	flag "github.com/spf13/pflag"
	clientset "k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/pkg/api"
	"k8s.io/client-go/1.4/pkg/api/v1"
	"k8s.io/client-go/1.4/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/1.4/pkg/runtime"
	"k8s.io/client-go/1.4/pkg/watch"
	restclient "k8s.io/client-go/1.4/rest"
	"k8s.io/client-go/1.4/tools/cache"
	"k8s.io/client-go/1.4/tools/clientcmd"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	resyncPeriod = 5 * time.Minute
	metricsPath  = "/metrics"
	healthzPath  = "/healthz"
)

var (
	flags = flag.NewFlagSet("", flag.ContinueOnError)

	inCluster = flags.Bool("in-cluster", true, `If true, use the built in kubernetes
		cluster for creating the client`)

	apiserver = flags.String("apiserver", "", `The URL of the apiserver to use as a master`)

	kubeconfig = flags.String("kubeconfig", "./config", "absolute path to the kubeconfig file")

	port = flags.Int("port", 80, `Port to expose metrics on.`)

	prefix = flag.String("prefix", "kube_", "A prefix appended to the start of all kubernetes metric names")
)

func main() {
	flags.Parse(os.Args)

	if *apiserver == "" && !(*inCluster) {
		glog.Fatalf("--apiserver not set and --in-cluster is false; apiserver must be set to a valid URL")
	}
	glog.Infof("apiServer set to: %v", *apiserver)

	proc.StartReaper()

	kubeClient, err := createKubeClient()
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	initializeMetrics(kubeClient)
	metricsServer()
}

func createKubeClient() (kubeClient clientset.Interface, err error) {
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
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		// if you want to change the loading rules (which files in which order), you can do so here
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
	glog.Infof("testing communication with server")
	_, err = kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("ERROR communicating with apiserver: %v", err)
	}

	return kubeClient, nil
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

type DeploymentLister func() ([]v1beta1.Deployment, error)

func (l DeploymentLister) List() ([]v1beta1.Deployment, error) {
	return l()
}

type PodLister func() ([]v1.Pod, error)

func (l PodLister) List() ([]v1.Pod, error) {
	return l()
}

type NodeLister func() (v1.NodeList, error)

func (l NodeLister) List() (v1.NodeList, error) {
	return l()
}

// initializeMetrics creates a new controller from the given config.
func initializeMetrics(kubeClient clientset.Interface) {
	dplStore, dplController := cache.NewNamespaceKeyedIndexerAndReflector(
		&cache.ListWatch{
			ListFunc: func(options api.ListOptions) (runtime.Object, error) {
				return kubeClient.Extensions().Deployments(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options api.ListOptions) (watch.Interface, error) {
				return kubeClient.Extensions().Deployments(api.NamespaceAll).Watch(options)
			},
		}, &v1beta1.Deployment{}, resyncPeriod)

	podStore, podController := cache.NewNamespaceKeyedIndexerAndReflector(
		cache.NewListWatchFromClient(
			kubeClient.Core().GetRESTClient(),
			"pods",
			api.NamespaceAll,
			nil,
		), &v1.Pod{}, resyncPeriod)

	nodeStore, nodeController := cache.NewNamespaceKeyedIndexerAndReflector(
		cache.NewListWatchFromClient(
			kubeClient.Core().GetRESTClient(),
			"nodes",
			api.NamespaceAll,
			nil,
		), &v1.Node{}, resyncPeriod)

	go dplController.Run()
	go podController.Run()
	go nodeController.Run()

	dplLister := DeploymentLister(func() (deployments []v1beta1.Deployment, err error) {
		for _, c := range dplStore.List() {
			deployments = append(deployments, *(c.(*v1beta1.Deployment)))
		}
		return deployments, nil
	})

	podLister := PodLister(func() (pods []v1.Pod, err error) {
		for _, m := range podStore.List() {
			pods = append(pods, *m.(*v1.Pod))
		}
		return pods, nil
	})

	nodeLister := NodeLister(func() (machines v1.NodeList, err error) {
		for _, m := range nodeStore.List() {
			machines.Items = append(machines.Items, *(m.(*v1.Node)))
		}
		return machines, nil
	})

	_ = promhttp.Handler()
	// FixMe: This change line is only there for the go compiler not to compile.
	// We first wanted to set the godeps correctly to include the promhttp in the
	// vendor directory. In the following changes I will actually use the promhttp

	prometheus.MustRegister(&deploymentCollector{store: dplLister})
	prometheus.MustRegister(&podCollector{store: podLister})
	prometheus.MustRegister(&nodeCollector{store: nodeLister})
}
