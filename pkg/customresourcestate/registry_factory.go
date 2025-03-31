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
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/klog/v2"

	"k8s.io/kube-state-metrics/v2/internal/store"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
)

func compile(resource Resource) ([]compiledFamily, error) {
	var families []compiledFamily
	// Explicitly add GVK labels to all CR metrics.
	if resource.CommonLabels == nil {
		resource.CommonLabels = map[string]string{}
	}
	resource.CommonLabels[customResourceState+"_group"] = resource.GroupVersionKind.Group
	resource.CommonLabels[customResourceState+"_version"] = resource.GroupVersionKind.Version
	resource.CommonLabels[customResourceState+"_kind"] = resource.GroupVersionKind.Kind
	for _, f := range resource.Metrics {
		family, err := compileFamily(f, resource)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		families = append(families, *family)
	}
	return families, nil
}

func compileCommon(c MetricMeta) (*compiledCommon, error) {
	eachPath, err := compilePath(c.Path)
	if err != nil {
		return nil, fmt.Errorf("path: %w", err)
	}
	eachLabelsFromPath, err := compilePaths(c.LabelsFromPath)
	if err != nil {
		return nil, fmt.Errorf("labelsFromPath: %w", err)
	}
	return &compiledCommon{
		path:          eachPath,
		labelFromPath: eachLabelsFromPath,
	}, nil
}

func compileFamily(f Generator, resource Resource) (*compiledFamily, error) {
	labels := resource.Merge(f.Labels)

	if f.Each.Type == metric.Info && !strings.HasSuffix(f.Name, "_info") {
		klog.InfoS("Info metric does not have _info suffix", "gvk", resource.GroupVersionKind.String(), "name", f.Name)
	}

	metric, err := newCompiledMetric(f.Each)
	if err != nil {
		return nil, fmt.Errorf("compiling metric: %w", err)
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
		Name:          fullName(resource, f),
		ErrorLogV:     errorLogV,
		Help:          f.Help,
		Each:          metric,
		Labels:        labels.CommonLabels,
		LabelFromPath: labelsFromPath,
	}, nil
}

