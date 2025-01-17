package machine

import (
	"context"
)

func (s *Suite) TestLightNode() {
	ctx := context.Background()

	target, err := s.Knuu.NewInstance("light-node")
	s.Require().NoError(err)

	s.Require().NoError(target.Build().SetImage(ctx, "ghcr.io/celestiaorg/celestia-node:v0.20.4"))
	s.Require().NoError(target.Build().SetArgs("celestia", "light", "start", "--p2p.network", "celestia"))
	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Build().SetEnvironmentVariable("NODE_TYPE", "light"))
	s.Require().NoError(target.Build().SetEnvironmentVariable("P2P_NETWORK", "celestia"))
	s.Require().NoError(target.Execution().Start(ctx))

	// Perform the test
	// expectedOutput := "Hello World"
	// output, err := target.Execution().ExecuteCommand(ctx, "echo", expectedOutput)
	// s.Require().NoError(err)

	// output = strings.TrimSpace(output)
	// s.Assert().Equal(expectedOutput, output)
}

// podman run -e NODE_TYPE=light -e P2P_NETWORK=celestia ghcr.io/celestiaorg/celestia-node:v0.20.4 celestia light start --p2p.network celestia
