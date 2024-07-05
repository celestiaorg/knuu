package chaosmesh

import (
	"context"
	"time"

	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
)

func (s *Suite) TestDelay() {
	// s.T().Parallel()
	env := s.setupTestEnvironment()
	ctx := context.Background()

	s.T().Cleanup(func() {
		if err := env.cleanUp(ctx); err != nil {
			s.T().Log(err)
		}
	})

	elapsedBefore, err := env.measureBigFileDownloadTime(ctx)
	s.Require().NoError(err)

	s.Require().NoError(env.web.EnableChaosMesh())
	s.Require().NoError(env.web.SetDelay(ctx, 1*time.Second, 0))

	s.T().Logf(waitingLogMessage, waitTimeAfterChaosMesh)
	time.Sleep(waitTimeAfterChaosMesh)

	elapsedAfter, err := env.measureBigFileDownloadTime(ctx)
	s.Require().NoError(err)
	s.T().Logf("Time taken for wget before delay: %s \t after delay: %s", elapsedBefore, elapsedAfter)

	s.Assert().Greater(elapsedAfter, elapsedBefore, "Time after delay is not longer than time before delay")
}

func (s *Suite) TestLoss() {
	// s.T().Parallel()
	env := s.setupTestEnvironment()
	ctx := context.Background()

	s.T().Cleanup(func() {
		if err := env.cleanUp(ctx); err != nil {
			s.T().Log(err)
		}
	})

	elapsedBefore, err := env.measureBigFileDownloadTime(ctx)
	s.Require().NoError(err)

	s.Require().NoError(env.web.EnableChaosMesh())
	s.Require().NoError(env.web.SetLoss(ctx, 30, 0))

	s.T().Logf(waitingLogMessage, waitTimeAfterChaosMesh)
	time.Sleep(waitTimeAfterChaosMesh)

	elapsedAfter, err := env.measureBigFileDownloadTime(ctx)
	s.Require().NoError(err)
	s.T().Logf("Time taken for wget before loss: %s \t after loss: %s", elapsedBefore, elapsedAfter)

	s.Assert().Greater(elapsedAfter, elapsedBefore, "Time after loss is not longer than time before loss")
}

func (s *Suite) TestDuplicate() {
	// s.T().Parallel()
	env := s.setupTestEnvironment()
	ctx := context.Background()

	s.T().Cleanup(func() {
		if err := env.cleanUp(ctx); err != nil {
			s.T().Log(err)
		}
	})

	elapsedBefore, err := env.measureBigFileDownloadTime(ctx)
	s.Require().NoError(err)

	s.Require().NoError(env.web.EnableChaosMesh())
	s.Require().NoError(env.web.SetDuplicate(ctx, 50, 0))

	s.T().Logf(waitingLogMessage, waitTimeAfterChaosMesh)
	time.Sleep(waitTimeAfterChaosMesh)

	elapsedAfter, err := env.measureBigFileDownloadTime(ctx)
	s.Require().NoError(err)
	s.T().Logf("Time taken for wget before duplicate: %s \t after duplicate: %s", elapsedBefore, elapsedAfter)

	s.Assert().Greater(elapsedAfter, elapsedBefore, "Time after duplicate is not longer than time before duplicate")
}

func (s *Suite) TestCorrupt() {
	// s.T().Parallel()
	env := s.setupTestEnvironment()
	ctx := context.Background()

	s.T().Cleanup(func() {
		if err := env.cleanUp(ctx); err != nil {
			s.T().Log(err)
		}
	})

	elapsedBefore, err := env.measureBigFileDownloadTime(ctx)
	s.Require().NoError(err)

	s.Require().NoError(env.web.EnableChaosMesh())
	s.Require().NoError(env.web.SetCorrupt(ctx, 50, 0))

	s.T().Logf(waitingLogMessage, waitTimeAfterChaosMesh)
	time.Sleep(waitTimeAfterChaosMesh)

	elapsedAfter, err := env.measureBigFileDownloadTime(ctx)
	s.Require().NoError(err)
	s.T().Logf("Time taken for wget before corrupt: %s \t after corrupt: %s", elapsedBefore, elapsedAfter)

	s.Assert().Greater(elapsedAfter, elapsedBefore, "Time after corrupt is not longer than time before corrupt")
}

func (s *Suite) TestBandwidth() {
	// s.T().Parallel()
	env := s.setupTestEnvironment()
	ctx := context.Background()

	s.T().Cleanup(func() {
		if err := env.cleanUp(ctx); err != nil {
			s.T().Log(err)
		}
	})

	elapsedBefore, err := env.measureBigFileDownloadTime(ctx)
	s.Require().NoError(err)

	s.Require().NoError(env.web.EnableChaosMesh())
	bandwidthSpec := &v1alpha1.BandwidthSpec{
		Rate:   "1mbps",
		Limit:  20 * 1024 * 1024,
		Buffer: 10_000,
	}
	s.Require().NoError(env.web.SetBandwidth(ctx, bandwidthSpec, 0))

	s.T().Logf(waitingLogMessage, waitTimeAfterChaosMesh)
	time.Sleep(waitTimeAfterChaosMesh)

	elapsedAfter, err := env.measureBigFileDownloadTime(ctx)
	s.Require().NoError(err)
	s.T().Logf("Time taken for wget before bandwidth: %s \t after bandwidth: %s", elapsedBefore, elapsedAfter)

	s.Assert().Greater(elapsedAfter, elapsedBefore, "Time after bandwidth is not longer than time before bandwidth")
}
