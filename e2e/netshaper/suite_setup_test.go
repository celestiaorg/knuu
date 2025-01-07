package netshaper

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/celestiaorg/knuu/e2e"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/knuu"
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

	k8sClient, err := k8s.NewClient(ctx, knuu.DefaultScope(), logger, s.K8sDefaultOptions()...)
	s.Require().NoError(err)

	s.Knuu, err = knuu.New(ctx, knuu.Options{
		ProxyEnabled: true,
		K8sClient:    k8sClient,
		Logger:       logger,
	})
	s.Require().NoError(err)
	s.T().Logf("Scope: %s", s.Knuu.Scope)
	s.Knuu.HandleStopSignal(ctx)
}
