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
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/pkg/customresource"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

type fieldMetrics struct {
	Namespace        string
	Subsystem        string
	GroupVersionKind schema.GroupVersionKind
	ResourceName     string
	Families         []compiledFamily
}

// NewFieldMetrics creates a customresource.RegistryFactory from a configuration object.
func NewFieldMetrics(resource Resource) (customresource.RegistryFactory, error) {
	compiled, err := compile(resource)
	if err != nil {
		return nil, err
	}
	gvk := schema.GroupVersionKind(resource.GroupVersionKind)
	return &fieldMetrics{
		Namespace:        resource.GetNamespace(),
		Subsystem:        resource.GetSubsystem(),
		GroupVersionKind: gvk,
		Families:         compiled,
		ResourceName:     resource.GetResourceName(),
	}, nil
}

func compile(resource Resource) ([]compiledFamily, error) {
	var families []compiledFamily
	for _, f := range resource.Metrics {
		family, err := compileFamily(f, resource)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		families = append(families, *family)
	}
	return families, nil
}

func compileFamily(f Generator, resource Resource) (*compiledFamily, error) {
	labels := resource.Labels.Merge(f.Labels)
	eachPath, err := compilePath(f.Each.Path)
	if err != nil {
		return nil, fmt.Errorf("each.path: %w", err)
	}
	valuePath, err := compilePath(f.Each.ValueFrom)
	if err != nil {
		return nil, fmt.Errorf("each.valueFrom: %w", err)
	}
	eachLabelsFromPath, err := compilePaths(f.Each.LabelsFromPath)
	if err != nil {
		return nil, fmt.Errorf("each.labelsFromPath: %w", err)
	}
	labelsFromPath, err := compilePaths(labels.LabelsFromPath)
	if err != nil {
		return nil, fmt.Errorf("labelsFromPath: %w", err)
	}

	errorLogV := f.ErrorLogV
	if errorLogV == 0 {
		errorLogV = resource.ErrorLogV
	}
	return &compiledFamily{
		Name:      fullName(resource, f),
		ErrorLogV: errorLogV,
		Help:      f.Help,
		Each: compiledEach{
			Path:          eachPath,
			ValueFrom:     valuePath,
			LabelFromKey:  f.Each.LabelFromKey,
			LabelFromPath: eachLabelsFromPath,
		},
		Labels:        labels.CommonLabels,
		LabelFromPath: labelsFromPath,
	}, nil
}

func fullName(resource Resource, f Generator) string {
	var parts []string
	if resource.GetNamespace() != "" {
		parts = append(parts, resource.GetNamespace())
	}
	if resource.GetSubsystem() != "" {
		parts = append(parts, resource.GetSubsystem())
	}
	parts = append(parts, f.Name)
	return strings.Join(parts, "_")
}

func compilePaths(paths map[string][]string) (result map[string]valuePath, err error) {
	result = make(map[string]valuePath)
	for k, v := range paths {
		result[k], err = compilePath(v)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
	}
	return result, nil
}

func (s fieldMetrics) Name() string {
	return s.ResourceName
}

func (s fieldMetrics) CreateClient(cfg *rest.Config) (interface{}, error) {
	c, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return c.Resource(schema.GroupVersionResource{
		Group:    s.GroupVersionKind.Group,
		Version:  s.GroupVersionKind.Version,
		Resource: s.ResourceName,
	}), nil
}

func (s fieldMetrics) ExpectedType() interface{} {
	u := unstructured.Unstructured{}
	u.SetGroupVersionKind(s.GroupVersionKind)
	return &u
}

func (s fieldMetrics) ListWatch(customResourceClient interface{}, ns string, fieldSelector string) cache.ListerWatcher {
	api := customResourceClient.(dynamic.NamespaceableResourceInterface).Namespace(ns)
	ctx := context.Background()
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return api.List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return api.Watch(ctx, options)
		},
	}
}

type compiledEach struct {
	Path          valuePath
	ValueFrom     valuePath
	LabelFromKey  string
	LabelFromPath map[string]valuePath
}

type eachValue struct {
	Labels map[string]string
	Value  float64
}

