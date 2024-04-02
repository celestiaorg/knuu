package k8s

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	namespaceLengthLimit = 58 // namespaceLengthLimit K8S namespace limit, 63 - 5 (knuu-)
)

// InitializeNamespace sets up the namespace based on the KNUU_DEDICATED_NAMESPACE environment variable
func InitializeNamespace() (string, error) {
	useDedicatedNamespace, err := strconv.ParseBool(os.Getenv("KNUU_DEDICATED_NAMESPACE"))
	if err != nil {
		useDedicatedNamespace = false
	}

	var namespaceName string
	if useDedicatedNamespace {
		// namespaceName get the random name
		namespaceName = generateRandomString()
		namespaceName = "knuu-" + sanitizeName(namespaceName)

		logrus.Debugf("namespace random generated: %s", namespaceName)
		if err := createNamespace(Clientset(), namespaceName); err != nil {
			return "", fmt.Errorf("failed to create dedicated namespace: %v", err)
		}

		logrus.Debugf("full namespace name generated: %s", namespaceName)
	} else {
		namespaceName = "test"
	}

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

// generateRandomString generates a random string that meets Kubernetes name restrictions.
func generateRandomString() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789-")
	rand.Seed(time.Now().UnixNano())

	b := make([]rune, namespaceLengthLimit)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	s := string(b)

	reg := regexp.MustCompile(`^[a-z0-9](.*[a-z0-9])?$`)
	if !reg.MatchString(s) {
		return generateRandomString()
	}

	return s
}
