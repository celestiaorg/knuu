/*
* This file is deprecated.
* Please use the new package knuu instead.
* This file keeps the old functionality of knuu for backward compatibility.
* A global variable is defined, tmpKnuu, which is used to hold the knuu instance.
 */
package knuu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/builder/docker"
	"github.com/celestiaorg/knuu/pkg/builder/kaniko"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/minio"
)

const minioBucketName = "knuu"

// This is a temporary variable to hold the knuu instance until we refactor knuu pkg
// TODO: remove this temporary variable
var tmpKnuu *Knuu

// Deprecated: Use the new package knuu instead.
// Initialize initializes knuu with a unique scope
func Initialize() error {
	t := time.Now()
	scope := fmt.Sprintf("%s-%03d", t.Format("20060102-150405"), t.Nanosecond()/1e6)
	return InitializeWithScope(scope)
}

// Deprecated: Use the new package knuu instead.
func Scope() string {
	if tmpKnuu == nil {
		return ""
	}
	return tmpKnuu.Scope()
}

// Deprecated: Use the new package knuu instead.
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

	k8sClient, err := k8s.NewClient(ctx, testScope)
	if err != nil {
		return ErrCannotInitializeKnuu.Wrap(err)
	}

	minioClient, err := minio.New(ctx, k8sClient)
	if err != nil {
		return ErrCannotInitializeKnuu.Wrap(err)
	}

	tmpKnuu, err = New(ctx, k8sClient, Options{
		TestScope:    testScope,
		Timeout:      timeout,
		ProxyEnabled: true,
		MinioClient:  minioClient,
	})
	if err != nil {
		return ErrCannotInitializeKnuu.Wrap(err)
	}

	builderType := os.Getenv("KNUU_BUILDER")
	switch builderType {
	case "kubernetes":
		tmpKnuu.ImageBuilder = &kaniko.Kaniko{
			SystemDependencies: tmpKnuu.SystemDependencies,
		}
	case "docker", "":
		tmpKnuu.ImageBuilder = &docker.Docker{
			K8sClientset: tmpKnuu.K8sClient.Clientset(),
			K8sNamespace: tmpKnuu.K8sClient.Namespace(),
		}
	default:
		return ErrInvalidKnuuBuilder.WithParams(builderType)
	}

	// TODO: this must be moved to somewhere more meaningful
	tmpKnuu.HandleStopSignal(context.Background())
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

// Deprecated: Use the new package knuu instead.
func ImageBuilder() builder.Builder {
	if tmpKnuu == nil {
		return nil
	}
	return tmpKnuu.ImageBuilder
}

// Deprecated: Use the new package knuu instead.
// IsInitialized returns true if knuu is initialized, and false otherwise
func IsInitialized() bool {
	return tmpKnuu != nil
}

// Deprecated: Use the new package knuu instead.
func CleanUp() error {
	if tmpKnuu == nil {
		return errors.New("tmpKnuu is not initialized")
	}
	return tmpKnuu.CleanUp(context.Background())
}

// Deprecated: Use the new package knuu instead.
func PushFileToMinio(ctx context.Context, contentName string, reader io.Reader) error {
	return tmpKnuu.MinioClient.Push(ctx, reader, contentName, minioBucketName)
}

// Deprecated: Use the new package knuu instead.
func GetMinioURL(ctx context.Context, contentName string) (string, error) {
	return tmpKnuu.MinioClient.GetURL(ctx, contentName, minioBucketName)
}