func (e compiledEach) Values(obj map[string]interface{}) (result []eachValue, errors []error) {
	v := e.Path.Get(obj)
	onError := func(err error) {
		errors = append(errors, fmt.Errorf("%s: %v", e.Path, err))
	}
	switch iter := v.(type) {
	case map[string]interface{}:
		for key, it := range iter {
			ev, err := e.value(it)
			if err != nil {
				onError(fmt.Errorf("[%s]: %w", key, err))
				continue
			}
			if key != "" && e.LabelFromKey != "" {
				ev.Labels[e.LabelFromKey] = key
			}
			addPathLabels(it, e.LabelFromPath, ev.Labels)
			result = append(result, *ev)
		}
	case []interface{}:
		for i, it := range iter {
			value, err := e.value(it)
			if err != nil {
				onError(fmt.Errorf("[%d]: %w", i, err))
				continue
			}
			addPathLabels(it, e.LabelFromPath, value.Labels)
			result = append(result, *value)
		}
	default:
		value, err := e.value(v)
		if err != nil {
			onError(err)
			break
		}
		addPathLabels(v, e.LabelFromPath, value.Labels)
		result = append(result, *value)
	}
	// return results in a consistent order (simplifies testing)
	sort.Slice(result, func(i, j int) bool {
		return less(result[i].Labels, result[j].Labels)
	})
	return result, errors
}

// less compares two maps of labels by keys and values
func less(a, b map[string]string) bool {
	var aKeys, bKeys sort.StringSlice
	for k := range a {
		aKeys = append(aKeys, k)
	}
	for k := range b {
		bKeys = append(bKeys, k)
	}
	aKeys.Sort()
	bKeys.Sort()
	for i := 0; i < int(math.Min(float64(len(aKeys)), float64(len(bKeys)))); i++ {
		if aKeys[i] != bKeys[i] {
			return aKeys[i] < bKeys[i]
		}

		va := a[aKeys[i]]
		vb := b[bKeys[i]]
		if va == vb {
			continue
		}
		return va < vb
	}
	return len(aKeys) < len(bKeys)
}

func (e compiledEach) value(it interface{}) (*eachValue, error) {
	labels := make(map[string]string)
	value, err := getNum(e.ValueFrom.Get(it))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", e.ValueFrom, err)
	}
	return &eachValue{
		Labels: labels,
		Value:  value,
	}, nil
}

func (e eachValue) DefaultLabels(defaults map[string]string) {
	for k, v := range defaults {
		if _, ok := e.Labels[k]; !ok {
			e.Labels[k] = v
		}
	}
}
func (e eachValue) ToMetric() *metric.Metric {
	var keys, values []string
	for k := range e.Labels {
		keys = append(keys, k)
	}
	// make it deterministic
	sort.Strings(keys)
	for _, key := range keys {
		values = append(values, e.Labels[key])
	}
	return &metric.Metric{
		LabelKeys:   keys,
		LabelValues: values,
		Value:       e.Value,
	}
}

type compiledFamily struct {
	Name          string
	Help          string
	Each          compiledEach
	Labels        map[string]string
	LabelFromPath map[string]valuePath
	ErrorLogV     klog.Level
}

func (f compiledFamily) BaseLabels(obj map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range f.Labels {
		result[k] = v
	}
	addPathLabels(obj, f.LabelFromPath, result)
	return result
}

func addPathLabels(obj interface{}, labels map[string]valuePath, result map[string]string) {
	// *prefixed is a special case, it means copy an object
	// always do that first so other labels can override
	var stars []string
	for k := range labels {
		if strings.HasPrefix(k, "*") {
			stars = append(stars, k)
		}
	}
	sort.Strings(stars)
	for _, k := range stars {
		m := labels[k].Get(obj)
		if kv, ok := m.(map[string]interface{}); ok {
			for k, v := range kv {
				result[k] = fmt.Sprintf("%v", v)
			}
		}
	}
	for k, v := range labels {
		if strings.HasPrefix(k, "*") {
			continue
		}
		result[k] = fmt.Sprintf("%v", v.Get(obj))
	}
}

type pathOp struct {
	part string
	op   func(interface{}) interface{}
}

type valuePath []pathOp

func (p valuePath) Get(obj interface{}) interface{} {
	for _, op := range p {
		if obj == nil {
			return nil
		}
		obj = op.op(obj)
	}
	return obj
}

func (p valuePath) String() string {
	var b strings.Builder
	b.WriteRune('[')
	for i, op := range p {
		if i > 0 {
			b.WriteRune(',')
		}
		b.WriteString(op.part)
	}
	b.WriteRune(']')
	return b.String()
}

