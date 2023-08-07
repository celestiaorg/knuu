package k8s

import (
	"time"
)

// namespace is the current namespace in use by the Kubernetes client.
var namespace = ""

// timeout is the timeout for Kubernetes operations
const timeout = 20 * time.Second

// Namespace returns the current namespace in use.
func Namespace() string {
	return namespace
}

// setNamespace updates the namespace to the provided string.
func setNamespace(newNamespace string) {
	namespace = newNamespace
}
