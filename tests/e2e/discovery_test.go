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

package e2e

import (
	"context"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/internal"
	"k8s.io/kube-state-metrics/v2/internal/discovery"
	"k8s.io/kube-state-metrics/v2/pkg/options"
)

// PopulateTimeout is the timeout on populating the cache for the first time.
const PopulateTimeout = 10 * time.Second

func TestVariableVKsDiscoveryAndResolution(t *testing.T) {

	// Initialise options.
	opts := options.NewOptions()
	cmd := options.InitCommand
	opts.AddFlags(cmd)
	klog.InfoS("options", "options", opts)

	// Create testdata.
	crConfigFile, err := os.CreateTemp("", "cr-config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	crdFile, err := os.CreateTemp("", "crd.yaml")
	if err != nil {
		t.Fatal(err)
	}
	crFile, err := os.CreateTemp("", "cr.yaml")
	if err != nil {
		t.Fatal(err)
	}
	klog.InfoS("testdata", "crConfigFile", crConfigFile.Name(), "crdFile", crdFile.Name(), "crFile", crFile.Name())

	// Delete artefacts.
	defer func() {
		err := os.Remove(crConfigFile.Name())
		if err != nil {
			t.Fatalf("failed to remove CR config: %v", err)
		}
		err = os.Remove(crdFile.Name())
		if err != nil {
			t.Fatalf("failed to remove CRD manifest: %v", err)
		}
		err = os.Remove(crFile.Name())
		if err != nil {
			t.Fatalf("failed to remove CR manifest: %v", err)
		}
		klog.InfoS("deleted artefacts", "crConfigFile", crConfigFile.Name(), "crdFile", crdFile.Name(), "crFile", crFile.Name())
	}()

	// Populate options, and parse them.
	opts.CustomResourceConfigFile = crConfigFile.Name()
	opts.Kubeconfig = os.Getenv("HOME") + "/.kube/config"
	if err := opts.Parse(); err != nil {
		t.Fatalf("failed to parse options: %v", err)
	}
	klog.InfoS("parsed options", "options", opts)

	// Write to the config file.
	crConfig := getCRConfig()
	err = os.WriteFile(opts.CustomResourceConfigFile, []byte(crConfig), 0600 /* rw------- */)
	if err != nil {
		t.Fatalf("cannot write to config file: %v", err)
	}
	klog.InfoS("populated cr config file", "crConfigFile", opts.CustomResourceConfigFile)

	// Make the process asynchronous.
	go internal.RunKubeStateMetricsWrapper(opts)
	klog.InfoS("started KSM")

	// Wait for port 8080 to come up.
	err = wait.PollUntilContextTimeout(context.TODO(), 1*time.Second, 20*time.Second, true, func(ctx context.Context) (bool, error) {
		conn, err := net.Dial("tcp", "localhost:8080")
		if err != nil {
			return false, nil
		}
		err = conn.Close()
		if err != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("failed while waiting for port 8080 to come up: %v", err)
	}
	klog.InfoS("port 8080 up")

	// Create CRD and CR files.
	crd := getCRD()
	cr := getCR()
	err = os.WriteFile(crdFile.Name(), []byte(crd), 0600 /* rw------- */)
	if err != nil {
		t.Fatalf("cannot write to crd file: %v", err)
	}
	err = os.WriteFile(crFile.Name(), []byte(cr), 0600 /* rw------- */)
	if err != nil {
		t.Fatalf("cannot write to cr file: %v", err)
	}
	klog.InfoS("created CR and CRD manifests")

	// Apply CRD and CR to the cluster.
	err = exec.Command("kubectl", "apply", "-f", crdFile.Name()).Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to apply crd: %v", err)
	}
	err = exec.Command("kubectl", "apply", "-f", crFile.Name()).Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to apply cr: %v", err)
	}
	klog.InfoS("applied CR and CRD manifests")

	// Wait for the metric to be available.
	ch := make(chan bool, 1)
	klog.InfoS("waiting for metrics to become available")
	err = wait.PollUntilContextTimeout(context.TODO(), discovery.Interval, PopulateTimeout, true, func(ctx context.Context) (bool, error) {
		out, err := exec.Command("curl", "localhost:8080/metrics").Output()
		if err != nil {
			return false, err
		}
		if string(out) == "" {
			return false, nil
		}
		// Note the "{" below. This is to ensure that the metric is not in a comment.
		if strings.Contains(string(out), "kube_customresource_test_metric{") {
			klog.InfoS("metrics available", "metric", string(out))
			// Signal the process to exit, since we know the metrics are being generated as expected.
			ch <- true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("failed while waiting for metrics to be available: %v", err)
	}

	// Wait for process to exit.
	select {
	case <-ch:
		t.Log("test passed successfully")
	case <-time.After(PopulateTimeout * 2):
		t.Fatal("timed out waiting for test to pass, check the logs for more info")
	}
}

func getCR() string {
	return `
apiVersion: contoso.com/v1alpha1
kind: MyPlatform
metadata:
    name: test-dotnet-app
spec:
    appId: testdotnetapp
    language: csharp
    os: linux
    instanceSize: small
    environmentType: dev
    replicas: 3
`
}

func getCRD() string {
	return `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
 name: myplatforms.contoso.com
spec:
 group: contoso.com
 names:
   plural: myplatforms
   singular: myplatform
   kind: MyPlatform
   shortNames:
   - myp
 scope: Namespaced
 versions:
   - name: v1alpha1
     served: true
     storage: true
     schema:
       openAPIV3Schema:
         type: object
         properties:
           spec:
             type: object
             properties:
               appId:
                 type: string
               language:
                 type: string
                 enum:
                 - csharp
                 - python
                 - go
               os:
                 type: string
                 enum:
                 - windows
                 - linux
               instanceSize:
                 type: string
                 enum:
                   - small
                   - medium
                   - large
               environmentType:
                 type: string
                 enum:
                 - dev
                 - test
                 - prod
               replicas:
                 type: integer
                 minimum: 1
             required: ["appId", "language", "environmentType"]
         required: ["spec"]
`
}

func getCRConfig() string {
	return `
kind: CustomResourceStateMetrics
spec:
  resources:
    - groupVersionKind:
        group: "contoso.com"
        version: "*"
        kind: "*"
      metrics:
        - name: "test_metric"
          help: "foo baz"
          each:
            type: Info
            info:
              path: [metadata]
              labelsFromPath:
                name: [name]
`
}
