//go:build linux

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

package proc

import (
	"sync"
	"testing"
)

// withStubs points the package at a fake pid and a launcher that only counts, so
// the test neither has to run as pid 1 nor registers a real SIGCHLD handler in
// the test binary.
func withStubs(t *testing.T, pid int) *int {
	t.Helper()

	origOnce, origGetpid, origLaunch := reaperOnce, getpid, launch
	t.Cleanup(func() {
		reaperOnce, getpid, launch = origOnce, origGetpid, origLaunch
	})

	launched := 0
	reaperOnce = new(sync.Once)
	getpid = func() int { return pid }
	launch = func() { launched++ }
	return &launched
}

// RunKubeStateMetrics calls StartReaper and the config file watchers restart it
// on every reload, so repeated calls must not each strand a reaper goroutine.
func TestStartReaperLaunchesOnce(t *testing.T) {
	launched := withStubs(t, 1)

	for i := 0; i < 5; i++ {
		StartReaper()
	}

	if *launched != 1 {
		t.Errorf("reaper launched %d times, want 1", *launched)
	}
}

// Reaping is only the responsibility of the init process.
func TestStartReaperSkippedWhenNotPID1(t *testing.T) {
	launched := withStubs(t, 4242)

	StartReaper()

	if *launched != 0 {
		t.Errorf("reaper launched %d times off pid 1, want 0", *launched)
	}
}