func fullName(resource Resource, f Generator) string {
	var parts []string
	if resource.GetMetricNamePrefix() != "" {
		parts = append(parts, resource.GetMetricNamePrefix())
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

type compiledEach compiledMetric

type compiledCommon struct {
	labelFromPath map[string]valuePath
	t             metric.Type
	path          valuePath
}

func (c compiledCommon) Path() valuePath {
	return c.path
}

func (c compiledCommon) LabelFromPath() map[string]valuePath {
	return c.labelFromPath
}

func (c compiledCommon) Type() metric.Type {
	return c.t
}

type eachValue struct {
	Labels map[string]string
	Value  float64
}

type compiledMetric interface {
	Values(v interface{}) (result []eachValue, err []error)
	Path() valuePath
	LabelFromPath() map[string]valuePath
	Type() metric.Type
}

// newCompiledMetric returns a compiledMetric depending on the given metric type.
func newCompiledMetric(m Metric) (compiledMetric, error) {
	switch m.Type {
	case metric.Gauge:
		if m.Gauge == nil {
			return nil, errors.New("expected each.gauge to not be nil")
		}
		cc, err := compileCommon(m.Gauge.MetricMeta)
		cc.t = metric.Gauge
		if err != nil {
			return nil, fmt.Errorf("each.gauge: %w", err)
		}
		valueFromPath, err := compilePath(m.Gauge.ValueFrom)
		if err != nil {
			return nil, fmt.Errorf("each.gauge.valueFrom: %w", err)
		}
		return &compiledGauge{
			compiledCommon: *cc,
			ValueFrom:      valueFromPath,
			NilIsZero:      m.Gauge.NilIsZero,
			labelFromKey:   m.Gauge.LabelFromKey,
		}, nil
	case metric.Info:
		if m.Info == nil {
			return nil, errors.New("expected each.info to not be nil")
		}
		cc, err := compileCommon(m.Info.MetricMeta)
		cc.t = metric.Info
		if err != nil {
			return nil, fmt.Errorf("each.info: %w", err)
		}
		return &compiledInfo{
			compiledCommon: *cc,
			labelFromKey:   m.Info.LabelFromKey,
		}, nil
	case metric.StateSet:
		if m.StateSet == nil {
			return nil, errors.New("expected each.stateSet to not be nil")
		}
		cc, err := compileCommon(m.StateSet.MetricMeta)
		cc.t = metric.StateSet
		if err != nil {
			return nil, fmt.Errorf("each.stateSet: %w", err)
		}
		valueFromPath, err := compilePath(m.StateSet.ValueFrom)
		if err != nil {
			return nil, fmt.Errorf("each.stateSet.valueFrom: %w", err)
		}
		return &compiledStateSet{
			compiledCommon: *cc,
			List:           m.StateSet.List,
			LabelName:      m.StateSet.LabelName,
			ValueFrom:      valueFromPath,
		}, nil
	default:
		return nil, fmt.Errorf("unknown metric type %s", m.Type)
	}
}

type compiledGauge struct {
	compiledCommon
	labelFromKey string
	ValueFrom    valuePath
	NilIsZero    bool
}

func (c *compiledGauge) Values(v interface{}) (result []eachValue, errs []error) {
	onError := func(err error) {
		errs = append(errs, fmt.Errorf("%s: %v", c.Path(), err))
	}

	switch iter := v.(type) {
	case map[string]interface{}:
		for key, it := range iter {
			// TODO: Handle multi-length valueFrom paths (https://github.com/kubernetes/kube-state-metrics/pull/1958#discussion_r1099243161).
			// Try to deduce `valueFrom`'s value from the current element.
			var ev *eachValue
			var err error
			var didResolveValueFrom bool
			// `valueFrom` will ultimately be rendered into a string and sent to the fallback in place, which also expects a string.
			// So we'll do the same and operate on the string representation of `valueFrom`'s value.
			sValueFrom := c.ValueFrom.String()
			// No comma means we're looking at a unit-length path (in an array).
			if !strings.Contains(sValueFrom, ",") &&
				sValueFrom[0] == '[' && sValueFrom[len(sValueFrom)-1] == ']' &&
				// "[...]" and not "[]".
				len(sValueFrom) > 2 {
				extractedValueFrom := sValueFrom[1 : len(sValueFrom)-1]
				if key == extractedValueFrom {
					gotFloat, err := toFloat64(it, c.NilIsZero)
					if err != nil {
						onError(fmt.Errorf("[%s]: %w", key, err))
						continue
					}
					labels := make(map[string]string)
					ev = &eachValue{
						Labels: labels,
						Value:  gotFloat,
					}
					didResolveValueFrom = true
				}
			}
			// Fallback to the regular path resolution, if we didn't manage to resolve `valueFrom`'s value.
			if !didResolveValueFrom {
				ev, err = c.value(it)
				if ev == nil {
					continue
				}
			}
			if err != nil {
				onError(fmt.Errorf("[%s]: %w", key, err))
				continue
			}
			if _, ok := ev.Labels[c.labelFromKey]; ok {
				onError(fmt.Errorf("labelFromKey (%s) generated labels conflict with labelsFromPath, consider renaming it", c.labelFromKey))
				continue
			}
			if key != "" && c.labelFromKey != "" {
				ev.Labels[c.labelFromKey] = key
			}
			addPathLabels(it, c.LabelFromPath(), ev.Labels)
			// Evaluate path from parent's context as well (search w.r.t. the root element, not just specific fields).
			addPathLabels(v, c.LabelFromPath(), ev.Labels)
			result = append(result, *ev)
		}
	case []interface{}:
		for i, it := range iter {
			value, err := c.value(it)
			if err != nil {
				onError(fmt.Errorf("[%d]: %w", i, err))
				continue
			}
			if value == nil {
				continue
			}
			addPathLabels(it, c.LabelFromPath(), value.Labels)
			result = append(result, *value)
		}
	default:
		value, err := c.value(v)
		if err != nil {
			onError(err)
			break
		}
		if value == nil {
			break
		}
		addPathLabels(v, c.LabelFromPath(), value.Labels)
		result = append(result, *value)
	}
	return
}

type compiledInfo struct {
	compiledCommon
	labelFromKey string
}

func (c *compiledInfo) Values(v interface{}) (result []eachValue, errs []error) {
	onError := func(err ...error) {
		errs = append(errs, fmt.Errorf("%s: %v", c.Path(), err))
	}

	switch iter := v.(type) {
	case []interface{}:
		for _, obj := range iter {
			ev, err := c.values(obj)
			if len(err) > 0 {
				onError(err...)
				continue
			}
			result = append(result, ev...)
		}
	case map[string]interface{}:
		value, err := c.values(v)
		if err != nil {
			onError(err...)
			break
		}
		for _, ev := range value {
			if _, ok := ev.Labels[c.labelFromKey]; ok {
				onError(fmt.Errorf("labelFromKey (%s) generated labels conflict with labelsFromPath, consider renaming it", c.labelFromKey))
				continue
			}
		}

		// labelFromKey / labelFromPath logic
		for key, it := range iter {
			labels := make(map[string]string)

			if key != "" && c.labelFromKey != "" {
				labels[c.labelFromKey] = key
			}

			addPathLabels(it, c.LabelFromPath(), labels)

			if len(labels) > 0 {
				result = append(result, eachValue{
					Labels: labels,
					Value:  1,
				})
			}
		}

		result = append(result, value...)
	default:
		result, errs = c.values(v)
	}

	return
}

func (c *compiledInfo) values(v interface{}) (result []eachValue, err []error) {
	if v == nil {
		return
	}
	value := eachValue{Value: 1, Labels: map[string]string{}}
	addPathLabels(v, c.labelFromPath, value.Labels)
	if len(value.Labels) != 0 {
		result = append(result, value)
	}
	return
}

type compiledStateSet struct {
	compiledCommon
	LabelName string
	ValueFrom valuePath
	List      []string
}

func (c *compiledStateSet) Values(v interface{}) (result []eachValue, errs []error) {
	if vs, isArray := v.([]interface{}); isArray {
		for _, obj := range vs {
			ev, err := c.values(obj)
			if len(err) > 0 {
				errs = append(errs, err...)
				continue
			}
			result = append(result, ev...)
		}
		return
	}

	return c.values(v)
}

func (c *compiledStateSet) values(v interface{}) (result []eachValue, errs []error) {
	comparable := c.ValueFrom.Get(v)
	value, ok := comparable.(string)
	if !ok {
		return []eachValue{}, []error{fmt.Errorf("%s: expected value for path to be string, got %T", c.path, comparable)}
	}

	for _, entry := range c.List {
		ev := eachValue{Value: 0, Labels: map[string]string{}}
		if value == entry {
			ev.Value = 1
		}
		ev.Labels[c.LabelName] = entry
		addPathLabels(v, c.labelFromPath, ev.Labels)
		result = append(result, ev)
	}
	return
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

func (c compiledGauge) value(it interface{}) (*eachValue, error) {
	labels := make(map[string]string)
	got := c.ValueFrom.Get(it)
	// If `valueFrom` was not resolved, respect `NilIsZero` and return.
	if got == nil {
		if c.NilIsZero {
			return &eachValue{
				Labels: labels,
				Value:  0,
			}, nil
		}
		// no it means no iterables were passed down, meaning that the path resolution never happened
		if it == nil {
			return nil, fmt.Errorf("got nil while resolving path")
		}
		// Don't error if there was not a type-casting issue (`toFloat64`).
		return nil, nil
	}
	value, err := toFloat64(got, c.NilIsZero)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.ValueFrom, err)
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
	Each          compiledEach
	Labels        map[string]string
	LabelFromPath map[string]valuePath
	Name          string
	Help          string
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
		if strings.HasPrefix(k, "*") || strings.HasSuffix(k, "*") {
			stars = append(stars, k)
		}
	}
	sort.Strings(stars)
	for _, star := range stars {
		m := labels[star].Get(obj)
		if kv, ok := m.(map[string]interface{}); ok {
			for k, v := range kv {
				if strings.HasSuffix(star, "*") {
					k = star[:len(star)-1] + k
				}
				result[store.SanitizeLabelName(k)] = fmt.Sprintf("%v", v)
			}
		}
	}
	for k, v := range labels {
		if strings.HasPrefix(k, "*") || strings.HasSuffix(k, "*") {
			continue
		}
		value := v.Get(obj)
		// skip label if value is nil
		if value == nil {
			continue
		}
		result[store.SanitizeLabelName(k)] = fmt.Sprintf("%v", value)
	}
}

type pathOp struct {
	op   func(interface{}) interface{}
	part string
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
			num, notNum := toFloat64(val, false)
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
									if i, err := toFloat64(candidate, false); err == nil && num == i {
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
						kv := strings.Split(part, "=")
						if len(kv) == 2 /* k=v */ {
							key := kv[0]
							val := kv[1]
							if v, ok := mp[key]; ok {
								if v == val {
									return v
								}
							}
						}
						return mp[part]
					} else if s, ok := m.([]interface{}); ok {
						i, err := strconv.Atoi(part)
						if err != nil {
							// This means we are here: [ <string>, <int>, ... ] (eg., [ "foo", "0", ... ], i.e., <path>.foo[0]...
							//                           ^
							// Skip over.
							return nil
						}
						if i < 0 {
							// negative index
							i += len(s)
						}
						if i < 0 || i > len(s) {
							return fmt.Errorf("list index out of range: %s", part)
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

func famGen(f compiledFamily) generator.FamilyGenerator {
	errLog := klog.V(f.ErrorLogV)
	return generator.FamilyGenerator{
		Name: f.Name,
		Type: f.Each.Type(),
		Help: f.Help,
		GenerateFunc: func(obj interface{}) *metric.Family {
			return generate(obj.(*unstructured.Unstructured), f, errLog)
		},
	}
}

// generate generates the metrics for a custom resource.
func generate(u *unstructured.Unstructured, f compiledFamily, errLog klog.Verbose) *metric.Family {
	klog.V(10).InfoS("Checked", "compiledFamilyName", f.Name, "unstructuredName", u.GetName())
	var metrics []*metric.Metric
	baseLabels := f.BaseLabels(u.Object)

	values, errors := scrapeValuesFor(f.Each, u.Object)
	for _, err := range errors {
		errLog.ErrorS(err, f.Name)
	}

	for _, v := range values {
		v.DefaultLabels(baseLabels)
		metrics = append(metrics, v.ToMetric())
	}
	klog.V(10).InfoS("Produced metrics for", "compiledFamilyName", f.Name, "metricsLength", len(metrics), "unstructuredName", u.GetName())

	return &metric.Family{
		Metrics: metrics,
	}
}

func scrapeValuesFor(e compiledEach, obj map[string]interface{}) ([]eachValue, []error) {
	v := e.Path().Get(obj)
	result, errs := e.Values(v)

	// return results in a consistent order (simplifies testing)
	sort.Slice(result, func(i, j int) bool {
		return less(result[i].Labels, result[j].Labels)
	})
	return result, errs
}

// toFloat64 converts the value to a float64 which is the value type for any metric.
func toFloat64(value interface{}, nilIsZero bool) (float64, error) {
	var v float64
	// same as bool==false but for bool pointers
	if value == nil {
		if nilIsZero {
			return 0, nil
		}
		return 0, fmt.Errorf("expected number but found nil value")
	}
	switch vv := value.(type) {
	case bool:
		if vv {
			return 1, nil
		}
		return 0, nil
	case string:
		// The string is a boolean or `"unknown"` according to https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1#Condition
		normalized := strings.ToLower(value.(string))
		if normalized == "true" || normalized == "yes" {
			return 1, nil
		}
		if normalized == "false" || normalized == "no" || normalized == "unknown" {
			return 0, nil
		}
		// The string contains a RFC3339 timestamp
		if t, e := time.Parse(time.RFC3339, value.(string)); e == nil {
			return float64(t.Unix()), nil
		}
		// The string contains a quantity with a suffix like "25m" (milli) or "5Gi" (binarySI)
		if t, e := resource.ParseQuantity(value.(string)); e == nil {
			return t.AsApproximateFloat64(), nil
		}
		// The string contains a percentage with a suffix "%"
		if e := validation.IsValidPercent(value.(string)); len(e) == 0 {
			t, e := strconv.ParseFloat(strings.TrimRight(value.(string), "%"), 64)
			return t / 100, e
		}

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
