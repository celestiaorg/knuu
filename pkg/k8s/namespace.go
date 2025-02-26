package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateNamespace(ctx context.Context, name string) error {
	if c.terminated {
		return ErrClientTerminated
	}
	if err := ValidateNamespace(name); err != nil {
		return err
	}

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
		c.logger.WithField("name", name).Debug("namespace already exists, continuing")
		return nil
	}

	c.logger.WithField("name", name).Debug("namespace created")
	return nil
}

func (c *Client) DeleteNamespace(ctx context.Context, name string) error {
	err := c.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrs.IsNotFound(err) {
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
		c.logger.WithField("name", name).WithError(err).Debug("namespace does not exist")
		return false, nil
	}

	c.logger.WithField("name", name).WithError(err).Error("getting namespace")
	return false, err
}
