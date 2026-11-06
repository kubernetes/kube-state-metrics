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

package runconfig

import "testing"

func TestPreparedClone_PreservesNonNilEmptyKubeconfigContent(t *testing.T) {
	empty := []byte{}
	original := &Prepared{
		KubeconfigPath:    "/tmp/kube",
		KubeconfigContent: empty,
		Apiserver:         "https://127.0.0.1:6443",
	}
	cloned := original.Clone()
	if cloned.KubeconfigContent == nil {
		t.Fatal("expected cloned kubeconfig content to be non-nil")
	}
	if len(cloned.KubeconfigContent) != 0 {
		t.Fatalf("expected empty snapshot, got len=%d", len(cloned.KubeconfigContent))
	}
	cloned.KubeconfigContent = append(cloned.KubeconfigContent, 'x')
	if len(original.KubeconfigContent) != 0 {
		t.Fatal("clone must not share backing array with original")
	}
}
