package chaosmesh

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	nginxImage = "docker.io/nginx:latest"
	nginxPort  = 80

	//
	waitTimeAfterChaosMesh = 10 * time.Second
	waitingLogMessage      = "Waiting %s for chaos mesh to settle"
	bigFileSizeMB          = 100 // 100MB
)

type Suite struct {
	suite.Suite
	// Knuu *knuu.Knuu
}

type Env struct {
	knuu     *knuu.Knuu
	executor *instance.Executor
	web      *instance.Instance
	webIP    string
}

func (s *Suite) SetupSuite() {
	// var (
	// 	err error
	// 	ctx = context.Background()
	// )
	// s.Knuu, err = knuu.New(ctx, knuu.Options{EnableChaosMesh: true})
	// s.Require().NoError(err)
	// s.T().Logf("Scope: %s", s.Knuu.Scope())
	// s.Knuu.HandleStopSignal(ctx)
}

func (s *Suite) TearDownSuite() {
	// s.T().Cleanup(func() {
	// 	logrus.Info("Tearing down test suite...")
	// 	err := s.Knuu.CleanUp(context.Background())
	// 	if err != nil {
	// 		s.T().Logf("Error cleaning up test suite: %v", err)
	// 	}
	// })
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) setupTestEnvironment() *Env {
	ctx := context.Background()
	kn, err := knuu.New(ctx, knuu.Options{EnableChaosMesh: true})
	s.Require().NoError(err)
	s.T().Logf("Scope: %s", kn.Scope())
	kn.HandleStopSignal(ctx)

	executor, err := kn.NewExecutor(ctx)
	s.Require().NoError(err)

	web := s.createAndStartWebInstance(ctx, kn)
	webIP, err := web.GetIP(ctx)
	s.Require().NoError(err)

	return &Env{
		knuu:     kn,
		executor: executor,
		web:      web,
		webIP:    webIP,
	}
}

func (s *Suite) createAndStartWebInstance(ctx context.Context, kn *knuu.Knuu) *instance.Instance {
	web, err := kn.NewInstance("web")
	s.Require().NoError(err)
	err = web.SetImage(ctx, nginxImage)
	s.Require().NoError(err)
	s.Require().NoError(web.AddPortTCP(nginxPort))
	s.Require().NoError(web.Commit())
	s.Require().NoError(web.Start(ctx))
	// Create a big file to download (3000MB)
	_, _ = web.ExecuteCommand(ctx, "dd", "if=/dev/zero", "of=/usr/share/nginx/html/bigfile", "bs=1M", fmt.Sprintf("count=%d", bigFileSizeMB))
	return web
}

func (e *Env) measureBigFileDownloadTime(ctx context.Context) (time.Duration, error) {
	startTime := time.Now()
	_, err := e.executor.ExecuteCommand(ctx, "wget", "-q", "-O", "-", e.webIP+"/bigfile")
	if err != nil {
		logrus.Errorf("Error downloading big file: %v", err)
	}
	return time.Since(startTime), nil
}

func (e *Env) cleanUp(ctx context.Context) error {
	// return instance.BatchDestroy(ctx, e.web, e.executor.Instance)
	return e.knuu.CleanUp(ctx)
}
