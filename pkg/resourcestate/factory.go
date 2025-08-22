package resourcestate

import (
	"context"
	"fmt"
	klog "k8s.io/klog/v2"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	cr "k8s.io/kube-state-metrics/v2/pkg/customresource"
	crs "k8s.io/kube-state-metrics/v2/pkg/customresourcestate"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/kube-state-metrics/v2/pkg/metric"
)

type coreFactory struct {
	name string
	kind string
	gvk  metav1.GroupVersionKind // Add this field
	fams []crs.CompiledFamily
}

var _ cr.RegistryFactory = (*coreFactory)(nil)

func (f *coreFactory) Name() string { return f.name }

func (f *coreFactory) CreateClient(cfg *rest.Config) (interface{}, error) {
	return kubernetes.NewForConfig(cfg)
}

func (f *coreFactory) MetricFamilyGenerators() []generator.FamilyGenerator {
	out := make([]generator.FamilyGenerator, 0, len(f.fams))
	for _, fam := range f.fams {
		// Wrap the CRS generator to handle typed-to-unstructured conversion
		crsGen := crs.FamGen(fam)
		wrappedGen := generator.FamilyGenerator{
			Name:              crsGen.Name,
			Help:              crsGen.Help,
			Type:              crsGen.Type,
			DeprecatedVersion: crsGen.DeprecatedVersion,
			StabilityLevel:    crsGen.StabilityLevel,
			OptIn:             crsGen.OptIn,
			GenerateFunc: func(obj interface{}) *metric.Family {
				// Convert typed object to unstructured
				unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
				if err != nil {
					klog.ErrorS(err, "Failed to convert to unstructured", "objType", fmt.Sprintf("%T", obj))
					return &metric.Family{}
				}

				u := &unstructured.Unstructured{Object: unstructuredObj}
				return crsGen.GenerateFunc(u)
			},
		}
		out = append(out, wrappedGen)
	}
	return out
}

func (f *coreFactory) ExpectedType() interface{} {
	switch f.kind {
	case "Pod":
		return &corev1.Pod{}
	case "Deployment":
		return &appsv1.Deployment{}
	default:
		return &corev1.Pod{}
	}
}

func (f *coreFactory) ListWatch(client interface{}, ns, fieldSelector string) cache.ListerWatcher {
	cs := client.(kubernetes.Interface)
	ctx := context.Background()
	switch f.kind {
	case "Pod":
		return &cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				opts.FieldSelector = fieldSelector
				return cs.CoreV1().Pods(ns).List(ctx, opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				opts.FieldSelector = fieldSelector
				return cs.CoreV1().Pods(ns).Watch(ctx, opts)
			},
		}
	case "Deployment":
		return &cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				opts.FieldSelector = fieldSelector
				return cs.AppsV1().Deployments(ns).List(ctx, opts)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				opts.FieldSelector = fieldSelector
				return cs.AppsV1().Deployments(ns).Watch(ctx, opts)
			},
		}
	default:
		// Return a valid but empty ListWatch instead of nil
		return &cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				return &corev1.PodList{}, nil
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				return watch.NewEmptyWatch(), nil
			},
		}
	}
}

func (f *coreFactory) GVRString() string {
	return fmt.Sprintf("%s/%s, Resource=%s", f.gvk.Group, f.gvk.Version, strings.ToLower(f.name))
}

// BuildFactoriesFromConfig compiles the config into RegistryFactories.
func BuildFactoriesFromConfig(c *Config) ([]cr.RegistryFactory, error) {
	var out []cr.RegistryFactory
	for _, r := range c.Spec.Resources {
		fams, err := crs.Compile(r)
		if err != nil {
			return nil, err
		}
		out = append(out, &coreFactory{
			name: r.GetResourceName(),
			kind: r.GroupVersionKind.Kind,
			gvk: metav1.GroupVersionKind{
				Group:   r.GroupVersionKind.Group,
				Version: r.GroupVersionKind.Version,
				Kind:    r.GroupVersionKind.Kind,
			}, // Store the GVK
			fams: fams,
		})
	}
	return out, nil
}
