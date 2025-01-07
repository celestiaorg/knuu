package k8s

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// build the configuration from the kubeconfig file
var kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")

// isClusterEnvironment checks if the program is running in a Kubernetes cluster.
func isClusterEnvironment() bool {
	return fileExists(tokenPath) && fileExists(certPath)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getClusterConfig returns the appropriate Kubernetes cluster configuration.
func getClusterConfig(opts *ClientOptions) (*rest.Config, error) {
	if opts.clusterConfig != nil {
		return opts.clusterConfig, nil
	}

	if isClusterEnvironment() {
		return rest.InClusterConfig()
	}

	if opts.authToken != "" {
		return getClusterConfigWithToken(opts)
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func getClusterConfigWithToken(opts *ClientOptions) (*rest.Config, error) {
	if opts.clusterHost == "" || opts.authToken == "" || opts.cert == "" {
		return nil, ErrEmptyClusterHostOrAuthTokenOrCert
	}

	return &rest.Config{
		Host:        opts.clusterHost,
		BearerToken: opts.authToken,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: []byte(opts.cert),
		},
	}, nil
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
// It will panic if the sanitized name is empty.
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

	if len(sanitized) == 0 {
		panic("sanitized name is empty")
	}
	return sanitized
}
