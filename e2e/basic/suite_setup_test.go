package basic

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/e2e"
	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	testTimeout = time.Minute * 15 // the same time that is used in the ci/cd pipeline

	nginxImage       = "docker.io/nginx:latest"
	nginxVolumeOwner = 0
	nginxPort        = 80
	nginxHTMLPath    = "/usr/share/nginx/html"
	nginxCommand     = "nginx -g daemon off"

	resourcesHTML = "../system/resources/html"

	alpineImage = "alpine:latest"
)

type Suite struct {
	suite.Suite
	Knuu     *knuu.Knuu
	Executor e2e.Executor

	cleanupMu     sync.Mutex
	totalTests    atomic.Int32
	finishedTests int32
}

var (
	nginxVolume = resource.MustParse("1Gi")
)

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) SetupSuite() {
	var (
		ctx = context.Background()
		// logger = logrus.New()
		err error
	)

	// k8sClient, err := k8s.NewClient(ctx, knuu.DefaultScope(), logger)
	// s.Require().NoError(err)

	// minioClient, err := minio.New(ctx, k8sClient, logger)
	// s.Require().NoError(err)

	s.Knuu, err = knuu.New(ctx, knuu.Options{
		ProxyEnabled: true,
		// K8sClient:    k8sClient,
		// MinioClient:  minioClient,
		// Timeout:      testTimeout,
	})
	s.Require().NoError(err)

	s.T().Logf("Scope: %s", s.Knuu.Scope)
	s.Knuu.HandleStopSignal(ctx)

	s.Executor.Kn = s.Knuu
}

// SetupTest is a test setup function that is called before each test is run.
func (s *Suite) SetupTest() {
	s.totalTests.Add(1)
	s.T().Parallel()
}

// TearDownTest is a test teardown function that is called after each test is run.
func (s *Suite) TearDownTest() {
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	s.finishedTests++

	// if I am the last test to finish, I need to clean up the suite
	if s.finishedTests == s.totalTests.Load() {
		s.cleanupSuite()
	}
}

func (s *Suite) cleanupSuite() {
	s.T().Logf("Cleaning up knuu...")
	if err := s.Knuu.CleanUp(context.Background()); err != nil {
		s.T().Logf("Error cleaning up test suite: %v", err)
	}
	s.T().Logf("Knuu is cleaned up")
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
