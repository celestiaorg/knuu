package e2e

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	NginxHTMLPath = "/usr/share/nginx/html"
	NginxPort     = 80
	NginxCommand  = "nginx -g daemon off"

	nginxImage       = "docker.io/nginx:latest"
	nginxVolumeOwner = 0
)

type Suite struct {
	suite.Suite
	Knuu     *knuu.Knuu
	Executor Executor

	cleanupMu     sync.Mutex
	totalTests    atomic.Int32
	finishedTests int32
}

var (
	nginxVolume = resource.MustParse("1Gi")
)

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
	return
	s.T().Logf("Cleaning up knuu...")
	if err := s.Knuu.CleanUp(context.Background()); err != nil {
		s.T().Logf("Error cleaning up test suite: %v", err)
	}
	s.T().Logf("Knuu is cleaned up")
}

func (s *Suite) CreateNginxInstance(ctx context.Context, name string) *instance.Instance {
	ins, err := s.Knuu.NewInstance(name)
	s.Require().NoError(err)

	s.Require().NoError(ins.Build().SetImage(ctx, nginxImage))
	s.Require().NoError(ins.Network().AddPortTCP(NginxPort))
	return ins
}

func (s *Suite) CreateNginxInstanceWithVolume(ctx context.Context, name string) *instance.Instance {
	ins := s.CreateNginxInstance(ctx, name)

	err := ins.Build().ExecuteCommand("mkdir", "-p", NginxHTMLPath)
	s.Require().NoError(err)

	s.Require().NoError(ins.Storage().AddVolumeWithOwner(NginxHTMLPath, nginxVolume, nginxVolumeOwner))
	return ins
}

func (s *Suite) RetryOperation(operation func() error, maxRetries int) error {
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
