// Package k8s provides utility functions for working with Kubernetes clusters.
package k8s

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
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
	if c.terminated {
		return ErrClientTerminated
	}
	if err := validateCustomResourceName(name); err != nil {
		return err
	}
	if err := validateGroupVersionResource(gvr); err != nil {
		return err
	}
	if err := validateCustomResourceObject(obj); err != nil {
		return err
	}
	// if onj.metadata.namespace exists, use it, otherwise use the client's namespace
	namespace := c.namespace
	if obj != nil && (*obj)["metadata"] != nil && (*obj)["metadata"].(map[string]interface{})["namespace"] != nil {
		namespace = (*obj)["metadata"].(map[string]interface{})["namespace"].(string)
	}
	res := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": gvr.GroupVersion().String(),
			"kind":       gvr.Resource,
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": (*obj)["spec"],
		},
	}

	c.logger.WithFields(logrus.Fields{
		"res":       res,
		"name":      name,
		"namespace": namespace,
	}).Info("creating custom resource")

	_, err := c.dynamicClient.Resource(*gvr).Namespace(namespace).Create(ctx, res, metav1.CreateOptions{})
	if err != nil {
		return ErrCreatingCustomResource.WithParams(gvr.Resource).Wrap(err)
	}

	c.logger.WithField("name", name).Debug("customResource created")
	return nil
}

func (c *Client) CustomResourceDefinitionExists(ctx context.Context, gvr *schema.GroupVersionResource) (bool, error) {
	rsList, err := c.discoveryClient.ServerResourcesForGroupVersion(gvr.GroupVersion().String())
	if err != nil {
		if apierrs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	for _, rs := range rsList.APIResources {
		if strings.EqualFold(rs.Kind, gvr.Resource) {
			return true, nil
		}
	}

	return false, nil
}
