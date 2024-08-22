package basic

import (
	"context"
	"strings"

	"github.com/stretchr/testify/assert"
)

func (s *Suite) TestBasic() {
	const namePrefix = "basic"
	ctx := context.Background()

	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)

	s.Require().NoError(target.Build().SetImage(ctx, alpineImage))
	s.Require().NoError(target.Build().SetStartCommand("sleep", "infinity"))
	s.Require().NoError(target.Build().Commit(ctx))

	// Test Logic
	s.Require().NoError(target.Execution().Start(ctx))

	// Perform the test
	tt := []struct {
		name string
	}{
		{name: "Hello World"},
	}

	for _, tc := range tt {
		tc := tc
		s.Run(tc.name, func() {
			output, err := target.Execution().ExecuteCommand(ctx, "echo", tc.name)
			s.Require().NoError(err)

			output = strings.TrimSpace(output)
			assert.Contains(s.T(), output, tc.name)
		})
	}
}
