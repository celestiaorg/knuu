package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateNamespace(ctx context.Context, name string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	_, err := c.clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		if !apierrs.IsAlreadyExists(err) {
			return ErrCreatingNamespace.WithParams(name).Wrap(err)
		}
		c.logger.Debugf("Namespace %s already exists, continuing.\n", name)
	}
	c.logger.Debugf("Namespace %s created.\n", name)

	return nil
}

func (c *Client) DeleteNamespace(ctx context.Context, name string) error {
	err := c.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return ErrDeletingNamespace.WithParams(name).Wrap(err)
	}
	return nil
}

func (c *Client) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) NamespaceExists(ctx context.Context, name string) (bool, error) {
	_, err := c.GetNamespace(ctx, name)
	if err == nil {
		return true, nil
	}

	if apierrs.IsNotFound(err) {
		c.logger.Debugf("Namespace %s does not exist, err: %v", name, err)
		return false, nil
	}

	c.logger.Errorf("Error getting namespace %s, err: %v", name, err)
	return false, err
}
