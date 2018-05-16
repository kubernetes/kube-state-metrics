/*
Copyright 2018 The Kubernetes Authors All rights reserved.

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

package options

import (
	"os"
	"sync"
	"testing"

	"github.com/spf13/pflag"
)

func TestOptionsParse(t *testing.T) {
	tests := []struct {
		Desc           string
		Args           []string
		RecoverInvoked bool
	}{
		{
			Desc:           "collectors command line argument",
			Args:           []string{"./kube-state-metrics", "--collectors=configmaps,pods"},
			RecoverInvoked: false,
		},
		{
			Desc:           "namespace command line argument",
			Args:           []string{"./kube-state-metrics", "--namespace=default,kube-system"},
			RecoverInvoked: false,
		},
	}

	for _, test := range tests {
		var wg sync.WaitGroup

		opts := NewOptions()
		opts.AddFlags()

		flags := pflag.NewFlagSet("options_test", pflag.PanicOnError)
		flags.AddFlagSet(opts.flags)

		opts.flags = flags

		os.Args = test.Args

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if err := recover(); err != nil {
					test.RecoverInvoked = true
				}
			}()

			opts.Parse()
		}()

		wg.Wait()
		if test.RecoverInvoked {
			t.Errorf("Test error for Desc: %s. Test panic", test.Desc)
		}
	}
}
