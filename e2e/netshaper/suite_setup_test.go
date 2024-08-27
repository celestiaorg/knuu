package netshaper

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

type Suite struct {
	suite.Suite
	Knuu *knuu.Knuu

	cleanupMu     sync.Mutex
	totalTests    atomic.Int32
	finishedTests int32
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
	s.T().Logf("Cleaning up knuu...")
	if err := s.Knuu.CleanUp(context.Background()); err != nil {
		s.T().Logf("Error cleaning up test suite: %v", err)
	}
	s.T().Logf("Knuu is cleaned up")
}
