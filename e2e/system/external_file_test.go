package system

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

func (s *Suite) TestExternalFile() {
	const namePrefix = "external-file"
	// Setup

	ctx := context.Background()
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	s.Require().NoError(err)

	server := s.createNginxInstance(ctx, namePrefix+"-server")

	// copy resources/html/index.html to /tmp/index.html
	srcFile, err := os.Open(resourcesHTML + "/index.html")
	s.Require().NoError(err)
	defer srcFile.Close()

	// Create the destination file
	htmlTmpPath := filepath.Join(os.TempDir(), "index.html")
	dstFile, err := os.Create(htmlTmpPath)
	s.Require().NoError(err)
	defer dstFile.Close()

	// Copy the contents of the source file into the destination file
	_, err = io.Copy(dstFile, srcFile)
	s.Require().NoError(err)

	// Ensure that the copy is successful by syncing the written data to the disk
	s.Require().NoError(dstFile.Sync())

	err = server.Storage().AddFile(htmlTmpPath, nginxHTMLPath+"/index.html", "0:0")
	s.Require().NoError(err)

	s.Require().NoError(server.Build().Commit(ctx))

	// Test logic
	serverIP, err := server.Network().GetIP(ctx)
	s.Require().NoError(err)

	s.Require().NoError(server.Execution().Start(ctx))

	wget, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", serverIP)
	s.Require().NoError(err)

	s.Assert().Contains(wget, "Hello World!")
}
