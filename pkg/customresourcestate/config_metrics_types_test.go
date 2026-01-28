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

package customresourcestate

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "sigs.k8s.io/yaml/goyaml.v3"
)

func Test_ValueFrom_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    ValueFrom
		wantErr bool
	}{
		{
			name: "string array to PathValueFrom",
			json: `["status", "phase"]`,
			want: ValueFrom{
				PathValueFrom: []string{"status", "phase"},
				CelExpr:       "",
			},
		},
		{
			name: "struct with pathValueFrom",
			json: `{"pathValueFrom": ["status", "replicas"]}`,
			want: ValueFrom{
				PathValueFrom: []string{"status", "replicas"},
				CelExpr:       "",
			},
		},
		{
			name: "struct with celExpr",
			json: `{"celExpr": "value * 2"}`,
			want: ValueFrom{
				PathValueFrom: nil,
				CelExpr:       "value * 2",
			},
		},
		{
			name:    "struct with both fields should error",
			json:    `{"pathValueFrom": ["status"], "celExpr": "value"}`,
			wantErr: true,
		},
		{
			name: "empty string array",
			json: `[]`,
			want: ValueFrom{
				PathValueFrom: []string{},
				CelExpr:       "",
			},
		},
		{
			name: "struct with empty pathValueFrom",
			json: `{"pathValueFrom": []}`,
			want: ValueFrom{
				PathValueFrom: []string{},
				CelExpr:       "",
			},
		},
		{
			name: "struct with empty celExpr",
			json: `{"celExpr": ""}`,
			want: ValueFrom{
				PathValueFrom: nil,
				CelExpr:       "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var vf ValueFrom
			err := json.Unmarshal([]byte(tt.json), &vf)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot specify both")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.PathValueFrom, vf.PathValueFrom)
				assert.Equal(t, tt.want.CelExpr, vf.CelExpr)
			}
		})
	}
}

func Test_ValueFrom_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    ValueFrom
		wantErr bool
	}{
		{
			name: "string array to PathValueFrom",
			yaml: `[status, phase]`,
			want: ValueFrom{
				PathValueFrom: []string{"status", "phase"},
				CelExpr:       "",
			},
		},
		{
			name: "struct with pathValueFrom",
			yaml: `
pathValueFrom: [status, replicas]
`,
			want: ValueFrom{
				PathValueFrom: []string{"status", "replicas"},
				CelExpr:       "",
			},
		},
		{
			name: "struct with celExpr",
			yaml: `
celExpr: "value * 2"
`,
			want: ValueFrom{
				PathValueFrom: nil,
				CelExpr:       "value * 2",
			},
		},
		{
			name: "struct with celExpr multiline",
			yaml: `
celExpr: |
  CELResult(
    double(value),
    {'status': 'ok'}
  )
`,
			want: ValueFrom{
				PathValueFrom: nil,
				CelExpr:       "CELResult(\n  double(value),\n  {'status': 'ok'}\n)\n",
			},
		},
		{
			name: "struct with both fields should error",
			yaml: `
pathValueFrom: [status]
celExpr: value
`,
			wantErr: true,
		},
		{
			name: "empty string array",
			yaml: `[]`,
			want: ValueFrom{
				PathValueFrom: []string{},
				CelExpr:       "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var vf ValueFrom
			err := yaml.Unmarshal([]byte(tt.yaml), &vf)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot specify both")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want.PathValueFrom, vf.PathValueFrom)
				assert.Equal(t, tt.want.CelExpr, vf.CelExpr)
			}
		})
	}
}
