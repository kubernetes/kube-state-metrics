/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

var cr map[string]interface{}

func init() {
	type Obj map[string]interface{}
	type Array []interface{}
	bytes, err := json.Marshal(Obj{
		"spec": Obj{
			"replicas": 1,
			"order": Array{
				Obj{
					"id":    1,
					"value": true,
				},
				Obj{
					"id":    3,
					"value": false,
				},
			},
		},
		"status": Obj{
			"active": Obj{
				"type-a": 1,
				"type-b": 3,
			},
			"sub": Obj{
				"type-a": Obj{
					"active": 1,
					"ready":  2,
				},
				"type-b": Obj{
					"active": 3,
					"ready":  4,
				},
			},
			"uptime": 43.21,
			"conditions": Array{
				Obj{
					"name":  "a",
					"value": 45,
				},
				Obj{
					"name":  "b",
					"value": 66,
				},
			},
		},
		"metadata": Obj{
			"name": "foo",
			"labels": Obj{
				"foo": "bar",
			},
			"annotations": Obj{
				"qux": "quxx",
				"bar": "baz",
			},
		},
	})
	if err != nil {
		panic(err)
	}
	_ = json.Unmarshal(bytes, &cr)
}

func Test_addPathLabels(t *testing.T) {
	type args struct {
		obj    interface{}
		labels map[string]valuePath
		want   map[string]string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "all", args: args{
			obj: cr,
			labels: map[string]valuePath{
				"bool":   mustCompilePath(t, "spec", "order", "-1", "value"),
				"number": mustCompilePath(t, "spec", "replicas"),
				"string": mustCompilePath(t, "metadata", "labels", "foo"),
			},
			want: map[string]string{
				"bool":   "false",
				"number": "1",
				"string": "bar",
			},
		}},
		{name: "*", args: args{
			obj: cr,
			labels: map[string]valuePath{
				"*1":  mustCompilePath(t, "metadata", "annotations"),
				"bar": mustCompilePath(t, "metadata", "labels", "foo"),
			},
			want: map[string]string{
				"qux": "quxx",
				"bar": "bar",
			},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := make(map[string]string)
			addPathLabels(tt.args.obj, tt.args.labels, m)
			assert.Equal(t, tt.args.want, m)
		})
	}
}

func Test_compiledEach_Values(t *testing.T) {
	type tc struct {
		name       string
		each       compiledEach
		wantResult []eachValue
		wantErrors []error
	}
	val := func(value float64, labels ...string) eachValue {
		t.Helper()
		if len(labels)%2 != 0 {
			t.Fatalf("labels must be even: %v", labels)
		}
		m := make(map[string]string)
		for i := 0; i < len(labels); i += 2 {
			m[labels[i]] = labels[i+1]
		}
		return eachValue{
			Value:  value,
			Labels: m,
		}
	}

	tests := []tc{
		{name: "single", each: compiledEach{
			Path: mustCompilePath(t, "spec", "replicas"),
		}, wantResult: []eachValue{val(1)}},
		{name: "obj", each: compiledEach{
			Path:         mustCompilePath(t, "status", "active"),
			LabelFromKey: "type",
		}, wantResult: []eachValue{
			val(1, "type", "type-a"),
			val(3, "type", "type-b"),
		}},
		{name: "deep obj", each: compiledEach{
			Path:         mustCompilePath(t, "status", "sub"),
			LabelFromKey: "type",
			ValueFrom:    mustCompilePath(t, "ready"),
			LabelFromPath: map[string]valuePath{
				"active": mustCompilePath(t, "active"),
			},
		}, wantResult: []eachValue{
			val(2, "type", "type-a", "active", "1"),
			val(4, "type", "type-b", "active", "3"),
		}},
		{name: "array", each: compiledEach{
			Path:      mustCompilePath(t, "status", "conditions"),
			ValueFrom: mustCompilePath(t, "value"),
			LabelFromPath: map[string]valuePath{
				"name": mustCompilePath(t, "name"),
			},
		}, wantResult: []eachValue{
			val(45, "name", "a"),
			val(66, "name", "b"),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotErrors := tt.each.Values(cr)
			assert.Equal(t, tt.wantResult, gotResult)
			assert.Equal(t, tt.wantErrors, gotErrors)
		})
	}
}

