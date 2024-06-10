package basic

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

type TestSuite struct {
	suite.Suite
	Knuu *knuu.Knuu
}

func (s *TestSuite) SetupSuite() {
	var (
		err error
		ctx = context.Background()
	)
	s.Knuu, err = knuu.New(ctx)
	s.Require().NoError(err)
	s.T().Logf("Error cleaning up test suite: %v", err)
	s.Knuu.HandleStopSignal(ctx)
}

func (s *TestSuite) TearDownSuite() {
	s.T().Cleanup(func() {
		logrus.Info("Tearing down test suite...")
		s.Require().NoError(s.Knuu.CleanUp(context.Background()))
	})
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
