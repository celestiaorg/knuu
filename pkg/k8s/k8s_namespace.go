package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// InitializeNamespace sets up the namespace based on the KNUU_DEDICATED_NAMESPACE environment variable
func InitializeNamespace(identifier string) (string, error) {
	namespaceName := "knuu-" + sanitizeName(identifier)
	logrus.Debugf("namespace random generated: %s", namespaceName)
	if err := createNamespace(Clientset(), namespaceName); err != nil {
		return "", fmt.Errorf("failed to create dedicated namespace: %v", err)
	}

	logrus.Debugf("full namespace name generated: %s", namespaceName)

	return namespaceName, nil
}

// createNamespace creates a new namespace if it does not exist
func createNamespace(clientset *kubernetes.Clientset, name string) error {
	ctx := context.TODO()
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			fmt.Printf("Namespace %s already exists, continuing.\n", name)
			return nil
		}
		return fmt.Errorf("error creating namespace %s: %v", name, err)
	}

	return nil
}

// sanitizeName ensures that the namespace name complies with Kubernetes restrictions
func sanitizeName(name string) string {
	sanitized := strings.ToLower(name)
	sanitized = strings.ReplaceAll(sanitized, "_", "-")

	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
	}

	return sanitized
}
