package sidecars

import (
	"context"
)

func (s *Suite) TestDownloadFileFromRunningSidecar() {
	const (
		namePrefix  = "download-file-running-sidecar"
		fileContent = "Hello World!"
		filePath    = "/hello.txt"
	)
	ctx := context.Background()

	sidecar := &testSidecar{
		StartCommand: []string{"sh", "-c", "sleep infinity"},
	}
	s.startNewInstanceWithSidecar(ctx, namePrefix, sidecar)

	// Create a file in the sidecar instance
	out, err := sidecar.Instance().Execution().ExecuteCommand(ctx, "echo", "-n", fileContent, ">", filePath)
	s.Require().NoError(err, "executing command output: %v", out)

	gotContent, err := sidecar.Instance().Storage().GetFileBytes(ctx, filePath)
	s.Require().NoError(err, "Failed to read file %s from sidecar: %v", filePath, err)

	s.Assert().Equal(fileContent, string(gotContent))
}
