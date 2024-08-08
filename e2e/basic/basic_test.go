package basic

import (
	"context"
	"strings"

	"github.com/stretchr/testify/assert"
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

	ts.Require().NoError(target.Build().SetImage(ctx, testImage))
	ts.Require().NoError(target.Build().SetStartCommand("sleep", "infinity"))
	ts.Require().NoError(target.Build().Commit(ctx))

	ts.T().Cleanup(func() {
		if err := target.Execution().Destroy(ctx); err != nil {
			ts.T().Logf("error destroying instance: %v", err)
		}
	})

	// Test Logic
	ts.Require().NoError(target.Execution().Start(ctx))

	// Perform the test
	tt := []struct {
		name string
	}{
		{name: "Hello World"},
	}

	for _, tc := range tt {
		tc := tc
		ts.Run(tc.name, func() {
			output, err := target.Execution().ExecuteCommand(ctx, "echo", tc.name)
			ts.Require().NoError(err)

			output = strings.TrimSpace(output)
			assert.Contains(ts.T(), output, tc.name)
		})
	}
}
