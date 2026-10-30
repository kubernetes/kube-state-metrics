/*
Copyright 2026 The Kubernetes Authors All rights reserved.

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

package internal

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/kube-state-metrics/v2/pkg/options"
)

func TestCheckConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	base := options.NewOptions()
	base.Config = path
	base.Port = 8080

	for _, tc := range []struct {
		name    string
		config  *string
		wantErr bool
	}{
		{name: "missing file is left to the run"},
		{name: "valid file", config: ptr("port: 9090\n")},
		{name: "unknown key", config: ptr("prot: 9090\n"), wantErr: true},
		{name: "invalid yaml", config: ptr("port: [\n"), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.config != nil {
				if err := os.WriteFile(path, []byte(*tc.config), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if err := checkConfigFile(base); (err != nil) != tc.wantErr {
				t.Fatalf("checkConfigFile() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
	if base.Port != 8080 {
		t.Errorf("checkConfigFile must not change the base options, got port %d", base.Port)
	}

	if err := checkConfigFile(options.NewOptions()); err != nil {
		t.Errorf("no config file: expected no error, got %v", err)
	}
}

func ptr(s string) *string { return &s }
