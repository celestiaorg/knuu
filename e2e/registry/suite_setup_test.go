package system

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/celestiaorg/knuu/e2e"
	"github.com/celestiaorg/knuu/pkg/builder/kaniko"
	"github.com/celestiaorg/knuu/pkg/builder/registry"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/celestiaorg/knuu/pkg/minio"
	"github.com/celestiaorg/knuu/pkg/system"
)

const (
	testTimeout = time.Minute * 100 // the same time that is used in the ci/cd pipeline
	alpineImage = "alpine:latest"
)

type Suite struct {
	e2e.Suite
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) SetupSuite() {
	var (
		ctx    = context.Background()
		logger = logrus.New()
	)

	k8sClient, err := k8s.NewClient(ctx, knuu.DefaultScope(), logger)
	s.Require().NoError(err, "Error creating k8s client")

	minioClient, err := minio.New(ctx, k8sClient, logger)
	s.Require().NoError(err, "Error creating minio client")

	// registry, err := registry.NewScaleway("fr-par", "test-moji", "username", "password")
	// s.Require().NoError(err)

	registry := registry.Registry(nil)

	imageBuilder, err := kaniko.New(
		&system.SystemDependencies{
			K8sClient:   k8sClient,
			MinioClient: minioClient,
			Logger:      logger,
		},
		registry,
	)
	s.Require().NoError(err)

	s.Knuu, err = knuu.New(ctx, knuu.Options{
		K8sClient:    k8sClient,
		MinioClient:  minioClient, // needed for build from git tests
		Timeout:      testTimeout,
		ImageBuilder: imageBuilder,
		Logger:       logger,
	})
	s.Require().NoError(err)

	s.T().Logf("Scope: %s", s.Knuu.Scope)
	s.Knuu.HandleStopSignal(ctx)

	s.Executor.Kn = s.Knuu
}
