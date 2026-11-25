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

// Package main is the kube-state-metrics command. It is identical to the main
// package at the module root, but its import path ends in kube-state-metrics
// rather than the module's major version, so ko names the binary in the
// container image /ko-app/kube-state-metrics instead of /ko-app/v2.
package main

import "k8s.io/kube-state-metrics/v2/internal"

func main() {
	internal.Main()
}
