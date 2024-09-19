package basic

import (
	"context"
	"fmt"
	"io"
	"time"
)

// func (s *Suite) XX_TestLogs() {
// 	const namePrefix = "logs"
// 	ctx := context.Background()

// 	target, err := s.Knuu.NewInstance(namePrefix + "-target")
// 	s.Require().NoError(err)

// 	s.Require().NoError(target.Build().SetImage(ctx, alpineImage))
// 	s.Require().NoError(target.Build().SetStartCommand("sleep", "infinity"))
// 	s.Require().NoError(target.Build().Commit(ctx))
// 	s.Require().NoError(target.Execution().Start(ctx))

// 	// Perform the test
// 	expectedOutput := "Hello World"
// 	output, err := target.Execution().ExecuteCommand(ctx, "echo", expectedOutput)
// 	s.Require().NoError(err)

// 	output = strings.TrimSpace(output)
// 	s.Assert().Equal(expectedOutput, output)
// }

func (s *Suite) TestLogs() {
	const (
		namePrefix     = "logs"
		expectedLogMsg = "Hello World"
	)
	ctx := context.Background()

	// Create a new instance
	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)

	// Set the image and start command to generate logs
	s.Require().NoError(target.Build().SetImage(ctx, alpineImage))
	s.Require().NoError(target.Build().SetStartCommand("sh", "-c", fmt.Sprintf("while true; do echo '%s'; sleep 1; done", expectedLogMsg)))
	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Execution().Start(ctx))

	// Wait for a short duration to allow log generation
	time.Sleep(5 * time.Second)

	logStream, err := target.Monitoring().Logs(ctx)
	s.Require().NoError(err)
	defer logStream.Close()

	logs, err := io.ReadAll(logStream)
	s.Require().NoError(err)

	logOutput := string(logs)
	s.Contains(logOutput, expectedLogMsg)
}

func (s *Suite) TestLogsWithSidecar() {
	const (
		namePrefix     = "logs-sidecar"
		expectedLogMsg = "Hello World"
	)
	ctx := context.Background()

	// Create a new instance
	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)

	sidecar, err := s.Knuu.NewInstance(namePrefix + "-sidecar")
	s.Require().NoError(err)
	s.Require().NoError(sidecar.Build().SetImage(ctx, alpineImage))
	s.Require().NoError(sidecar.Build().SetStartCommand("sh", "-c", fmt.Sprintf("while true; do echo '%s'; sleep 1; done", expectedLogMsg)))
	s.Require().NoError(sidecar.Build().Commit(ctx))
	s.Require().NoError(sidecar.Execution().Start(ctx))

	s.Require().NoError(target.Build().SetImage(ctx, alpineImage))
	s.Require().NoError(target.Build().SetStartCommand("sh", "-c", "sleep infinity"))
	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Execution().Start(ctx))

	// Wait for a short duration to allow log generation
	time.Sleep(5 * time.Second)

	logStream, err := sidecar.Monitoring().Logs(ctx)
	s.Require().NoError(err)
	defer logStream.Close()

	logs, err := io.ReadAll(logStream)
	s.Require().NoError(err)

	logOutput := string(logs)
	s.Contains(logOutput, expectedLogMsg)
}
