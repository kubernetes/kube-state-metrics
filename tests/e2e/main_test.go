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
	"bufio"
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/prometheus/prometheus/util/promlint"
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

func TestLintMetrics(t *testing.T) {
	buf := &bytes.Buffer{}

	err := framework.KsmClient.metrics(buf)
	if err != nil {
		t.Fatalf("failed to get metrics from kube-state-metrics: %v", err)
	}

	l := promlint.New(buf)
	problems, err := l.Lint()
	if err != nil {
		t.Fatalf("failed to lint: %v", err)
	}

	if len(problems) != 0 {
		t.Fatalf("the problems encountered in Lint are: %v", problems)
	}
}

func TestDefaultCollectorMetricsAvailable(t *testing.T) {
	buf := &bytes.Buffer{}

	err := framework.KsmClient.metrics(buf)
	if err != nil {
		t.Fatalf("failed to get metrics from kube-state-metrics: %v", err)
	}

	resources := map[string]struct{}{}
	files, err := ioutil.ReadDir("../../internal/store/")
	if err != nil {
		t.Fatalf("failed to read dir to get all resouces name: %v", err)
	}

	re := regexp.MustCompile(`^([a-z]*).go$`)
	for _, file := range files {
		params := re.FindStringSubmatch(file.Name())
		if len(params) != 2 {
			continue
		}
		if params[1] == "builder" || params[1] == "utils" || params[1] == "testutils" || params[1] == "verticalpodautoscaler" {
			continue
		}
		resources[params[1]] = struct{}{}
	}

	re = regexp.MustCompile(`^kube_([a-z]*)_`)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		params := re.FindStringSubmatch(scanner.Text())
		if len(params) != 2 {
			continue
		}
		delete(resources, params[1])
	}

	err = scanner.Err()
	if err != nil {
		t.Fatalf("failed to scan metrics: %v", err)
	}

	if len(resources) != 0 {
		s := []string{}
		for k := range resources {
			s = append(s, k)
		}
		sort.Strings(s)
		t.Fatalf("failed to find metrics of resources: %s", strings.Join(s, ", "))
	}
}
