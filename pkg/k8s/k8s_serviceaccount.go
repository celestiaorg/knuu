package k8s

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// CreateServiceAccount creates a service account
func CreateServiceAccount(name, namespace string, labels map[string]string) error {

	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	if _, err := Clientset().CoreV1().ServiceAccounts(namespace).Create(ctx, serviceAccount, metav1.CreateOptions{}); err != nil {
		return err
	}

	return nil
}

// DeleteServiceAccount deletes a service account
func DeleteServiceAccount(name, namespace string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	if err := Clientset().CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}
