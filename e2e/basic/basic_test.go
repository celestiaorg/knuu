package basic

import (
	"context"
	"strings"
)

func (s *Suite) TestBasic() {
	const namePrefix = "basic"
	ctx := context.Background()

	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)

	s.Require().NoError(target.Build().SetImage(ctx, alpineImage))
	s.Require().NoError(target.Build().SetStartCommand("sleep", "infinity"))
	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Execution().Start(ctx))

	// Perform the test
	expectedOutput := "Hello World"
	output, err := target.Execution().ExecuteCommand(ctx, "echo", expectedOutput)
	s.Require().NoError(err)

	output = strings.TrimSpace(output)
	s.Assert().Equal(expectedOutput, output)
}
