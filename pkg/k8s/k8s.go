// Package k8s provides utility functions for working with Kubernetes clusters.
package k8s

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// tokenPath path in the filesystem to the service account token
	tokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// certPath path in the filesystem to the ca.crt
	certPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
)

type Client struct {
	clientset *kubernetes.Clientset
	namespace string
}

func New(ctx context.Context, namespace string) (*Client, error) {
	config, err := getClusterConfig()
	if err != nil {
		return nil, ErrRetrievingKubernetesConfig.Wrap(err)
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, ErrCreatingClientset.Wrap(err)
	}
	kc := &Client{clientset: cs}

	namespace = SanitizeName(namespace)
	kc.namespace = namespace
	if kc.NamespaceExists(ctx, namespace) {
		logrus.Debugf("Namespace %s already exists, continuing.\n", namespace)
		return kc, nil
	}

	if err := kc.CreateNamespace(ctx, namespace); err != nil {
		return nil, ErrCreatingNamespace.WithParams(namespace).Wrap(err)
	}

	return kc, nil
}

func (c *Client) Clientset() *kubernetes.Clientset {
	return c.clientset
}

func (c *Client) Namespace() string {
	return c.namespace
}

// isClusterEnvironment checks if the program is running in a Kubernetes cluster.
func isClusterEnvironment() bool {
	_, errToken := os.Stat(tokenPath)
	_, errCert := os.Stat(certPath)
	return errToken == nil && errCert == nil
}

// getClusterConfig returns the appropriate Kubernetes cluster configuration.
func getClusterConfig() (*rest.Config, error) {
	if isClusterEnvironment() {
		return rest.InClusterConfig()
	}

	// build the configuration from the kubeconfig file
	kubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
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
