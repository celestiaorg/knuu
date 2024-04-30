package k8s

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateRole(
	ctx context.Context,
	name string,
	labels map[string]string,
	policyRules []rbacv1.PolicyRule,
) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.namespace,
			Labels:    labels,
		},
		Rules: policyRules,
	}

	_, err := c.clientset.RbacV1().Roles(c.namespace).Create(ctx, role, metav1.CreateOptions{})
	return err
}

func (c *Client) DeleteRole(ctx context.Context, name string) error {
	return c.clientset.RbacV1().Roles(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
