package netshaper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/celestiaorg/knuu/e2e"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

type Suite struct {
	e2e.Suite
}

func TestRunSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) SetupSuite() {
	ctx := context.Background()

	var err error
	s.Knuu, err = knuu.New(ctx, knuu.Options{
		ProxyEnabled: true,
	})
	s.Require().NoError(err)
	s.T().Logf("Scope: %s", s.Knuu.Scope)
	s.Knuu.HandleStopSignal(ctx)
}
