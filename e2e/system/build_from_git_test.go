package system

import (
	"bytes"
	"context"
	"strings"

	"github.com/celestiaorg/knuu/pkg/builder"
)

const (
	gitRepo   = "https://github.com/celestiaorg/knuu.git"
	gitBranch = "test/build-from-git" // This branch has a Dockerfile and is protected as to not be deleted
)

func (s *Suite) TestBuildFromGit() {
	const namePrefix = "build-from-git"
	s.T().Parallel()

	// Setup
	ctx := context.Background()

	s.T().Log("Creating new instance")
	target, err := s.Knuu.NewInstance(namePrefix)
	if err != nil {
		s.Require().NoError(err, "Error creating new instance")
	}

	s.T().Log("Building the image")

	// This is a blocking call which builds the image from git repo
	err = target.Build().SetGitRepo(ctx, builder.GitContext{
		Repo:     gitRepo,
		Branch:   gitBranch,
		Username: "",
		Password: "",
	})
	s.Require().NoError(err, "Error setting git repo")

	s.T().Log("Image built")

	s.T().Cleanup(func() {
		if err := target.Execution().Destroy(ctx); err != nil {
			s.T().Logf("Error cleaning up knuu: %v", err)
		}
	})

	s.Require().NoError(target.Build().Commit(ctx))

	s.T().Logf("Starting instance")
	s.Require().NoError(target.Execution().Start(ctx))

	s.T().Log("Instance started")

	s.T().Log("Getting file bytes")
	// The file is created by the dockerfile in the repo,
	// so to make sure it is built correctly, we check the file
	data, err := target.Storage().GetFileBytes(ctx, "/test.txt")
	s.Require().NoError(err)

	data = []byte(strings.TrimSpace(string(data)))
	if !bytes.Equal([]byte("Hello, World!"), data) {
		s.Require().NoError(err, "File bytes do not match. Expected 'Hello, World!', got '%s'", string(data))
	}

	s.T().Log("Test completed successfully")
}

func (s *Suite) TestBuildFromGitWithModifications() {
	const (
		namePrefix = "build-from-git-with-modifications"
		maxRetries = 3
	)
	s.T().Parallel()

	// Setup
	ctx := context.Background()

	s.T().Log("Creating new instance")
	target, err := s.Knuu.NewInstance(namePrefix)
	if err != nil {
		s.Require().NoError(err, "Error creating new instance")
	}

	s.T().Log("Setting git repo")
	err = s.retryOperation(func() error {
		return target.Build().SetGitRepo(ctx, builder.GitContext{
			Repo:     gitRepo,
			Branch:   gitBranch,
			Username: "",
			Password: "",
		})
	}, maxRetries)
	s.Require().NoError(err, "Error setting git repo")

	s.Require().NoError(target.Build().SetStartCommand("sleep", "infinity"))

	const (
		filePath = "/home/hello.txt"
		fileData = "Hello, world!"
	)

	err = target.Storage().AddFileBytes([]byte(fileData), filePath, "root:root")
	s.Require().NoError(err, "Error adding file")

	s.T().Log("Committing changes")
	s.Require().NoError(target.Build().Commit(ctx))

	s.T().Cleanup(func() {
		s.T().Log("Cleaning up instance")
		if err := target.Execution().Destroy(ctx); err != nil {
			s.T().Logf("error destroying instance: %v", err)
		}
	})

	s.T().Log("Starting instance")
	s.Require().NoError(target.Execution().Start(ctx))

	var data []byte
	err = s.retryOperation(func() error {
		var err error
		data, err = target.Storage().GetFileBytes(ctx, filePath)
		return err
	}, maxRetries)

	s.Require().NoError(err, "Error getting file bytes")
	s.Assert().Equal([]byte(fileData), data, "file bytes do not match.")
}
