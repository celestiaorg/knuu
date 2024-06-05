// Package k8s provides utility functions for working with Kubernetes clusters.
package k8s

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (c *Client) CreateCustomResource(
	ctx context.Context,
	name string,
	gvr *schema.GroupVersionResource,
	obj *map[string]interface{},
) error {

	resourceUnstructured := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": gvr.GroupVersion().String(),
			"kind":       gvr.Resource,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": c.namespace,
			},
			"spec": (*obj)["spec"],
		},
	}

	if _, err := c.dynamicClient.Resource(*gvr).Namespace(c.namespace).Create(context.TODO(), resourceUnstructured, metav1.CreateOptions{}); err != nil {
		return ErrCreatingCustomResource.WithParams(gvr.Resource).Wrap(err)
	}

	logrus.Debugf("CustomResource %s created", name)
	return nil
}

func (c *Client) CustomResourceDefinitionExists(ctx context.Context, gvr *schema.GroupVersionResource) bool {
	resourceList, err := c.discoveryClient.ServerResourcesForGroupVersion(gvr.GroupVersion().String())
	if err != nil {
		return false
	}

	resourceExists := false
	for _, resource := range resourceList.APIResources {
		if strings.EqualFold(resource.Kind, gvr.Resource) {
			resourceExists = true
			break
		}
	}

	return resourceExists
}
