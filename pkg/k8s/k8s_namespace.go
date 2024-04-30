package k8s

import (
	"context"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespace creates a new namespace if it does not exist
func CreateNamespace(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	_, err := Clientset().CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return ErrCreatingNamespace.WithParams(name).Wrap(err)
		}
		logrus.Debugf("Namespace %s already exists, continuing.\n", name)
	}
	logrus.Debugf("Namespace %s created.\n", name)

	return nil
}

// DeleteNamespace deletes an existing namespace
func DeleteNamespace(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := Clientset().CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return ErrDeletingNamespace.WithParams(name).Wrap(err)
	}

	return nil
}

// GetNamespace retrieves an existing namespace
func GetNamespace(name string) (*corev1.Namespace, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	namespace, err := Clientset().CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, ErrGettingNamespace.WithParams(name).Wrap(err)
	}

	return namespace, nil
}

// NamespaceExists checks if a namespace exists
func NamespaceExists(name string) bool {
	_, err := GetNamespace(name)
	if err != nil {
		logrus.Debugf("Namespace %s does not exist, err: %v", name, err)
		return false
	}

	return true
}
