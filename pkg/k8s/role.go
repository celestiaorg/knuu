package k8s

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateRole(
	ctx context.Context,
	name string,
	labels map[string]string,
	policyRules []rbacv1.PolicyRule,
) error {
	if err := validateRoleName(name); err != nil {
		return err
	}
	if err := validateLabels(labels); err != nil {
		return err
	}
	if err := validatePolicyRules(policyRules); err != nil {
		return err
	}

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

func (c *Client) CreateClusterRole(
	ctx context.Context,
	name string,
	labels map[string]string,
	policyRules []rbacv1.PolicyRule,
) error {
	if err := validateClusterRoleName(name); err != nil {
		return err
	}
	if err := validateLabels(labels); err != nil {
		return err
	}
	if err := validatePolicyRules(policyRules); err != nil {
		return err
	}

	_, err := c.clientset.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
	if err == nil || !errors.IsNotFound(err) {
		return ErrClusterRoleAlreadyExists.WithParams(name).Wrap(err)
	}

	role := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Rules: policyRules,
	}
	_, err = c.clientset.RbacV1().ClusterRoles().Create(ctx, role, metav1.CreateOptions{})
	return err
}

func (c *Client) DeleteClusterRole(ctx context.Context, name string) error {
	return c.clientset.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
}
