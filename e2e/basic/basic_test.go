package basic

import (
	"context"

	"github.com/stretchr/testify/assert"

	"github.com/celestiaorg/knuu/pkg/instance"
)

const (
	testImage = "alpine:latest"
)

func (ts *TestSuite) TestBasic() {
	ts.T().Parallel()
	// Setup

	ctx := context.Background()

	target, err := ts.Knuu.NewInstance("alpine")
	ts.Require().NoError(err)

	ts.Require().NoError(target.SetImage(ctx, testImage))
	ts.Require().NoError(target.SetCommand("sleep", "infinity"))
	ts.Require().NoError(target.Commit())

	ts.T().Cleanup(func() {
		s.T().Log("Tearing down Basic Test...")
		err := instance.BatchDestroy(ctx, target)
		if err != nil {
			s.T().Logf("error destroying instances: %v", err)
		}
	})

	// Test Logic
	ts.Require().NoError(target.Start(ctx))
	ts.Require().NoError(target.WaitInstanceIsRunning(ctx))

	// Perform the test
	type testCase struct {
		name string
	}

	tt := []testCase{
		{"Hello World"},
	}

	for _, tc := range tt {
		tc := tc
		ts.Run(tc.name, func() {
			output, err := target.ExecuteCommand(ctx, "echo", tc.name)
			ts.Require().NoError(err)

			assert.Contains(ts.T(), output, tc.name)
		})
	}

}
