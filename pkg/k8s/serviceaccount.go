package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateServiceAccount(ctx context.Context, name string, labels map[string]string) error {
	if c.terminated {
		return ErrClientTerminated
	}
	if err := validateServiceName(name); err != nil {
		return err
	}
	if err := validateLabels(labels); err != nil {
		return err
	}

	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.namespace,
			Labels:    labels,
		},
	}

	_, err := c.clientset.CoreV1().ServiceAccounts(c.namespace).Create(ctx, sa, metav1.CreateOptions{})
	if apierrs.IsAlreadyExists(err) {
		return ErrServiceAccountAlreadyExists.WithParams(name).Wrap(err)
	}
	return err
}

func (c *Client) DeleteServiceAccount(ctx context.Context, name string) error {
	if c.terminated {
		return ErrClientTerminated
	}
	return c.clientset.CoreV1().ServiceAccounts(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
