package sidecars

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

func (s *Suite) TestLogsWithSidecar() {
	const (
		namePrefix     = "logs-sidecar"
		expectedLogMsg = "Hello World"
		timeout        = 10 * time.Second
		interval       = 1 * time.Second
	)
	ctx := context.Background()

	sidecar := &testSidecar{
		StartCommand: []string{
			"sh", "-c",
			fmt.Sprintf("while true; do echo '%s'; sleep 1; done", expectedLogMsg),
		},
	}
	s.startNewInstanceWithSidecar(ctx, namePrefix, sidecar)

	// Wait for a short duration to allow log generation
	s.Require().Eventually(func() bool {
		logStream, err := sidecar.Instance().Monitoring().Logs(ctx)
		if err != nil {
			return false
		}
		defer logStream.Close()

		logs, err := io.ReadAll(logStream)
		if err != nil {
			return false
		}

		return strings.Contains(string(logs), expectedLogMsg)
	}, timeout, interval, "failed to get expected log message")
}
