// Package k8s provides utility functions for working with Kubernetes clusters.
package k8s

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (c *Client) CreateCustomResource(
	ctx context.Context,
	name string,
	gvr *schema.GroupVersionResource,
	obj *unstructured.Unstructured,
) error {

	resourceUnstructured := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": gvr.GroupVersion().String(),
			"kind":       gvr.Resource,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": c.namespace,
			},
			"spec": obj.Object["spec"],
		},
	}

	if _, err := c.dynamicClient.Resource(*gvr).Namespace(c.namespace).Create(ctx, resourceUnstructured, metav1.CreateOptions{}); err != nil {
		return ErrCreatingCustomResource.WithParams(gvr.Resource).Wrap(err)
	}

	logrus.Debugf("CustomResource %s created", name)
	return nil
}

func (c *Client) CustomResourceDefinitionExists(ctx context.Context, gvr *schema.GroupVersionResource) (bool, error) {
	resourceList, err := c.discoveryClient.ServerResourcesForGroupVersion(gvr.GroupVersion().String())
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	resourceExists := false
	for _, resource := range resourceList.APIResources {
		if strings.EqualFold(resource.Kind, gvr.Resource) {
			resourceExists = true
			break
		}
	}

	return resourceExists, nil
}

func (c *Client) GetCustomResource(ctx context.Context, name string, gvr *schema.GroupVersionResource) (*unstructured.Unstructured, error) {
	return c.dynamicClient.Resource(*gvr).Namespace(c.namespace).Get(ctx, name, metav1.GetOptions{})
}
