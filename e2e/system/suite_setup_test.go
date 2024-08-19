package system

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/e2e"
	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/celestiaorg/knuu/pkg/minio"
)

const (
	nginxImage       = "docker.io/nginx:latest"
	nginxVolumeOwner = 0
	nginxPort        = 80
	nginxHTMLPath    = "/usr/share/nginx/html"
	nginxCommand     = "nginx -g daemon off"

	resourcesHTML           = "resources/html"
	resourcesFileCMToFolder = "resources/file_cm_to_folder"
)

type Suite struct {
	suite.Suite
	Knuu     *knuu.Knuu
	Executor e2e.Executor
}

var (
	nginxVolume = resource.MustParse("1Gi")
)

func (s *Suite) SetupSuite() {
	var (
		ctx    = context.Background()
		logger = logrus.New()
	)

	k8sClient, err := k8s.NewClient(ctx, knuu.DefaultScope(), logger)
	s.Require().NoError(err, "Error creating k8s client")

	minioClient, err := minio.New(ctx, k8sClient, logger)
	s.Require().NoError(err, "Error creating minio client")

	s.Knuu, err = knuu.New(ctx, knuu.Options{
		ProxyEnabled: true,
		K8sClient:    k8sClient,
		MinioClient:  minioClient, // needed for build from git tests
	})
	s.Require().NoError(err)

	s.T().Logf("Scope: %s", s.Knuu.Scope)
	s.Knuu.HandleStopSignal(ctx)

	s.Executor.Kn = s.Knuu
}

func (s *Suite) TearDownSuite() {
	logrus.Info("Tearing down test suite...")
	if err := s.Knuu.CleanUp(context.Background()); err != nil {
		s.T().Logf("Error cleaning up test suite: %v", err)
	}
}

func TestRunSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(Suite))
}

func (s *Suite) createNginxInstance(ctx context.Context, name string) *instance.Instance {
	ins, err := s.Knuu.NewInstance(name)
	s.Require().NoError(err)

	s.Require().NoError(ins.Build().SetImage(ctx, nginxImage))
	s.Require().NoError(ins.Network().AddPortTCP(nginxPort))
	return ins
}

func (s *Suite) createNginxInstanceWithVolume(ctx context.Context, name string) *instance.Instance {
	ins := s.createNginxInstance(ctx, name)

	err := ins.Build().ExecuteCommand("mkdir", "-p", nginxHTMLPath)
	s.Require().NoError(err)

	s.Require().NoError(ins.Storage().AddVolumeWithOwner(nginxHTMLPath, nginxVolume, nginxVolumeOwner))
	return ins
}

func (s *Suite) retryOperation(operation func() error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		s.T().Logf("Retrying operation (%d/%d)...", i+1, maxRetries)
		if err = operation(); err == nil {
			return nil
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
}
