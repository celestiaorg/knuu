package k8s

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateRoleBinding creates a roleBinding
func CreateRoleBinding(
	namespace,
	name string,
	labels map[string]string,
	role,
	serviceAccount string,
) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
				Namespace: namespace,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return ErrKnuuNotInitialized
	}
	if _, err := Clientset().RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

// DeleteRoleBinding deletes a roleBinding
func DeleteRoleBinding(namespace, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return ErrKnuuNotInitialized
	}
	if err := Clientset().RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}
