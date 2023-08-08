package k8s

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetConfigMap retrieves a configmap
func GetConfigMap(namespace, name string) (*v1.ConfigMap, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	cm, err := Clientset().CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting configmap %s: %w", name, err)
	}

	return cm, nil
}

// ConfigMapExists checks if a configmap exists
func ConfigMapExists(namespace, name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return false, fmt.Errorf("knuu is not initialized")
	}
	_, err := Clientset().CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("error getting configmap %s: %w", name, err)
	}
	return true, nil
}

// CreateConfigMap creates a configmap
func CreateConfigMap(namespace, name string, labels map[string]string, data map[string]string) (*v1.ConfigMap, error) {

	// check if configmap exists
	exists, err := ConfigMapExists(namespace, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("configmap %s already exists", name)
	}

	cm, err := prepareConfigMap(namespace, name, labels, data)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	created, err := Clientset().CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating configmap %s: %w", name, err)
	}

	return created, nil
}

// DeleteConfigMap deletes a configmap
func DeleteConfigMap(namespace, name string) error {

	// check if configmap exists
	exists, err := ConfigMapExists(namespace, name)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("configmap %s does not exist", name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	err = Clientset().CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting configmap %s: %w", name, err)
	}

	return nil
}

// prepareConfigMap prepares a configmap
func prepareConfigMap(namespace, name string, labels map[string]string, data map[string]string) (*v1.ConfigMap, error) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}
	return cm, nil
}
