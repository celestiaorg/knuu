// Package knuu provides the core functionality of knuu.
package knuu

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/builder/docker"
	"github.com/celestiaorg/knuu/pkg/builder/kaniko"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/minio"
)

var (
	// testScope is the testScope of the current knuu instance
	testScope    string
	startTime    string
	timeout      time.Duration
	imageBuilder builder.Builder

	// TODO: this temporary until we refactor knuu pkg
	k8sClient *k8s.Client
)

// Initialize initializes knuu with a unique scope
func Initialize() error {
	t := time.Now()
	scope := fmt.Sprintf("%s-%03d", t.Format("20060102-150405"), t.Nanosecond()/1e6)
	return InitializeWithScope(scope)
}

// Scope returns the scope of the current knuu instance
func Scope() string {
	return testScope
}

// InitializeWithScope initializes knuu with a given scope
func InitializeWithScope(scope string) error {
	var err error
	err = godotenv.Load()
	if err != nil {
		return ErrCannotLoadEnv.Wrap(err)
	}

	if scope == "" {
		return ErrCannotInitializeKnuuWithEmptyScope
	}

	// Sanitize the scope to ensure it is a valid Kubernetes name and match the namespace
	testScope = k8s.SanitizeName(scope)

	t := time.Now()
	startTime = fmt.Sprintf("%s-%03d", t.Format("20060102-150405"), t.Nanosecond()/1e6)

	setupLogging()

	// Override scope if KNUU_NAMESPACE is set
	namespaceEnv := os.Getenv("KNUU_NAMESPACE")
	if namespaceEnv != "" {
		scope = namespaceEnv
		logrus.Warnf("KNUU_NAMESPACE is deprecated. Scope overridden to: %s", scope)
	}

	logrus.Infof("Initializing knuu with scope: %s", scope)

	// read timeout from env
	timeoutString := os.Getenv("KNUU_TIMEOUT")
	if timeoutString == "" {
		timeout = 60 * time.Minute
	} else {
		parsedTimeout, err := time.ParseDuration(timeoutString)
		if err != nil {
			return ErrCannotParseTimeout.Wrap(err)
		}
		timeout = parsedTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	k8sClient, err = k8s.New(ctx, scope)
	if err != nil {
		return ErrCannotInitializeK8s.Wrap(err)
	}

	if err := handleTimeout(); err != nil {
		return ErrCannotHandleTimeout.Wrap(err)
	}

	minioClient = &minio.Minio{
		Clientset: k8sClient.Clientset(),
		Namespace: k8sClient.Namespace(),
	}

	builderType := os.Getenv("KNUU_BUILDER")
	switch builderType {
	case "kubernetes":
		SetImageBuilder(&kaniko.Kaniko{
			K8sClientset: k8sClient.Clientset(),
			K8sNamespace: k8sClient.Namespace(),
			Minio:        minioClient, // same client is used to make the best use of the resources
		})
	case "docker", "":
		SetImageBuilder(&docker.Docker{
			K8sClientset: k8sClient.Clientset(),
			K8sNamespace: k8sClient.Namespace(),
		})
	default:
		return ErrInvalidKnuuBuilder.WithParams(builderType)
	}

	HandleStopSignal()
	return nil
}

// Deprecated: Identifier is deprecated, use Scope() instead.
func Identifier() string {
	logrus.Warn("Identifier() is deprecated, use Scope() instead.")
	return Scope()
}

// Deprecated: InitializeWithIdentifier is deprecated, use InitializeWithScope(scope string) instead.
func InitializeWithIdentifier(uniqueIdentifier string) error {
	logrus.Warn("InitializeWithIdentifier is deprecated, use InitializeWithScope(scope string) instead.")
	return InitializeWithScope(uniqueIdentifier)
}

// setupLogging Configures the log
func setupLogging() {
	// Set the default log level
	logrus.SetLevel(logrus.InfoLevel)

	// Set the custom formatter
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			directory := path.Base(path.Dir(f.File))
			return "", directory + "/" + filename + ":" + strconv.Itoa(f.Line)
		},
	})

	// Enable reporting the file and line
	logrus.SetReportCaller(true)

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

	logrus.Info("LOG_LEVEL: ", logrus.GetLevel())
}

