/*
Copyright 2023 The Kubernetes Authors All rights reserved.

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

package util

import (
	"fmt"
	"runtime"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	testUnstructuredMock "k8s.io/sample-controller/pkg/apis/samplecontroller/v1alpha1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"

	"k8s.io/kube-state-metrics/v2/pkg/customresource"
)

var config *rest.Config
var currentKubeClient clientset.Interface
var currentDiscoveryClient *discovery.DiscoveryClient

// CreateKubeClient creates a Kubernetes clientset and a custom resource clientset.
func CreateKubeClient(apiserver string, kubeconfig string) (clientset.Interface, error) {
	if currentKubeClient != nil {
		return currentKubeClient, nil
	}

	var err error

	if config == nil {
		config, err = clientcmd.BuildConfigFromFlags(apiserver, kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	config.UserAgent = fmt.Sprintf("%s/%s (%s/%s) kubernetes/%s", "kube-state-metrics", version.Version, runtime.GOOS, runtime.GOARCH, version.Revision)
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"

	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	// Informers don't seem to do a good job logging error messages when it
	// can't reach the server, making debugging hard. This makes it easier to
	// figure out if apiserver is configured incorrectly.
	klog.InfoS("Tested communication with server")
	v, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("error while trying to communicate with apiserver: %w", err)
	}
	klog.InfoS("Run with Kubernetes cluster version", "major", v.Major, "minor", v.Minor, "gitVersion", v.GitVersion, "gitTreeState", v.GitTreeState, "gitCommit", v.GitCommit, "platform", v.Platform)
	klog.InfoS("Communication with server successful")

	currentKubeClient = kubeClient
	return kubeClient, nil
}

// CreateCustomResourceClients creates a custom resource clientset.
func CreateCustomResourceClients(apiserver string, kubeconfig string, factories ...customresource.RegistryFactory) (map[string]interface{}, error) {
	// Not relying on memoized clients here because the factories are subject to change.
	var err error
	if config == nil {
		config, err = clientcmd.BuildConfigFromFlags(apiserver, kubeconfig)
		if err != nil {
			return nil, err
		}
	}
	customResourceClients := make(map[string]interface{}, len(factories))
	for _, f := range factories {
		customResourceClient, err := f.CreateClient(config)
		if err != nil {
			return nil, err
		}
		gvr, err := GVRFromType(f.Name(), f.ExpectedType())
		if err != nil {
			return nil, err
		}
		var gvrString string
		if gvr != nil {
			gvrString = gvr.String()
		} else {
			gvrString = f.Name()
		}
		customResourceClients[gvrString] = customResourceClient
	}
	return customResourceClients, nil
}

// CreateDiscoveryClient creates a Kubernetes discovery client.
func CreateDiscoveryClient(apiserver string, kubeconfig string) (*discovery.DiscoveryClient, error) {
	if currentDiscoveryClient != nil {
		return currentDiscoveryClient, nil
	}
	var err error
	if config == nil {
		var err error
		config, err = clientcmd.BuildConfigFromFlags(apiserver, kubeconfig)
		if err != nil {
			return nil, err
		}
	}
	currentDiscoveryClient, err = discovery.NewDiscoveryClientForConfig(config)
	return currentDiscoveryClient, err
}

// GVRFromType returns the GroupVersionResource for a given type.
func GVRFromType(resourceName string, expectedType interface{}) (*schema.GroupVersionResource, error) {
	if _, ok := expectedType.(*testUnstructuredMock.Foo); ok {
		// testUnstructuredMock.Foo is a mock type for testing
		return nil, nil
	}
	t, err := meta.TypeAccessor(expectedType)
	if err != nil {
		return nil, fmt.Errorf("failed to get type accessor for %T: %w", expectedType, err)
	}
	apiVersion := t.GetAPIVersion()
	g, v, found := strings.Cut(apiVersion, "/")
	if !found {
		g = "core"
		v = apiVersion
	}
	r := resourceName
	return &schema.GroupVersionResource{
		Group:    g,
		Version:  v,
		Resource: r,
	}, nil
}

// GatherAndCount gathers all metrics from the provided Gatherer and counts
// them. It returns the number of metric children in all gathered metric
// families together.
func GatherAndCount(g prometheus.Gatherer) (int, error) {
	got, err := g.Gather()
	if err != nil {
		return 0, fmt.Errorf("gathering metrics failed: %w", err)
	}

	result := 0
	for _, mf := range got {
		result += len(mf.GetMetric())
	}
	return result, nil
}
