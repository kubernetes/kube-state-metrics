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
	"path/filepath"
	"runtime"
	"strings"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	testUnstructuredMock "k8s.io/sample-controller/pkg/apis/samplecontroller/v1alpha1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"

	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/runconfig"
)

// BuildRESTConfig loads cluster credentials. When kubeconfigContent is non-nil it
// is used instead of re-reading kubeconfigPath, with relative cert paths resolved
// from the kubeconfig file directory.
func BuildRESTConfig(apiserver, kubeconfigPath string, kubeconfigContent []byte) (*rest.Config, error) {
	config, err := buildConfigFromSources(apiserver, kubeconfigPath, kubeconfigContent)
	if err != nil {
		return nil, err
	}
	config.UserAgent = fmt.Sprintf("%s/%s (%s/%s) kubernetes/%s", "kube-state-metrics", version.Version, runtime.GOOS, runtime.GOARCH, version.Revision)
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"
	return config, nil
}

func buildConfigFromSources(apiserver, kubeconfigPath string, kubeconfigContent []byte) (*rest.Config, error) {
	overrides := &clientcmd.ConfigOverrides{}
	if apiserver != "" {
		overrides.ClusterInfo = clientcmdapi.Cluster{Server: apiserver}
	}
	if kubeconfigContent == nil {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
		return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	}
	if len(kubeconfigContent) == 0 {
		return clientcmd.BuildConfigFromFlags(apiserver, "")
	}
	cfg, err := clientcmd.Load(kubeconfigContent)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(kubeconfigPath)
	if baseDir == "" || baseDir == "." {
		var wdErr error
		baseDir, wdErr = osGetwd()
		if wdErr != nil {
			return nil, wdErr
		}
	}
	absolutizeConfigPaths(cfg, baseDir)
	return clientcmd.NewDefaultClientConfig(*cfg, overrides).ClientConfig()
}

var osGetwd = func() (string, error) { return filepath.Abs(".") }

func absolutizeConfigPaths(cfg *clientcmdapi.Config, baseDir string) {
	for _, cluster := range cfg.Clusters {
		if cluster == nil {
			continue
		}
		if cluster.CertificateAuthority != "" && !filepath.IsAbs(cluster.CertificateAuthority) {
			cluster.CertificateAuthority = filepath.Join(baseDir, cluster.CertificateAuthority)
		}
	}
	for _, auth := range cfg.AuthInfos {
		if auth == nil {
			continue
		}
		if auth.ClientCertificate != "" && !filepath.IsAbs(auth.ClientCertificate) {
			auth.ClientCertificate = filepath.Join(baseDir, auth.ClientCertificate)
		}
		if auth.ClientKey != "" && !filepath.IsAbs(auth.ClientKey) {
			auth.ClientKey = filepath.Join(baseDir, auth.ClientKey)
		}
	}
}

func buildConfig(apiserver string, kubeconfig string) (*rest.Config, error) {
	return BuildRESTConfig(apiserver, kubeconfig, nil)
}

func buildConfigFromPrepared(prepared *runconfig.Prepared) (*rest.Config, error) {
	if prepared == nil {
		return buildConfig("", "")
	}
	content := prepared.KubeconfigContent
	return BuildRESTConfig(prepared.Apiserver, prepared.KubeconfigPath, content)
}

// CreateKubeClient creates a Kubernetes clientset and a custom resource clientset.
func CreateKubeClient(apiserver string, kubeconfig string) (clientset.Interface, error) {
	return CreateKubeClientWithPrepared(apiserver, kubeconfig, nil)
}

// CreateKubeClientWithPrepared creates a clientset using optional captured kubeconfig bytes.
func CreateKubeClientWithPrepared(apiserver, kubeconfig string, prepared *runconfig.Prepared) (clientset.Interface, error) {
	var config *rest.Config
	var err error
	if prepared != nil {
		config, err = buildConfigFromPrepared(prepared)
	} else {
		config, err = buildConfig(apiserver, kubeconfig)
	}
	if err != nil {
		return nil, err
	}

	kubeClient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	klog.InfoS("Tested communication with server")
	v, err := kubeClient.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("error while trying to communicate with apiserver: %w", err)
	}
	klog.InfoS("Run with Kubernetes cluster version", "major", v.Major, "minor", v.Minor, "gitVersion", v.GitVersion, "gitTreeState", v.GitTreeState, "gitCommit", v.GitCommit, "platform", v.Platform)
	klog.InfoS("Communication with server successful")

	return kubeClient, nil
}

// CreateCustomResourceClients creates a custom resource clientset.
func CreateCustomResourceClients(apiserver string, kubeconfig string, factories ...customresource.RegistryFactory) (map[string]any, error) {
	return CreateCustomResourceClientsWithPrepared(nil, apiserver, kubeconfig, factories...)
}

// CreateCustomResourceClientsWithPrepared creates CR clients using optional captured kubeconfig bytes.
func CreateCustomResourceClientsWithPrepared(prepared *runconfig.Prepared, apiserver string, kubeconfig string, factories ...customresource.RegistryFactory) (map[string]any, error) {
	var config *rest.Config
	var err error
	if prepared != nil {
		config, err = buildConfigFromPrepared(prepared)
	} else {
		config, err = buildConfig(apiserver, kubeconfig)
	}
	if err != nil {
		return nil, err
	}
	customResourceClients := make(map[string]any, len(factories))
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

// GVRFromType returns the GroupVersionResource for a given type.
func GVRFromType(resourceName string, expectedType any) (*schema.GroupVersionResource, error) {
	if _, ok := expectedType.(*testUnstructuredMock.Foo); ok {
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
