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

type resourceManager struct {
	crConfigFile *os.File
	initCrdFile  *os.File
	initCrFile   *os.File
	newCrdFile   *os.File
	newCrFile    *os.File
}

func (rm *resourceManager) createConfigAndResourceFiles(t *testing.T) {
	crConfigFile, err := os.CreateTemp("", "cr-config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rm.crConfigFile = crConfigFile

	initCrdFile, err := os.CreateTemp("", "crd.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rm.initCrdFile = initCrdFile

	initCrFile, err := os.CreateTemp("", "cr.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rm.initCrFile = initCrFile

	newCrdFile, err := os.CreateTemp("", "new-crd.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rm.newCrdFile = newCrdFile

	newCrFile, err := os.CreateTemp("", "new-cr.yaml")
	if err != nil {
		t.Fatal(err)
	}
	rm.newCrFile = newCrFile
	klog.InfoS("testdata", "crConfigFile", crConfigFile.Name(), "initCrdFile", initCrdFile.Name(), "initCrFile", initCrFile.Name(), "newCrdFile", newCrdFile.Name(), "newCrFile", newCrFile.Name())
}

func (rm *resourceManager) removeResourceFiles(t *testing.T) {
	err := os.Remove(rm.crConfigFile.Name())
	if err != nil {
		t.Fatalf("failed to remove CR config: %v", err)
	}
	err = os.Remove(rm.initCrdFile.Name())
	if err != nil {
		t.Fatalf("failed to remove initial CRD manifest: %v", err)
	}
	err = os.Remove(rm.initCrFile.Name())
	if err != nil {
		t.Fatalf("failed to remove initial CR manifest: %v", err)
	}
	err = os.Remove(rm.newCrdFile.Name())
	if err != nil {
		t.Fatalf("failed to remove new CRD manifest: %v", err)
	}
	err = os.Remove(rm.newCrFile.Name())
	if err != nil {
		t.Fatalf("failed to remove new CR manifest: %v", err)
	}
	klog.InfoS("deleted artefacts", "crConfigFile", rm.crConfigFile.Name(), "initCrdFile", rm.initCrdFile.Name(), "initCrFile", rm.initCrFile.Name(), "newCrdFile", rm.newCrdFile.Name(), "newCrFile", rm.newCrFile.Name())
}

func (rm *resourceManager) writeConfigFile(t *testing.T) {
	crConfig := getCRConfig()
	configFile := rm.crConfigFile.Name()
	err := os.WriteFile(configFile, []byte(crConfig), 0600 /* rw------- */)
	if err != nil {
		t.Fatalf("cannot write to config file: %v", err)
	}
	klog.InfoS("populated cr config file", "crConfigFile", configFile)
}

func (rm *resourceManager) writeResourceFiles(t *testing.T) {
	initCr := getCR()
	initCrd := getCRD()

	newCr := getNewCR()
	newCrd := getNewCRD()
	err := os.WriteFile(rm.initCrdFile.Name(), []byte(initCrd), 0600 /* rw------- */)
	if err != nil {
		t.Fatalf("cannot write to initial crd file: %v", err)
	}
	err = os.WriteFile(rm.initCrFile.Name(), []byte(initCr), 0600 /* rw------- */)
	if err != nil {
		t.Fatalf("cannot write to initial cr file: %v", err)
	}
	err = os.WriteFile(rm.newCrdFile.Name(), []byte(newCrd), 0600 /* rw------- */)
	if err != nil {
		t.Fatalf("cannot write to new crd file: %v", err)
	}
	err = os.WriteFile(rm.newCrFile.Name(), []byte(newCr), 0600 /* rw------- */)
	if err != nil {
		t.Fatalf("cannot write to new cr file: %v", err)
	}
	klog.InfoS("created initial and new CR and CRD manifests")
}

func TestVariableVKsDiscoveryAndResolution(t *testing.T) {
	// populateTimeout is the timeout on populating the cache for the first time.
	const populateTimeout = 10 * time.Second

	rm := &resourceManager{}
	// Create testdata.
	rm.createConfigAndResourceFiles(t)

	// Initialise options.
	opts := options.NewOptions()
	cmd := options.InitCommand
	opts.AddFlags(cmd)
	klog.InfoS("options", "options", opts)

	// Delete artefacts.
	defer rm.removeResourceFiles(t)

	// Populate options, and parse them.
	opts.CustomResourceConfigFile = rm.crConfigFile.Name()
	opts.Kubeconfig = os.Getenv("HOME") + "/.kube/config"
	if err := opts.Parse(); err != nil {
		t.Fatalf("failed to parse options: %v", err)
	}
	klog.InfoS("parsed options", "options", opts)

	// Write to the config file.
	rm.writeConfigFile(t)

	// Make the process asynchronous.
	go internal.RunKubeStateMetricsWrapper(opts)
	klog.InfoS("started KSM")

	// Wait for port 8080 to come up.
	err := wait.PollUntilContextTimeout(context.TODO(), 1*time.Second, 20*time.Second, true, func(_ context.Context) (bool, error) {
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
	rm.writeResourceFiles(t)

	// Apply initial CRD and CR to the cluster.
	err = exec.Command("kubectl", "apply", "-f", rm.initCrdFile.Name()).Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to apply initial crd: %v", err)
	}
	err = exec.Command("kubectl", "apply", "-f", rm.initCrFile.Name()).Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to apply initial cr: %v", err)
	}
	klog.InfoS("applied initial CR and CRD manifests")

	// Wait for the metric to be available.
	ch := make(chan bool, 1)
	klog.InfoS("waiting for first metrics to become available")
	testMetric := `kube_customresource_test_metric{customresource_group="contoso.com",customresource_kind="MyPlatform",customresource_version="v1alpha1",name="test-dotnet-app"}`
	err = wait.PollUntilContextTimeout(context.TODO(), discovery.Interval, populateTimeout, true, func(_ context.Context) (bool, error) {
		out, err := exec.Command("curl", "localhost:8080/metrics").Output()
		if err != nil {
			return false, err
		}
		if string(out) == "" {
			return false, nil
		}
		// Note: we use count to make sure that only one metrics handler is running
		if strings.Count(string(out), testMetric) == 1 {
			// klog.InfoS("metrics available", "metric", string(out))
			// Signal the process to exit, since we know the metrics are being generated as expected.
			ch <- true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("failed while waiting for initial metrics to be available: %v", err)
	}

	// Wait for process to exit.
	select {
	case <-ch:
		t.Log("initial metrics are available")
	case <-time.After(populateTimeout * 2):
		t.Fatal("timed out waiting for test to pass, check the logs for more info")
	}

	// Apply new CRD and CR to the cluster.
	err = exec.Command("kubectl", "apply", "-f", rm.newCrdFile.Name()).Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to apply new crd: %v", err)
	}
	err = exec.Command("kubectl", "apply", "-f", rm.newCrFile.Name()).Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to apply new cr: %v", err)
	}
	err = exec.Command("kubectl", "delete", "myplatform", "test-dotnet-app").Run() //nolint:gosec
	if err != nil {
		t.Fatalf("failed to delete myplatform resource: %v", err)
	}
	klog.InfoS("applied new CR and CRD manifests")

	// Wait for the the new metric to be available
	ch = make(chan bool, 1)
	klog.InfoS("waiting for new metrics to become available")
	testUpdateCRDMetric := `kube_customresource_test_update_crd_metric{customresource_group="contoso.com",customresource_kind="Update",customresource_version="v1",name="test-dotnet-app-update"}`
	err = wait.PollUntilContextTimeout(context.TODO(), discovery.Interval, populateTimeout, true, func(_ context.Context) (bool, error) {
		out, err := exec.Command("curl", "localhost:8080/metrics").Output()
		if err != nil {
			return false, err
		}
		if string(out) == "" {
			return false, nil
		}
		// Note: we use count to make sure that only one metrics handler is running, and we also want to validate that the
		// new metric is available and the old one was removed, otherwise, the response could come from the
		// previous handler before its context was cancelled, or maybe because it failed to be cancelled.
		if strings.Contains(string(out), testUpdateCRDMetric) && !strings.Contains(string(out), testMetric) {
			klog.InfoS("metrics available", "metric", string(out))
			// Signal the process to exit, since we know the metrics are being generated as expected.
			ch <- true
			return true, nil
		}
		klog.InfoS("metrics available", "metric", string(out))
		return false, nil
	})
	if err != nil {
		t.Fatalf("failed while waiting for new metrics to be available: %v", err)
	}

	// Wait for process to exit.
	select {
	case <-ch:
		t.Log("test passed successfully")
	case <-time.After(populateTimeout * 2):
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
        version: "v1alpha1"
        kind: "MyPlatform"
      metrics:
        - name: "test_metric"
          help: "foo baz"
          each:
            type: Info
            info:
              path: [metadata]
              labelsFromPath:
                name: [name]
    - groupVersionKind:
        group: "contoso.com"
        version: "v1"
        kind: "Update"
      metrics:
        - name: "test_update_crd_metric"
          help: "foo baz"
          each:
            type: Info
            info:
              path: [metadata]
              labelsFromPath:
                name: [name]
`
}

func getNewCR() string {
	return `
apiVersion: contoso.com/v1
kind: Update
metadata:
  name: test-dotnet-app-update
spec:
  new: just-added
`
}

func getNewCRD() string {
	return `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: updates.contoso.com
spec:
  group: contoso.com
  names:
    plural: updates
    singular: update
    kind: Update
    shortNames:
    - updt
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                new:
                  type: string
          required: ["spec"]
`
}
