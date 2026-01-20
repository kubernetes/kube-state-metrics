package discovery

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sdiscovery "k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
)

type crdExtractor struct{}

// SourceID returns a unique identifier for the CRD.
func (e *crdExtractor) SourceID(obj interface{}) string {
	u := obj.(*unstructured.Unstructured)
	return "crd:" + u.GetName()
}

// ExtractGVKs extracts GVK information from a CRD object.
func (e *crdExtractor) ExtractGVKs(obj interface{}) []*DiscoveredResource {
	objSpec := obj.(*unstructured.Unstructured).Object["spec"].(map[string]interface{})
	var resources []*DiscoveredResource
	for _, version := range objSpec["versions"].([]interface{}) {
		g := objSpec["group"].(string)
		v := version.(map[string]interface{})["name"].(string)
		k := objSpec["names"].(map[string]interface{})["kind"].(string)
		p := objSpec["names"].(map[string]interface{})["plural"].(string)
		resources = append(resources, &DiscoveredResource{
			GroupVersionKind: schema.GroupVersionKind{
				Group:   g,
				Version: v,
				Kind:    k,
			},
			Plural: p,
		})
	}
	return resources
}

func isAPIServiceReady(obj interface{}) bool {
	status, found, err := unstructured.NestedSlice(obj.(*unstructured.Unstructured).Object, "status", "conditions")
	if err != nil || !found {
		return false
	}

	for _, condition := range status {
		conditionMap, ok := condition.(map[string]interface{})
		if !ok {
			continue // skip invalid condition
		}
		if conditionMap["type"] == "Available" && conditionMap["status"] == "True" {
			return true
		}
	}
	return false
}

type apiServiceExtractor struct {
	discoveryClient *k8sdiscovery.DiscoveryClient
}

// SourceID returns a unique identifier for the APIService.
func (e *apiServiceExtractor) SourceID(obj interface{}) string {
	u := obj.(*unstructured.Unstructured)
	return "apiservice:" + u.GetName()
}

// ExtractGVKs extracts GVK information from an APIService object.
// Returns nil if the APIService is not ready (signals "skip update").
func (e *apiServiceExtractor) ExtractGVKs(obj interface{}) []*DiscoveredResource {
	serviceSpec := obj.(*unstructured.Unstructured).Object["spec"].(map[string]interface{})
	group, _, err := unstructured.NestedString(serviceSpec, "group")
	if err != nil {
		klog.ErrorS(err, "failed to extract group from APIService")
		return nil
	}
	version, _, err := unstructured.NestedString(serviceSpec, "version")
	if err != nil {
		klog.ErrorS(err, "failed to extract version from APIService")
		return nil
	}

	// Check if APIService has a service defined - i.e. not local
	if svc, ok := serviceSpec["service"]; !ok || svc == nil {
		klog.V(5).InfoS("skipping local APIService", "group", group, "version", version)
		// Return empty slice to clear any existing resources for this source
		return []*DiscoveredResource{}
	}

	if !isAPIServiceReady(obj) {
		klog.InfoS("skipping non-ready APIService", "group", group, "version", version)
		// Return empty slice to remove resources for non-ready APIService
		return []*DiscoveredResource{}
	}

	resourceList, err := e.discoveryClient.ServerResourcesForGroupVersion(fmt.Sprintf("%s/%s", group, version))
	if err != nil {
		klog.ErrorS(err, "failed to fetch server resources for group version", "groupVersion", fmt.Sprintf("%s/%s", group, version))
		// Return nil to skip resources update
		return nil
	}

	var resources []*DiscoveredResource
	for _, resource := range resourceList.APIResources {
		// Skip subresources
		if strings.Contains(resource.Name, "/") {
			continue
		}

		resources = append(resources, &DiscoveredResource{
			GroupVersionKind: schema.GroupVersionKind{
				Group:   group,
				Version: version,
				Kind:    resource.Kind,
			},
			Plural: resource.Name,
		})
	}

	return resources
}
