// Package k8s provides utility functions for working with Kubernetes clusters.
package k8s

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// TODO: Clean namespace resources before running tests

// Clientset is a global variable that holds a kubernetes clientset.
var Clientset *kubernetes.Clientset

// Namespace is the current namespace in use by the Kubernetes client.
var Namespace = "default"

// Initialize sets up the Kubernetes client with the appropriate configuration.
func Initialize() {
	k8sConfig, err := getClusterConfig()
	if err != nil {
		logrus.Fatalf("Retrieving the Kubernetes config: %w", err)
	}

	Clientset, err = kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		logrus.Fatalf("Creating clientset for Kubernetes: %w", err)
	}

	if isClusterEnvironment() {
		namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			logrus.Fatalf("Reading namespace from pod's spec: %w", err)
		}
		setNamespace(string(namespaceBytes))
	} else {
		setNamespace("test")
	}
}

// GetCurrentNamespace returns the current namespace in use.
func GetCurrentNamespace() string {
	return Namespace
}

// setNamespace updates the Namespace to the provided string.
func setNamespace(newNamespace string) {
	Namespace = newNamespace
}

// isClusterEnvironment checks if the program is running in a Kubernetes cluster.
func isClusterEnvironment() bool {
	tokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"
	certPath := "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// getClusterConfig returns the appropriate Kubernetes cluster configuration.
func getClusterConfig() (*rest.Config, error) {
	if isClusterEnvironment() {
		return rest.InClusterConfig()
	}
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
