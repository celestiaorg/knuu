package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateServiceAccount(ctx context.Context, name string, labels map[string]string) error {
	sa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.namespace,
			Labels:    labels,
		},
	}

	_, err := c.clientset.CoreV1().ServiceAccounts(c.namespace).Create(ctx, sa, metav1.CreateOptions{})
	return err
}

func (c *Client) DeleteServiceAccount(ctx context.Context, name string) error {
	return c.clientset.CoreV1().ServiceAccounts(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
