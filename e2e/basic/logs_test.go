package basic

import (
	"context"
	"fmt"
	"io"
	"time"
)

const expectedLogMsg = "Hello World"

func (s *Suite) TestLogs() {
	const namePrefix = "logs"
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
