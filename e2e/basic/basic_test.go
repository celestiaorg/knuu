package basic

import (
	"context"

	"github.com/stretchr/testify/assert"

	"github.com/celestiaorg/knuu/pkg/instance"
)

const (
	TestImage = "alpine:latest"
)

func (s *TestSuite) TestBasic() {
	s.T().Parallel()
	// Setup

	ctx := context.Background()

	target, err := s.Knuu.NewInstance("alpine")
	s.Require().NoError(err)

	s.Require().NoError(target.SetImage(ctx, TestImage))
	s.Require().NoError(target.SetCommand("sleep", "infinity"))
	s.Require().NoError(target.Commit())

	s.T().Cleanup(func() {
		s.T().Log("Tearing down Basic Test...")
		err := instance.BatchDestroy(ctx, target)
		if err != nil {
			s.T().Logf("error destroying instances: %v", err)
		}
	})

	// Test Logic
	s.Require().NoError(target.Start(ctx))
	s.Require().NoError(target.WaitInstanceIsRunning(ctx))

	// Perform the test
	type testCase struct {
		name string
	}

	tt := []testCase{
		{"Hello World"},
	}

	for _, tc := range tt {
		tc := tc
		s.Run(tc.name, func() {
			s.T().Logf("Running test case: %s", tc.name)
			output, err := target.ExecuteCommand(ctx, "echo", tc.name)
			s.Require().NoError(err)

			assert.Contains(s.T(), output, tc.name)
		})
	}

}
