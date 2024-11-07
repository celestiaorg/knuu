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

	ctx := context.Background()

	sidecar := &testSidecar{
		StartCommand: []string{"sh", "-c", "sleep infinity"},
	}
	s.startNewInstanceWithSidecar(ctx, namePrefix, sidecar)

	// Create a file in the sidecar instance
	out, err := sidecar.Instance().Execution().ExecuteCommand(ctx, command)
	s.Require().NoError(err, "executing command output: %v", out)

	outTrimmed := strings.TrimSpace(out)
	s.Assert().Equal(cmdMsg, outTrimmed)
}
