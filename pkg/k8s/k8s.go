// Package k8s provides utility functions for working with Kubernetes clusters.
package k8s

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// clientset is a global variable that holds a kubernetes clientset.
	clientset *kubernetes.Clientset

	// tokenPath path in the filesystem to the service account token
	tokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	// certPath path in the filesystem to the ca.crt
	certPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

// Initialize sets up the Kubernetes client.
func Initialize() error {
	k8sConfig, err := getClusterConfig()
	if err != nil {
		return ErrRetrievingKubernetesConfig.Wrap(err)
	}

	clientset, err = kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return ErrCreatingClientset.Wrap(err)
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

// precompile the regular expression to avoid recompiling it on every function call
var invalidCharsRegexp = regexp.MustCompile(`[^a-z0-9-]+`)

// SanitizeName ensures compliance with Kubernetes DNS-1123 subdomain names. It:
//  1. Converts the input string to lowercase.
//  2. Replaces underscores and any non-DNS-1123 compliant characters with hyphens.
//  3. Trims leading and trailing hyphens.
//  4. Ensures the name does not exceed 63 characters, trimming excess characters if necessary
//     and ensuring it does not end with a hyphen after trimming.
//
// Use this function to sanitize strings to be used as Kubernetes names for resources.
func SanitizeName(name string) string {
	sanitized := strings.ToLower(name)
	// Replace underscores and any other disallowed characters with hyphens
	sanitized = invalidCharsRegexp.ReplaceAllString(sanitized, "-")
	// Trim leading and trailing hyphens
	sanitized = strings.Trim(sanitized, "-")
	if len(sanitized) > 63 {
		sanitized = sanitized[:63]
		// Ensure it does not end with a hyphen after cutting it to the max length
		sanitized = strings.TrimRight(sanitized, "-")
	}
	return sanitized
}