func Test_compiledFamily_BaseLabels(t *testing.T) {
	tests := []struct {
		name   string
		fields compiledFamily
		want   map[string]string
	}{
		{name: "both", fields: compiledFamily{
			Labels: map[string]string{
				"hello": "world",
			},
			LabelFromPath: map[string]valuePath{
				"foo": mustCompilePath(t, "metadata", "annotations", "bar"),
			},
		}, want: map[string]string{
			"hello": "world",
			"foo":   "baz",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.fields
			if got := f.BaseLabels(cr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BaseLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_eachValue_DefaultLabels(t *testing.T) {
	tests := []struct {
		name     string
		Labels   map[string]string
		defaults map[string]string
		want     map[string]string
	}{
		{name: "all", Labels: map[string]string{
			"foo": "bar",
		}, defaults: map[string]string{
			"foo": "baz",
			"baz": "quxx",
		}, want: map[string]string{
			"foo": "bar",
			"baz": "quxx",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := eachValue{
				Labels: tt.Labels,
			}
			e.DefaultLabels(tt.defaults)
			assert.Equal(t, tt.want, e.Labels)
		})
	}
}

func Test_eachValue_ToMetric(t *testing.T) {
	assert.Equal(t, &metric.Metric{
		Value:       123,
		LabelKeys:   []string{"bar", "foo", "quxx"},
		LabelValues: []string{"baz", "bar", "qux"},
	}, eachValue{
		Labels: map[string]string{
			"foo":  "bar",
			"bar":  "baz",
			"quxx": "qux",
		},
		Value: 123,
	}.ToMetric())
}

func Test_fullName(t *testing.T) {
	type args struct {
		resource Resource
		f        Generator
	}
	count := Generator{
		Name: "count",
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "defaults",
			args: args{
				resource: r("", ""),
				f:        count,
			},
			want: "kube_apps_v1_Deployment_count",
		},
		{
			name: "_",
			args: args{
				resource: r("_", "_"),
				f:        count,
			},
			want: "count",
		},
		{
			name: "_namespace",
			args: args{
				resource: r("_", ""),
				f:        count,
			},
			want: "apps_v1_Deployment_count",
		},
		{
			name: "_subsystem",
			args: args{
				resource: r("", "_"),
				f:        count,
			},
			want: "kube_count",
		},
		{
			name: "custom",
			args: args{
				resource: r("foo", "bar_baz"),
				f:        count,
			},
			want: "foo_bar_baz_count",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fullName(tt.args.resource, tt.args.f); got != tt.want {
				t.Errorf("fullName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func r(namespace, subsystem string) Resource {
	return Resource{Namespace: namespace, Subsystem: subsystem, GroupVersionKind: gkv("apps", "v1", "Deployment")}
}

func gkv(group, version, kind string) GroupVersionKind {
	return GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    kind,
	}
}

func Test_valuePath_Get(t *testing.T) {

	type testCase struct {
		name string
		p    []string
		want interface{}
	}
	tt := func(name string, want interface{}, path ...string) testCase {
		return testCase{
			name: name,
			p:    path,
			want: want,
		}
	}
	tests := []testCase{
		tt("obj", float64(1), "spec", "replicas"),
		tt("array", float64(66), "status", "conditions", "[name=b]", "value"),
		tt("array index", true, "spec", "order", "0", "value"),
		tt("string", "bar", "metadata", "labels", "foo"),
		tt("match number", false, "spec", "order", "[id=3]", "value"),
		tt("match bool", float64(3), "spec", "order", "[value=false]", "id"),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := mustCompilePath(t, tt.p...)
			assert.Equal(t, tt.want, p.Get(cr))
		})
	}
}

func mustCompilePath(t *testing.T, path ...string) valuePath {
	t.Helper()
	out, err := compilePath(path)
	if err != nil {
		t.Fatalf("path %v: %v", path, err)
	}
	return out
}
