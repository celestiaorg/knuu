package k8s

import (
	"context"
	"fmt"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// CreateRole creates a role
func CreateRole(name, namespace string, labels map[string]string, apiGroups, resources, verbs []string) error {

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: apiGroups,
				Resources: resources,
				Verbs:     verbs,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	if _, err := Clientset().RbacV1().Roles(namespace).Create(ctx, role, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

// DeleteRole deletes a role
func DeleteRole(name, namespace string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	if err := Clientset().RbacV1().Roles(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}
