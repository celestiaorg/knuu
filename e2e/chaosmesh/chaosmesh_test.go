package chaosmesh

import (
	"context"
	"time"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
)

const (
	nginxImage = "docker.io/nginx:latest"
	nginxPort  = 80
)

func (s *Suite) TestDelay() {
	s.T().Parallel()
	executor, web, webIP := s.setupTestEnvironment()
	ctx := context.Background()

	s.T().Cleanup(func() {
		err := instance.BatchDestroy(ctx, executor.Instance, web)
		if err != nil {
			s.T().Log(err)
		}
	})

	elapsedTimeBeforeChaos := s.measureBigFileDownloadTime(executor, webIP)

	s.Require().NoError(web.EnableChaosMesh())
	s.Require().NoError(web.SetDelay(ctx, 10*time.Second, 0))

	elapsedTimeAfterChaos := s.measureBigFileDownloadTime(executor, webIP)
	s.T().Logf("Time taken for wget before delay: %s \t after delay: %s", elapsedTimeBeforeChaos, elapsedTimeAfterChaos)

	s.Assert().Greater(elapsedTimeAfterChaos, elapsedTimeBeforeChaos, "Time after delay is not longer than time before delay")
}

func (s *Suite) TestLoss() {
	s.T().Parallel()
	executor, web, webIP := s.setupTestEnvironment()
	ctx := context.Background()

	s.T().Cleanup(func() {
		err := instance.BatchDestroy(ctx, executor.Instance, web)
		if err != nil {
			s.T().Log(err)
		}
	})

	elapsedTimeBeforeChaos := s.measureBigFileDownloadTime(executor, webIP)

	s.Require().NoError(web.EnableChaosMesh())
	s.Require().NoError(web.SetLoss(ctx, 50, 0))

	elapsedTimeAfterChaos := s.measureBigFileDownloadTime(executor, webIP)
	s.T().Logf("Time taken for wget before loss: %s \t after loss: %s", elapsedTimeBeforeChaos, elapsedTimeAfterChaos)

	s.Assert().Greater(elapsedTimeAfterChaos, elapsedTimeBeforeChaos, "Time after loss is not longer than time before loss")
}

func (s *Suite) TestDuplicate() {
	s.T().Parallel()
	executor, web, webIP := s.setupTestEnvironment()
	ctx := context.Background()

	s.T().Cleanup(func() {
		err := instance.BatchDestroy(ctx, executor.Instance, web)
		if err != nil {
			s.T().Log(err)
		}
	})

	elapsedTimeBeforeChaos := s.measureBigFileDownloadTime(executor, webIP)

	s.Require().NoError(web.EnableChaosMesh())
	s.Require().NoError(web.SetDuplicate(ctx, 50, 0))

	elapsedTimeAfterChaos := s.measureBigFileDownloadTime(executor, webIP)
	s.T().Logf("Time taken for wget before duplicate: %s \t after duplicate: %s", elapsedTimeBeforeChaos, elapsedTimeAfterChaos)

	s.Assert().Greater(elapsedTimeAfterChaos, elapsedTimeBeforeChaos, "Time after duplicate is not longer than time before duplicate")
}

func (s *Suite) TestCorrupt() {
	s.T().Parallel()
	executor, web, webIP := s.setupTestEnvironment()
	ctx := context.Background()

	s.T().Cleanup(func() {
		err := instance.BatchDestroy(ctx, executor.Instance, web)
		if err != nil {
			s.T().Log(err)
		}
	})

	elapsedTimeBeforeChaos := s.measureBigFileDownloadTime(executor, webIP)

	s.Require().NoError(web.EnableChaosMesh())
	s.Require().NoError(web.SetCorrupt(ctx, 50, 0))

	elapsedTimeAfterChaos := s.measureBigFileDownloadTime(executor, webIP)
	s.T().Logf("Time taken for wget before corrupt: %s \t after corrupt: %s", elapsedTimeBeforeChaos, elapsedTimeAfterChaos)

	s.Assert().Greater(elapsedTimeAfterChaos, elapsedTimeBeforeChaos, "Time after corrupt is not longer than time before corrupt")
}

func (s *Suite) TestBandwidth() {
	s.T().Parallel()
	executor, web, webIP := s.setupTestEnvironment()
	ctx := context.Background()

	s.T().Cleanup(func() {
		err := instance.BatchDestroy(ctx, executor.Instance, web)
		if err != nil {
			s.T().Log(err)
		}
	})

	elapsedTimeBeforeChaos := s.measureBigFileDownloadTime(executor, webIP)

	s.Require().NoError(web.EnableChaosMesh())
	bandwidthSpec := &v1alpha1.BandwidthSpec{
		Rate:   "1mbps",
		Limit:  20971520,
		Buffer: 10000,
	}
	s.Require().NoError(web.SetBandwidth(ctx, bandwidthSpec, 0))

	elapsedTimeAfterChaos := s.measureBigFileDownloadTime(executor, webIP)
	s.T().Logf("Time taken for wget before bandwidth: %s \t after bandwidth: %s", elapsedTimeBeforeChaos, elapsedTimeAfterChaos)

	s.Assert().Greater(elapsedTimeAfterChaos, elapsedTimeBeforeChaos, "Time after bandwidth is not longer than time before bandwidth")
}

func (s *Suite) setupTestEnvironment() (*instance.Executor, *instance.Instance, string) {
	executor, err := s.Knuu.NewExecutor(context.Background())
	s.Require().NoError(err)

	web := s.createAndStartWebInstance()
	webIP, err := web.GetIP(context.Background())
	s.Require().NoError(err)

	return executor, web, webIP
}

func (s *Suite) measureBigFileDownloadTime(executor *instance.Executor, webIp string) time.Duration {
	startTime := time.Now()
	_, err := executor.ExecuteCommand(context.Background(), "wget", "-q", "-O", "-", webIp+"/bigfile")
	s.Require().NoError(err)
	return time.Since(startTime)
}

func (s *Suite) createAndStartWebInstance() *instance.Instance {
	web, err := s.Knuu.NewInstance("web")
	s.Require().NoError(err)
	err = web.SetImage(context.Background(), nginxImage)
	s.Require().NoError(err)
	s.Require().NoError(web.AddPortTCP(nginxPort))
	s.Require().NoError(web.Commit())
	s.Require().NoError(web.Start(context.Background()))
	// Create a big file to download (50MB)
	_, _ = web.ExecuteCommand(context.Background(), "dd", "if=/dev/zero", "of=/usr/share/nginx/html/bigfile", "bs=1M", "count=50")
	return web
}
