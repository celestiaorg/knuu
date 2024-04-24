package k8s

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRole creates a role
func CreateRole(
	namespace,
	name string,
	labels map[string]string,
	policyRules []rbacv1.PolicyRule,
) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Rules: policyRules,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return ErrKnuuNotInitialized
	}
	if _, err := Clientset().RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

// DeleteRole deletes a role
func DeleteRole(namespace, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return ErrKnuuNotInitialized
	}
	if err := Clientset().RbacV1().Roles(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}
