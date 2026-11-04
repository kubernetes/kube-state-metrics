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

// Package runconfig holds immutable configuration inputs captured at prepare time.
package runconfig

// Prepared holds file bytes and paths captured during PrepareRun for one process run.
// Preparation logic lives in pkg/app; this package avoids an options/app import cycle.
type Prepared struct {
	OptionsConfigPath string
	OptionsConfig     []byte

	// CRSInline is set when inline custom_resource_config wins over the file path.
	CRSInline      string
	CRSFilePath    string
	CRSFileContent []byte

	KubeconfigPath    string
	KubeconfigContent []byte
	Apiserver         string
}

// Clone returns a deep copy of captured bytes.
func (p *Prepared) Clone() *Prepared {
	if p == nil {
		return nil
	}
	c := *p
	if len(p.OptionsConfig) > 0 {
		c.OptionsConfig = append([]byte(nil), p.OptionsConfig...)
	}
	if len(p.CRSFileContent) > 0 {
		c.CRSFileContent = append([]byte(nil), p.CRSFileContent...)
	}
	if p.KubeconfigContent != nil {
		c.KubeconfigContent = make([]byte, len(p.KubeconfigContent))
		copy(c.KubeconfigContent, p.KubeconfigContent)
	}
	return &c
}
