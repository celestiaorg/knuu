// Package knuu provides the core functionality of knuu.
package knuu

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/builder/docker"
	"github.com/celestiaorg/knuu/pkg/builder/kaniko"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/minio"
)

var (
	// identifier is the identifier of the current knuu instance
	identifier   string
	startTime    string
	timeout      time.Duration
	imageBuilder builder.Builder
)

// Initialize initializes knuug
func Initialize() error {
	t := time.Now()
	identifier = fmt.Sprintf("%s-%03d", t.Format("20060102-150405"), t.Nanosecond()/1e6)
	return InitializeWithIdentifier(identifier)
}

// Identifier returns the identifier of the current knuu instance
func Identifier() string {
	return identifier
}

// InitializeWithIdentifier initializes knuu with a unique identifier
// Default timeout is 60 minutes and can be changed by setting the KNUU_TIMEOUT environment variable
func InitializeWithIdentifier(uniqueIdentifier string) error {
	if uniqueIdentifier == "" {
		return fmt.Errorf("cannot initialize knuu with empty identifier")
	}
	identifier = uniqueIdentifier

	t := time.Now()
	startTime = fmt.Sprintf("%s-%03d", t.Format("20060102-150405"), t.Nanosecond()/1e6)

	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	useDedicatedNamespaceEnv := os.Getenv("KNUU_DEDICATED_NAMESPACE")
	useDedicatedNamespace, err := strconv.ParseBool(useDedicatedNamespaceEnv)
	if err != nil {
		useDedicatedNamespace = false
	}

	logrus.Debugf("Use dedicated namespace: %t", useDedicatedNamespace)

	err = k8s.Initialize(identifier)
	if err != nil {
		return err
	}

	// read timeout from env
	timeoutString := os.Getenv("KNUU_TIMEOUT")
	if timeoutString == "" {
		timeout = 60 * time.Minute
	} else {
		parsedTimeout, err := time.ParseDuration(timeoutString)
		if err != nil {
			return fmt.Errorf("cannot parse timeout: %s", err)
		}
		timeout = parsedTimeout
	}

	if err := handleTimeout(); err != nil {
		return fmt.Errorf("cannot handle timeout: %s", err)
	}

	builderType := os.Getenv("KNUU_BUILDER")
	switch builderType {
	case "kubernetes":
		SetImageBuilder(&kaniko.Kaniko{
			K8sClientset: k8s.Clientset(),
			K8sNamespace: k8s.Namespace(),
			Minio: &minio.Minio{
				Clientset: k8s.Clientset(),
				Namespace: k8s.Namespace(),
			},
		})
	case "docker", "":
		SetImageBuilder(&docker.Docker{
			K8sClientset: k8s.Clientset(),
			K8sNamespace: k8s.Namespace(),
		})
	default:
		return fmt.Errorf("invalid KNUU_BUILDER, available [kubernetes, docker], value used: %s", builderType)
	}

	return nil
}

func SetImageBuilder(b builder.Builder) {
	imageBuilder = b
}

func ImageBuilder() builder.Builder {
	return imageBuilder
}

// IsInitialized returns true if knuu is initialized, and false otherwise
func IsInitialized() bool {
	return k8s.IsInitialized()
}

// handleTimeout creates a timeout handler that will delete all resources with the identifier after the timeout
func handleTimeout() error {
	instance, err := NewInstance("timeout-handler")
	if err != nil {
		return fmt.Errorf("cannot create instance: %s", err)
	}
	instance.instanceType = TimeoutHandlerInstance
	// FIXME: use supported kubernetes version images (use of latest could break) (https://github.com/celestiaorg/knuu/issues/116)
	if err := instance.SetImage("docker.io/bitnami/kubectl:latest"); err != nil {
		return fmt.Errorf("cannot set image: %s", err)
	}
	if err := instance.Commit(); err != nil {
		return fmt.Errorf("cannot commit instance: %s", err)
	}

	var commands []string

	// Wait for a specific period before executing the next operation.
	// This is useful to ensure that any previous operation has time to complete.
	commands = append(commands, fmt.Sprintf("sleep %d", int64(timeout.Seconds())))
	// Collects all resources (pods, services, etc.) within the specified namespace that match a specific label, excluding certain types,
	// and then deletes them. This is useful for cleaning up specific test resources before proceeding to delete the namespace.
	commands = append(commands, fmt.Sprintf("kubectl get all,pvc,netpol,roles,serviceaccounts,rolebindings,configmaps -l knuu.sh/test-run-id=%s -n %s -o json | jq -r '.items[] | select(.metadata.labels.\"knuu.sh/type\" != \"%s\") | \"\\(.kind)/\\(.metadata.name)\"' | xargs -r kubectl delete -n %s", identifier, k8s.Namespace(), TimeoutHandlerInstance.String(), k8s.Namespace()))

	// Get KNUU_DEDICATED_NAMESPACE from the environment
	useDedicatedNamespace, _ := strconv.ParseBool(os.Getenv("KNUU_DEDICATED_NAMESPACE"))

	// If KNUU_DEDICATED_NAMESPACE is true, it indicates that a dedicated namespace is being used for this run.
	// Therefore, if it is set to be deleted, this command will delete the dedicated namespace.
	// This helps ensure that all resources within the namespace are deleted, and the namespace itself as well.
	if useDedicatedNamespace {
		logrus.Debugf("The namespace generated [%s] will be deleted", k8s.Namespace())
		commands = append(commands, fmt.Sprintf("kubectl delete namespace %s", k8s.Namespace()))
	}

	// Delete all labeled resources within the namespace.
	// Unlike the previous command that excludes certain types, this command ensures that everything remaining is deleted.
	commands = append(commands, fmt.Sprintf("kubectl delete all,pvc,netpol,roles,serviceaccounts,rolebindings,configmaps -l knuu.sh/test-run-id=%s -n %s", identifier, k8s.Namespace()))

	finalCmd := strings.Join(commands, " && ")

	// Run the command
	if err := instance.SetCommand("sh", "-c", finalCmd); err != nil {
		logrus.Debugf("The full command generated is [%s]", finalCmd)
		return fmt.Errorf("cannot set command: %s", err)
	}

	rule := rbacv1.PolicyRule{
		Verbs:     []string{"*"},
		APIGroups: []string{"*"},
		Resources: []string{"*"},
	}

	if err := instance.AddPolicyRule(rule); err != nil {
		return fmt.Errorf("cannot add policy rule: %s", err)
	}
	if err := instance.Start(); err != nil {
		return fmt.Errorf("cannot start instance: %s", err)
	}

	return nil
}
