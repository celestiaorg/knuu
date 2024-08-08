package system

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/celestiaorg/knuu/pkg/instance"

	"github.com/google/uuid"
)

func (s *Suite) TestFile() {
	const namePrefix = "file"
	s.T().Parallel()
	// Setup

	ctx := context.Background()
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	s.Require().NoError(err)

	serverfile := s.createNginxInstanceWithVolume(ctx, namePrefix+"-serverfile")

	err = serverfile.Storage().AddFile(resourcesHTML+"/index.html", nginxHTMLPath+"/index.html", "0:0")
	s.Require().NoError(err)

	s.Require().NoError(serverfile.Build().Commit(ctx))

	s.T().Cleanup(func() {
		err := instance.BatchDestroy(ctx, serverfile, executor)
		if err != nil {
			s.T().Logf("Error destroying instance: %v", err)
		}
	})

	// Test logic

	serverfileIP, err := serverfile.Network().GetIP(ctx)
	s.Require().NoError(err)

	s.Require().NoError(serverfile.Execution().Start(ctx))

	wget, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", serverfileIP)
	s.Require().NoError(err)

	s.Assert().Contains(wget, "Hello World!")
}

func (s *Suite) TestDownloadFileFromRunningInstance() {
	const namePrefix = "download-file-running"
	s.T().Parallel()
	// Setup

	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)

	ctx := context.Background()
	s.Require().NoError(target.Build().SetImage(ctx, "alpine:latest"))
	s.Require().NoError(target.Build().SetArgs("tail", "-f", "/dev/null")) // Keep the container running
	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Execution().Start(ctx))

	s.T().Cleanup(func() {
		if err := target.Execution().Destroy(ctx); err != nil {
			s.T().Logf("error destroying instance: %v", err)
		}
	})

	// Test logic
	const (
		fileContent = "Hello World!"
		filePath    = "/hello.txt"
	)

	// Create a file in the target instance
	out, err := target.Execution().ExecuteCommand(ctx, "echo", "-n", fileContent, ">", filePath)
	s.Require().NoError(err, "executing command output: %v", out)

	gotContent, err := target.Storage().GetFileBytes(ctx, filePath)
	s.Require().NoError(err, "Error getting file bytes")

	s.Assert().Equal(fileContent, string(gotContent))
}

func (s *Suite) TestMinio() {
	const (
		namePrefix       = "minio"
		minioBucketName  = "knuu-e2e-test"
		minioPushTimeout = 1 * time.Minute
	)
	s.T().Parallel()
	// Setup
	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)

	ctx := context.Background()
	s.Require().NoError(target.Build().SetImage(ctx, "alpine:latest"))
	s.Require().NoError(target.Build().SetArgs("tail", "-f", "/dev/null")) // Keep the container running
	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Execution().Start(ctx))

	s.T().Cleanup(func() {
		if err := target.Execution().Destroy(ctx); err != nil {
			s.T().Logf("error destroying instance: %v", err)
		}
	})

	var (
		fileContent = "Hello World!"
		contentName = uuid.New().String()
	)
	s.T().Logf("contentName: %v", contentName)

	tmpFile, err := os.CreateTemp("", "hello.txt")
	s.Require().NoError(err, "Error creating temporary file")
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(fileContent)
	s.Require().NoError(err, "writing to temporary file")
	tmpFile.Close()

	// Write to file did not work, so need to open it again
	tmpFile, err = os.Open(tmpFile.Name())
	s.Require().NoError(err, "opening temporary file")
	defer tmpFile.Close()

	s.T().Logf("tmpFile name: %v", tmpFile.Name())

	mCtx, cancel := context.WithTimeout(ctx, minioPushTimeout)
	defer cancel()
	err = s.Knuu.MinioClient.Push(mCtx, tmpFile, contentName, minioBucketName)
	s.Require().NoError(err)

	url, err := s.Knuu.MinioClient.GetURL(ctx, contentName, minioBucketName)
	s.Require().NoError(err)

	resp, err := http.Get(url)
	s.Require().NoError(err)
	defer resp.Body.Close()

	gotContent, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	s.Assert().Equal(fileContent, string(gotContent))
}
