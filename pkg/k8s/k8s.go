// Package k8s provides utility functions for working with Kubernetes clusters.
package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// clientset is a global variable that holds a kubernetes clientset.
	clientset *kubernetes.Clientset

	// namespacePath path in the filesystem to the namespace name
	namespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	// tokenPath path in the filesystem to the service account token
	tokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	// certPath path in the filesystem to the ca.crt
	certPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

// Initialize sets up the Kubernetes client with the appropriate configuration.
func Initialize() error {
	k8sConfig, err := getClusterConfig()
	if err != nil {
		return fmt.Errorf("retrieving the Kubernetes config: %w", err)
	}

	clientset, err = kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return fmt.Errorf("creating clientset for Kubernetes: %w", err)
	}

	// Check if the program is running in a Kubernetes cluster environment
	if isClusterEnvironment() {
		// Read the namespace from the pod's spec
		namespaceBytes, err := os.ReadFile(namespacePath)
		if err != nil {
			return fmt.Errorf("reading namespace from pod's spec: %w", err)
		}
		setNamespace(string(namespaceBytes))
	} else {
		// Read the namespace from KNUU_NAMESPACE environment variable
		if os.Getenv("KNUU_NAMESPACE") != "" {
			setNamespace(os.Getenv("KNUU_NAMESPACE"))
		} else {
			setNamespace("test")
		}
	}
	return nil
}

// IsInitialized checks if the Kubernetes clientset has been initialized.
func IsInitialized() bool {
	return clientset != nil
}

// Clientset returns the Kubernetes clientset.
func Clientset() *kubernetes.Clientset {
	return clientset
}

// isClusterEnvironment checks if the program is running in a Kubernetes cluster.
func isClusterEnvironment() bool {
	if _, err := os.Stat(tokenPath); err != nil {
		return false
	}
	if _, err := os.Stat(certPath); err != nil {
		return false
	}

	return true
}

// getClusterConfig returns the appropriate Kubernetes cluster configuration.
func getClusterConfig() (*rest.Config, error) {
	// Check if the program is running in a Kubernetes cluster environment
	if isClusterEnvironment() {
		return rest.InClusterConfig()
	}

	// If not running in a Kubernetes cluster environment, build the configuration from the kubeconfig file
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

// isNotFound checks if the error is a NotFound error
func isNotFound(err error) bool {
	return apierrs.IsNotFound(err)
}
