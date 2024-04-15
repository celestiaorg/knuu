package k8s

import (
	"time"
)

// namespace is the current namespace in use by the Kubernetes client.
var namespace = ""

// timeout is the timeout for Kubernetes operations
const timeout = 20 * time.Second

func Timeout() time.Duration {
	return timeout
}

// Namespace returns the current namespace in use.
func Namespace() string {
	return namespace
}

// SetNamespace sets the used namespace to the provided string.
func SetNamespace(newNamespace string) {
	namespace = newNamespace
}
