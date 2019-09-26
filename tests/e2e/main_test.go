/*
Copyright 2019 The Kubernetes Authors All rights reserved.

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
	"flag"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	ksmurl := flag.String(
		"ksmurl",
		"",
		"url to access the kube-state-metrics service",
	)
	flag.Parse()

	var (
		err      error
		exitCode int
	)

	if framework, err = NewFramework(*ksmurl); err != nil {
		log.Fatalf("failed to setup framework: %v\n", err)
	}

	exitCode = m.Run()

	os.Exit(exitCode)
}

func TestIsHealthz(t *testing.T) {
	ok, err := framework.KsmClient.isHealthz()
	if err != nil {
		t.Fatalf("kube-state-metrics healthz check failed: %v", err)
	}

	if ok == false {
		t.Fatal("kube-state-metrics is unhealthy")
	}
}
