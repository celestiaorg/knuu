package k8s

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateRoleBinding(
	ctx context.Context,
	name string,
	labels map[string]string,
	role, serviceAccount string,
) error {
	if c.terminated {
		return ErrClientTerminated
	}
	if err := validateRoleBindingName(name); err != nil {
		return err
	}
	if err := validateLabels(labels); err != nil {
		return err
	}
	if err := validateRoleName(role); err != nil {
		return err
	}
	if err := validateServiceAccountName(serviceAccount); err != nil {
		return err
	}

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.namespace,
			Labels:    labels,
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "Role",
			Name: role,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: c.namespace,
			},
		},
	}

	_, err := c.clientset.RbacV1().RoleBindings(c.namespace).Create(ctx, rb, metav1.CreateOptions{})
	return err
}

func (c *Client) DeleteRoleBinding(ctx context.Context, name string) error {
	return c.clientset.RbacV1().RoleBindings(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *Client) CreateClusterRoleBinding(
	ctx context.Context,
	name string,
	labels map[string]string,
	clusterRole, serviceAccount string,
) error {
	if c.terminated {
		return ErrClientTerminated
	}
	if err := validateClusterRoleBindingName(name); err != nil {
		return err
	}
	if err := validateLabels(labels); err != nil {
		return err
	}
	if err := validateRoleName(clusterRole); err != nil {
		return err
	}
	if err := validateServiceAccountName(serviceAccount); err != nil {
		return err
	}

	_, err := c.clientset.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err == nil || !errors.IsNotFound(err) {
		return ErrClusterRoleBindingAlreadyExists.WithParams(name).Wrap(err)
	}

	role := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterRole,
			APIGroup: rbacv1.GroupName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: c.namespace,
			},
		},
	}

	_, err = c.clientset.RbacV1().ClusterRoleBindings().Create(ctx, role, metav1.CreateOptions{})
	return err
}

func (c *Client) DeleteClusterRoleBinding(ctx context.Context, name string) error {
	return c.clientset.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
}