func SetImageBuilder(b builder.Builder) {
	imageBuilder = b
}

func ImageBuilder() builder.Builder {
	return imageBuilder
}

// IsInitialized returns true if knuu is initialized, and false otherwise
func IsInitialized() bool {
	return k8sClient != nil
}

func CleanUp() error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return k8sClient.DeleteNamespace(ctx, testScope)
}

func HandleStopSignal() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-stop
		logrus.Info("Received signal to stop, cleaning up resources...")
		if err := CleanUp(); err != nil {
			logrus.Errorf("Error deleting namespace: %v", err)
		}
	}()
}

// handleTimeout creates a timeout handler that will delete all resources with the scope after the timeout
func handleTimeout() error {
	instance, err := NewInstance("timeout-handler")
	if err != nil {
		return ErrCannotCreateInstance.Wrap(err)
	}
	instance.instanceType = TimeoutHandlerInstance
	// FIXME: use supported kubernetes version images (use of latest could break) (https://github.com/celestiaorg/knuu/issues/116)
	if err := instance.SetImage("docker.io/bitnami/kubectl:latest"); err != nil {
		return ErrCannotSetImage.Wrap(err)
	}
	if err := instance.Commit(); err != nil {
		return ErrCannotCommitInstance.Wrap(err)
	}

	var commands []string

	// Wait for a specific period before executing the next operation.
	// This is useful to ensure that any previous operation has time to complete.
	commands = append(commands, fmt.Sprintf("sleep %d", int64(timeout.Seconds())))
	// Collects all resources (pods, services, etc.) within the specified namespace that match a specific label, excluding certain types,
	// and then deletes them. This is useful for cleaning up specific test resources before proceeding to delete the namespace.
	commands = append(commands, fmt.Sprintf("kubectl get all,pvc,netpol,roles,serviceaccounts,rolebindings,configmaps -l knuu.sh/scope=%s -n %s -o json | jq -r '.items[] | select(.metadata.labels.\"knuu.sh/type\" != \"%s\") | \"\\(.kind)/\\(.metadata.name)\"' | xargs -r kubectl delete -n %s", testScope, k8sClient.Namespace(), TimeoutHandlerInstance.String(), k8sClient.Namespace()))

	// Delete the namespace as it was created by knuu.
	logrus.Debugf("The namespace generated [%s] will be deleted", k8sClient.Namespace())
	commands = append(commands, fmt.Sprintf("kubectl delete namespace %s", k8sClient.Namespace()))

	// Delete all labeled resources within the namespace.
	// Unlike the previous command that excludes certain types, this command ensures that everything remaining is deleted.
	commands = append(commands, fmt.Sprintf("kubectl delete all,pvc,netpol,roles,serviceaccounts,rolebindings,configmaps -l knuu.sh/scope=%s -n %s", testScope, k8sClient.Namespace()))

	finalCmd := strings.Join(commands, " && ")

	// Run the command
	if err := instance.SetCommand("sh", "-c", finalCmd); err != nil {
		logrus.Debugf("The full command generated is [%s]", finalCmd)
		return ErrCannotSetCommand.Wrap(err)
	}

	rule := rbacv1.PolicyRule{
		Verbs:     []string{"*"},
		APIGroups: []string{"*"},
		Resources: []string{"*"},
	}

	if err := instance.AddPolicyRule(rule); err != nil {
		return ErrCannotAddPolicyRule.Wrap(err)
	}
	if err := instance.Start(); err != nil {
		return ErrCannotStartInstance.Wrap(err)
	}

	return nil
}
