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
	out, err := sidecar.Instance().Execution().ExecuteCommand(ctx, "echo", "-n", fileContent, ">", filePath)
	s.Require().NoError(err, "executing command output: %v", out)

	gotContent, err := sidecar.Instance().Storage().GetFileBytes(ctx, filePath)
	s.Require().NoError(err, "Error getting file bytes")

	s.Assert().Equal(fileContent, string(gotContent))
}
