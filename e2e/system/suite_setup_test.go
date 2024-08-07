package system

import (
	"context"
	"fmt"
	"net"
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

	maxRetries = 10
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
	s.T().Cleanup(func() {
		logrus.Info("Tearing down test suite...")
		err := s.Knuu.CleanUp(context.Background())
		if err != nil {
			s.T().Logf("Error cleaning up test suite: %v", err)
		}
	})
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) createNginxInstance(ctx context.Context, name string) *instance.Instance {
	ins, err := s.Knuu.NewInstance(name)
	s.Require().NoError(err)

	s.Require().NoError(ins.SetImage(ctx, nginxImage))
	s.Require().NoError(ins.AddPortTCP(nginxPort))

	return ins
}

func (s *Suite) createNginxInstanceWithVolume(ctx context.Context, name string) *instance.Instance {
	ins := s.createNginxInstance(ctx, name)

	_, err := ins.ExecuteCommand(ctx, "mkdir", "-p", nginxHTMLPath)
	s.Require().NoError(err)

	s.Require().NoError(ins.AddVolumeWithOwner(nginxHTMLPath, nginxVolume, nginxVolumeOwner))
	return ins
}

func (s *Suite) waitForNginxReady(ctx context.Context, instance *instance.Instance) error {
	return retryOperation(func() error {
		ip, err := instance.GetIP(ctx)
		if err != nil {
			return err
		}

		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, fmt.Sprintf("%d", nginxPort)), 5*time.Second)
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}, maxRetries)
}
