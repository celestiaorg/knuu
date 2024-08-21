package system

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
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

	wg            sync.WaitGroup
	knuuCleanupMu sync.Mutex
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

	// Since the SetupTest is called when the test is going to actually running i.e. `CONT`,
	// which is called sometimes after some other tests are already finished,
	// it calls the cleanup function prematurely; therefore, We need to count the number of
	// all tests in advance and add them to the wait group.
	// This way we can be sure that the teardown will be executed only after all tests are finished.
	s.wg.Add(s.countTests())
}

// SetupTest is a test setup function that is called before each test is run.
func (s *Suite) SetupTest() {
	s.T().Parallel()
}

// TearDownTest is a test teardown function that is called after each test is run.
func (s *Suite) TearDownTest() {
	s.wg.Done()
}

func (s *Suite) TearDownSuite() {
	// We have to use a goroutine because for some strange reasons the tests
	// are waiting for this function to finish and therefore we are in a deadlock
	go func() {
		// we need to handle it because of the parallelism, the TearDownSuite() is called prematurely
		s.wg.Wait()

		s.T().Logf("Cleaning up knuu...")
		if err := s.Knuu.CleanUp(context.Background()); err != nil {
			s.T().Logf("Error cleaning up test suite: %v", err)
		}
		s.T().Logf("Knuu is cleaned up")
	}()
}

func TestRunSuite(t *testing.T) {
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

// A little bit of a hack to count the number of tests in the suite.
// We need to know the number of tests in advance to be able to call the teardown function only after all tests are finished.
func (s *Suite) countTests() int {
	var (
		methodFinder = reflect.TypeOf(s)
		numOfTests   = 0
	)
	for i := 0; i < methodFinder.NumMethod(); i++ {
		method := methodFinder.Method(i)
		if strings.HasPrefix(method.Name, "Test") {
			numOfTests++
		}
	}
	return numOfTests
}
