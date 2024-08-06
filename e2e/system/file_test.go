package system

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/celestiaorg/knuu/pkg/instance"

	"github.com/google/uuid"
)

func (s *Suite) TestFile() {
	const (
		namePrefix = "file"
		maxRetries = 3
	)
	s.T().Parallel()

	// Setup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	s.T().Log("Creating executor instance")
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	if err != nil {
		s.T().Fatalf("Error creating executor instance: %v", err)
	}

	s.T().Log("Creating nginx instance with volume")
	serverfile := s.createNginxInstanceWithVolume(ctx, namePrefix+"-serverfile")

	s.T().Log("Adding file to nginx instance")
	err = retryOperation(func() error {
		return serverfile.AddFile(resourcesHTML+"/index.html", nginxHTMLPath+"/index.html", "0:0")
	}, maxRetries)
	if err != nil {
		s.T().Fatalf("Error adding file to nginx instance: %v", err)
	}

	s.T().Log("Committing changes")
	err = retryOperation(func() error {
		return serverfile.Commit()
	}, maxRetries)
	if err != nil {
		s.T().Fatalf("Error committing changes: %v", err)
	}

	s.T().Cleanup(func() {
		s.T().Log("Cleaning up instances")
		err := instance.BatchDestroy(ctx, serverfile, executor)
		if err != nil {
			s.T().Logf("Error destroying instances: %v", err)
		}
	})

	// Test logic
	s.T().Log("Getting server IP")
	var serverfileIP string
	err = retryOperation(func() error {
		var err error
		serverfileIP, err = serverfile.GetIP(ctx)
		return err
	}, maxRetries)
	if err != nil {
		s.T().Fatalf("Error getting server IP: %v", err)
	}

	s.T().Log("Starting server")
	err = retryOperation(func() error {
		return serverfile.Start(ctx)
	}, maxRetries)
	if err != nil {
		s.T().Fatalf("Error starting server: %v", err)
	}

	s.T().Log("Executing wget command")
	var wget string
	err = retryOperation(func() error {
		var err error
		wget, err = executor.ExecuteCommand(ctx, "wget", "-q", "-O", "-", serverfileIP)
		return err
	}, maxRetries)
	if err != nil {
		s.T().Fatalf("Error executing wget command: %v", err)
	}

	s.T().Log("Asserting wget output")
	if !strings.Contains(wget, "Hello World!") {
		s.T().Fatalf("Expected 'Hello World!' in wget output, but got: %s", wget)
	}

	s.T().Log("Test completed successfully")
}

func (s *Suite) TestDownloadFileFromRunningInstance() {
	const (
		namePrefix = "download-file-running"
	)
	s.T().Parallel()

	// Setup

	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)

	ctx := context.Background()
	s.Require().NoError(target.SetImage(ctx, "alpine:latest"))
	s.Require().NoError(target.SetArgs("tail", "-f", "/dev/null")) // Keep the container running
	s.Require().NoError(target.Commit())
	s.Require().NoError(target.Start(ctx))

	s.T().Cleanup(func() {
		if err := target.Destroy(ctx); err != nil {
			s.T().Logf("error destroying instance: %v", err)
		}
	})

	// Test logic
	const (
		fileContent = "Hello World!"
		filePath    = "/hello.txt"
	)

	// Create a file in the target instance
	out, err := target.ExecuteCommand(ctx, "echo", "-n", fileContent, ">", filePath)
	s.Require().NoError(err, "executing command output: %v", out)

	gotContent, err := target.GetFileBytes(ctx, filePath)
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
	s.Require().NoError(target.SetImage(ctx, "alpine:latest"))
	s.Require().NoError(target.SetArgs("tail", "-f", "/dev/null")) // Keep the container running
	s.Require().NoError(target.Commit())
	s.Require().NoError(target.Start(ctx))

	s.T().Cleanup(func() {
		if err := target.Destroy(ctx); err != nil {
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
