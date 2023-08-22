// Package knuu provides the core functionality of knuu.
package knuu

import (
	"fmt"
	"os"
	"time"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
)

var (
	// identifier is the identifier of the current knuu instance
	identifier string
	startTime  string
	timeout    time.Duration
)

// Initialize initializes knuug
func Initialize() error {
	t := time.Now()
	identifier = fmt.Sprintf("%s_%03d", t.Format("20060102_150405"), t.Nanosecond()/1e6)
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
	startTime = fmt.Sprintf("%s_%03d", t.Format("20060102_150405"), t.Nanosecond()/1e6)

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

	err := k8s.Initialize()
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

	return nil
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
	timeoutSeconds := int64(timeout.Seconds())

	// command to wait for timeout and delete all resources with the identifier
	var command = []string{"sh", "-c"}
	// Command runs in-cluster to delete resources post-test. Chosen for simplicity over a separate Go app.
	wait := fmt.Sprintf("sleep %d", timeoutSeconds)
	deleteAllButTimeOutType := fmt.Sprintf("kubectl get all,pvc,netpol,roles,serviceaccounts,rolebindings,configmaps -l knuu.sh/test-run-id=%s -n %s -o json | jq -r '.items[] | select(.metadata.labels.\"knuu.sh/type\" != \"%s\") | \"\\(.kind)/\\(.metadata.name)\"' | xargs -r kubectl delete -n %s", identifier, k8s.Namespace(), TimeoutHandlerInstance.String(), k8s.Namespace())
	deleteAll := fmt.Sprintf("kubectl delete all,pvc,netpol,roles,serviceaccounts,rolebindings,configmaps -l knuu.sh/test-run-id=%s -n %s", identifier, k8s.Namespace())
	cmd := fmt.Sprintf("%s && %s && %s", wait, deleteAllButTimeOutType, deleteAll)
	command = append(command, cmd)

	if err := instance.SetCommand(command...); err != nil {
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
