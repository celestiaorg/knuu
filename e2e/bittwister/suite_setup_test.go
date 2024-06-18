package bittwister

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

type Suite struct {
	suite.Suite
	Knuu *knuu.Knuu
}

func (s *Suite) SetupSuite() {
	ctx := context.Background()
	k8sClient, err := k8s.NewClient(ctx, knuu.DefaultTestScope())
	s.Require().NoError(err)

	s.Knuu, err = knuu.New(ctx, k8sClient, knuu.Options{ProxyEnabled: true})
	s.Require().NoError(err)
	s.T().Logf("Scope: %s", s.Knuu.Scope())
	s.Knuu.HandleStopSignal(ctx)
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
