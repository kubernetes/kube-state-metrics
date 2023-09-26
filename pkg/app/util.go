package app

import (
	"fmt"
	"runtime"

	"github.com/prometheus/common/version"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func createKubeClient(apiserver string, kubeconfig string) (clientset.Interface, error) {
	var config *rest.Config

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

	return kubeClient, nil
}
