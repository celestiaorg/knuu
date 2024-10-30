package sidecars

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/celestiaorg/knuu/e2e"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	testTimeout = time.Minute * 5 // the same time that is used in the ci/cd pipeline
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
		err    error
	)

	s.Knuu, err = knuu.New(ctx, knuu.Options{
		Timeout: testTimeout,
		Logger:  logger,
	})
	s.Require().NoError(err)

	s.T().Logf("Scope: %s", s.Knuu.Scope)
	s.Knuu.HandleStopSignal(ctx)

	s.Executor.Kn = s.Knuu
}
