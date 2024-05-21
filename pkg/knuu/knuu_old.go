// Package knuu provides the core functionality of knuu.
package knuu

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/builder/docker"
	"github.com/celestiaorg/knuu/pkg/builder/kaniko"
)

// This is a temporary variable to hold the knuu instance until we refactor knuu pkg
// TODO: remove this temporary variable
var tmpKnuu *Knuu

// Initialize initializes knuu with a unique scope
func Initialize() error {
	t := time.Now()
	scope := fmt.Sprintf("%s-%03d", t.Format("20060102-150405"), t.Nanosecond()/1e6)
	return InitializeWithScope(scope)
}

func Scope() string {
	if tmpKnuu == nil {
		return ""
	}
	return tmpKnuu.Scope()
}

// InitializeWithScope initializes knuu with a given scope
func InitializeWithScope(testScope string) error {
	// Override scope if KNUU_NAMESPACE is set
	namespaceEnv := os.Getenv("KNUU_NAMESPACE")
	if namespaceEnv != "" {
		testScope = namespaceEnv
		logrus.Warnf("KNUU_NAMESPACE is deprecated. Scope overridden to: %s", testScope)
	}

	logrus.Infof("Initializing knuu with scope: %s", testScope)

	// read timeout from env
	var (
		timeoutString = os.Getenv("KNUU_TIMEOUT")
		timeout       = 60 * time.Minute
	)
	if timeoutString != "" {
		parsedTimeout, err := time.ParseDuration(timeoutString)
		if err != nil {
			return ErrCannotParseTimeout.Wrap(err)
		}
		timeout = parsedTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var err error
	tmpKnuu, err = New(ctx,
		WithTestScope(testScope),
		WithTimeout(timeout),
	)
	if err != nil {
		return ErrCannotInitializeKnuu.Wrap(err)
	}

	builderType := os.Getenv("KNUU_BUILDER")
	switch builderType {
	case "kubernetes":
		tmpKnuu.ImageBuilder = &kaniko.Kaniko{
			K8sClientset: tmpKnuu.K8sCli.Clientset(),
			K8sNamespace: tmpKnuu.K8sCli.Namespace(),
			Minio:        tmpKnuu.MinioCli,
		}
	case "docker", "":
		tmpKnuu.ImageBuilder = &docker.Docker{
			K8sClientset: tmpKnuu.K8sCli.Clientset(),
			K8sNamespace: tmpKnuu.K8sCli.Namespace(),
		}
	default:
		return ErrInvalidKnuuBuilder.WithParams(builderType)
	}

	// TODO: this must be moved to somewhere more meaningful
	tmpKnuu.HandleStopSignal()
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

func ImageBuilder() builder.Builder {
	if tmpKnuu == nil {
		return nil
	}
	return tmpKnuu.ImageBuilder
}

// IsInitialized returns true if knuu is initialized, and false otherwise
func IsInitialized() bool {
	return tmpKnuu != nil
}

func CleanUp() error {
	if tmpKnuu == nil {
		return errors.New("tmpKnuu is not initialized")
	}
	return tmpKnuu.CleanUp(context.Background())
}
