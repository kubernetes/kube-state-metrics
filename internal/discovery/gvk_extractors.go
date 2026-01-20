package discovery

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sdiscovery "k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
)

type crdGVKExtractor struct{}

func (e *crdGVKExtractor) ExtractGVKs(obj interface{}) []groupVersionKindPlural {
	objSpec := obj.(*unstructured.Unstructured).Object["spec"].(map[string]interface{})
	var gvkps []groupVersionKindPlural
	for _, version := range objSpec["versions"].([]interface{}) {
		g := objSpec["group"].(string)
		v := version.(map[string]interface{})["name"].(string)
		k := objSpec["names"].(map[string]interface{})["kind"].(string)
		p := objSpec["names"].(map[string]interface{})["plural"].(string)
		gvkps = append(gvkps, groupVersionKindPlural{
			GroupVersionKind: schema.GroupVersionKind{
				Group:   g,
				Version: v,
				Kind:    k,
			},
			Plural: p,
		})
	}
	return gvkps
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

type apiServiceGVKExtractor struct {
	discoveryClient *k8sdiscovery.DiscoveryClient
}

func (e *apiServiceGVKExtractor) ExtractGVKs(obj interface{}) []groupVersionKindPlural {
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

	// check if APIService has a service defined - i.e. not local
	if svc, ok := serviceSpec["service"]; !ok || svc == nil {
		klog.V(5).InfoS("skipping local APIService", "group", group, "version", version)
		return nil
	}

	if !isAPIServiceReady(obj) {
		klog.V(5).InfoS("skipping non-ready APIService", "group", group, "version", version)
		return nil
	}

	resourceList, err := e.discoveryClient.ServerResourcesForGroupVersion(fmt.Sprintf("%s/%s", group, version))
	if err != nil {
		klog.ErrorS(err, "failed to fetch server resources for group version", "groupVersion", fmt.Sprintf("%s/%s", group, version))
		return nil
	}

	var gvkps []groupVersionKindPlural
	for _, resource := range resourceList.APIResources {
		// Skip subresources
		if strings.Contains(resource.Name, "/") {
			continue
		}

		gvkps = append(gvkps, groupVersionKindPlural{
			GroupVersionKind: schema.GroupVersionKind{
				Group:   group,
				Version: version,
				Kind:    resource.Kind,
			},
			Plural: resource.Name,
		})
	}
	return gvkps
}
