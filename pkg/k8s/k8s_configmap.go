package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetConfigMap retrieves a configmap
func GetConfigMap(namespace, name string) (*v1.ConfigMap, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return nil, ErrKnuuNotInitialized
	}
	cm, err := Clientset().CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, ErrGettingConfigmap.WithParams(name).Wrap(err)
	}

	return cm, nil
}

// ConfigMapExists checks if a configmap exists
func ConfigMapExists(namespace, name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return false, ErrKnuuNotInitialized
	}
	_, err := Clientset().CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, ErrGettingConfigmap.WithParams(name).Wrap(err)
	}
	return true, nil
}

// CreateConfigMap creates a configmap
func CreateConfigMap(
	namespace,
	name string,
	labels,
	data map[string]string,
) (*v1.ConfigMap, error) {
	// check if configmap exists
	exists, err := ConfigMapExists(namespace, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConfigmapAlreadyExists.WithParams(name)
	}

	cm, err := prepareConfigMap(namespace, name, labels, data)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return nil, ErrKnuuNotInitialized
	}
	created, err := Clientset().CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return nil, ErrCreatingConfigmap.WithParams(name).Wrap(err)
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
		return ErrConfigmapDoesNotExist.WithParams(name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return ErrKnuuNotInitialized
	}
	err = Clientset().CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return ErrDeletingConfigmap.WithParams(name).Wrap(err)
	}

	return nil
}

// prepareConfigMap prepares a configmap
func prepareConfigMap(
	namespace,
	name string,
	labels,
	data map[string]string,
) (*v1.ConfigMap, error) {
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
