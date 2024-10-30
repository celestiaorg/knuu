package sidecars

import (
	"context"
	"strings"
)

func (s *Suite) TestExecuteCommandInSidecar() {
	const (
		namePrefix = "execute-command-in-sidecar"
		cmdMsg     = "Hello World!"
		command    = "echo " + cmdMsg
	)

	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)

	ctx := context.Background()
	s.Require().NoError(target.Build().SetImage(ctx, alpineImage))
	s.Require().NoError(target.Build().SetArgs("tail", "-f", "/dev/null")) // Keep the container running
	s.Require().NoError(target.Build().Commit(ctx))

	sidecar := &testSidecar{
		StartCommand: []string{"sh", "-c", "sleep infinity"},
	}

	s.Require().NoError(target.Sidecars().Add(ctx, sidecar))
	s.Require().NoError(target.Execution().Start(ctx))

	// Create a file in the sidecar instance
	out, err := sidecar.Instance().Execution().ExecuteCommand(ctx, command)
	s.Require().NoError(err, "executing command output: %v", out)

	outTrimmed := strings.TrimSpace(out)
	s.Assert().Equal(cmdMsg, outTrimmed)
}
