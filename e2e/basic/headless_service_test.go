package basic

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

const gopingImage = "ghcr.io/celestiaorg/goping:4803195"

func (s *Suite) TestHeadlessService() {
	const (
		namePrefix       = "headless-srv-test"
		numOfPingPackets = 100
		numOfTests       = 10
		packetTimeout    = 1 * time.Second
		gopingPort       = 8001
	)
	ctx := context.Background()

	mother, err := s.Knuu.NewInstance(namePrefix + "mother")
	s.Require().NoError(err)

	err = mother.Build().SetImage(ctx, gopingImage)
	s.Require().NoError(err)

	s.Require().NoError(mother.Network().AddPortTCP(gopingPort))
	s.Require().NoError(mother.Build().Commit(ctx))

	err = mother.Build().SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", gopingPort))
	s.Require().NoError(err)

	target, err := mother.CloneWithName(namePrefix + "target")
	s.Require().NoError(err)

	executor, err := mother.CloneWithName(namePrefix + "executor")
	s.Require().NoError(err)

	// Prepare ping executor & target

	s.Require().NoError(target.Execution().Start(ctx))
	s.Require().NoError(executor.Execution().Start(ctx))

	targetEndpoint, err := target.Network().GetServiceEndpoint(gopingPort)
	s.Require().NoError(err)
	s.T().Logf("targetEndpoint: %v", targetEndpoint)

	s.T().Log("Starting ping test. It takes a while.")
	for i := 0; i < numOfTests; i++ {
		startTime := time.Now()

		output, err := executor.Execution().ExecuteCommand(ctx, "goping", "ping", "-q",
			"-c", fmt.Sprint(numOfPingPackets),
			"-t", packetTimeout.String(),
			"-m", "packetloss",
			targetEndpoint)
		s.Require().NoError(err)

		elapsed := time.Since(startTime)
		s.T().Logf("i: %d, test took %d seconds, output: `%s`", i, int64(elapsed.Seconds()), output)

		gotPacketloss, err := strconv.ParseFloat(output, 64)
		s.Require().NoError(err, fmt.Sprintf("error parsing output: `%s`", output))

		s.Assert().Zero(gotPacketloss)
	}
}