func compilePath(path []string) (out valuePath, _ error) {
	for i := range path {
		part := path[i]
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			// list lookup: [key=value]
			eq := strings.SplitN(part[1:len(part)-1], "=", 2)
			if len(eq) != 2 {
				return nil, fmt.Errorf("invalid list lookup: %s", part)
			}
			key, val := eq[0], eq[1]
			num, notNum := getNum(val)
			boolVal, notBool := strconv.ParseBool(val)
			out = append(out, pathOp{
				part: part,
				op: func(m interface{}) interface{} {
					if s, ok := m.([]interface{}); ok {
						for _, v := range s {
							if m, ok := v.(map[string]interface{}); ok {
								candidate, set := m[key]
								if !set {
									continue
								}

								if candidate == val {
									return m
								}

								if notNum == nil {
									if i, err := getNum(candidate); err == nil && num == i {
										return m
									}
								}

								if notBool == nil {
									if v, ok := candidate.(bool); ok && v == boolVal {
										return m
									}
								}

							}
						}
					}
					return nil
				},
			})
		} else {
			out = append(out, pathOp{
				part: part,
				op: func(m interface{}) interface{} {
					if mp, ok := m.(map[string]interface{}); ok {
						return mp[part]
					} else if s, ok := m.([]interface{}); ok {
						i, err := strconv.Atoi(part)
						if err != nil {
							return nil
						}
						if i < 0 {
							i += len(s)
						}
						if !(0 <= i && i < len(s)) {
							return nil
						}
						return s[i]
					}
					return nil
				},
			})
		}
	}
	return out, nil
}

func (s fieldMetrics) MetricFamilyGenerators(_, _ []string) (result []generator.FamilyGenerator) {
	klog.Infof("custom resource state adding metrics: %v", s.names())
	for _, f := range s.Families {
		result = append(result, famGen(f))
	}

	return result
}

func famGen(f compiledFamily) generator.FamilyGenerator {
	errLog := klog.V(f.ErrorLogV)
	return generator.FamilyGenerator{
		Name: f.Name,
		Type: metric.Gauge,
		Help: f.Help,
		GenerateFunc: func(obj interface{}) *metric.Family {
			return generate(obj.(*unstructured.Unstructured), f, errLog)
		},
	}
}

func generate(u *unstructured.Unstructured, f compiledFamily, errLog klog.Verbose) *metric.Family {
	klog.V(10).Infof("%s: checking %s", f.Name, u.GetName())
	var metrics []*metric.Metric
	baseLabels := f.BaseLabels(u.Object)
	values, errors := f.Each.Values(u.Object)

	for _, err := range errors {
		errLog.ErrorS(err, f.Name)
	}

	for _, v := range values {
		v.DefaultLabels(baseLabels)
		metrics = append(metrics, v.ToMetric())
	}
	klog.V(10).Infof("%s: produced %d metrics for %s", f.Name, len(metrics), u.GetName())

	return &metric.Family{
		Metrics: metrics,
	}
}

func (s fieldMetrics) names() (names []string) {
	for _, family := range s.Families {
		names = append(names, family.Name)
	}
	return names
}

func getNum(value interface{}) (float64, error) {
	var v float64
	if value == nil {
		return 0, fmt.Errorf("expected number but found nil value")
	}
	switch vv := value.(type) {
	case bool:
		if vv {
			return 1, nil
		}
		return 0, nil
	case string:
		return strconv.ParseFloat(value.(string), 64)
	case byte:
		v = float64(vv)
	case int:
		v = float64(vv)
	case int32:
		v = float64(vv)
	case int64:
		v = float64(vv)
	case uint:
		v = float64(vv)
	case uint32:
		v = float64(vv)
	case uint64:
		v = float64(vv)
	case float32:
		v = float64(vv)
	case float64:
		v = vv
	default:
		return 0, fmt.Errorf("expected number but was %v", value)
	}
	return v, nil
}

var _ customresource.RegistryFactory = &fieldMetrics{}

// ConfigDecoder is for use with FromConfig.
type ConfigDecoder interface {
	Decode(v interface{}) (err error)
}

// FromConfig decodes a configuration source into a slice of customresource.RegistryFactory that are ready to use.
func FromConfig(decoder ConfigDecoder) (factories []customresource.RegistryFactory, err error) {
	var crconfig Metrics
	if err := decoder.Decode(&crconfig); err != nil {
		return nil, fmt.Errorf("failed to parse Custom Resource State metrics: %w", err)
	}
	for _, resource := range crconfig.Spec.Resources {
		factory, err := NewFieldMetrics(resource)
		if err != nil {
			return nil, fmt.Errorf("failed to create metrics factory for %s: %w", resource.GroupVersionKind, err)
		}
		factories = append(factories, factory)
	}
	return factories, nil
}
