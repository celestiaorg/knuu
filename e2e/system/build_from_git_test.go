package system

import (
	"context"
	"strings"
	"time"

	"github.com/celestiaorg/knuu/pkg/builder"
)

func (s *Suite) TestBuildFromGit() {
	const namePrefix = "build-from-git"
	s.T().Parallel()

	// Setup
	ctx := context.Background()

	target, err := s.Knuu.NewInstance(namePrefix)
	s.Require().NoError(err)

	s.T().Log("Building the image")

	// This is a blocking call which builds the image from git repo
	err = target.SetGitRepo(ctx, builder.GitContext{
		Repo:     "https://github.com/celestiaorg/knuu.git",
		Branch:   "test/build-from-git", // This branch has a Dockerfile and is protected as to not be deleted
		Username: "",
		Password: "",
	})
	s.Require().NoError(err)

	s.T().Log("Image built")

	s.T().Cleanup(func() {
		if err := target.Destroy(ctx); err != nil {
			s.T().Logf("Error cleaning up knuu: %v", err)
		}
	})

	s.Require().NoError(target.Commit())

	s.T().Logf("Starting instance")
	s.Require().NoError(target.Start(ctx))

	s.T().Logf("Instance started")

	// The file is created by the dockerfile in the repo,
	// so to make sure it is built correctly, we check the file
	data, err := target.GetFileBytes(ctx, "/test.txt")
	s.Require().NoError(err)

	data = []byte(strings.TrimSpace(string(data)))
	s.Assert().Equal([]byte("Hello, World!"), data, "File bytes do not match")
}

func (s *Suite) TestBuildFromGitWithModifications() {
	const (
		namePrefix = "build-from-git-with-modifications"
		maxRetries = 3
	)
	s.T().Parallel()

	// Setup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	target, err := s.Knuu.NewInstance(namePrefix)
	s.Require().NoError(err, "Error creating new instance")

	// This is a blocking call which builds the image from git repo
	err = retryOperation(func() error {
		return target.SetGitRepo(ctx, builder.GitContext{
			Repo:     "https://github.com/celestiaorg/knuu.git",
			Branch:   "test/build-from-git", // This branch has a Dockerfile and is protected as to not be deleted
			Username: "",
			Password: "",
		})
	}, maxRetries)
	s.Require().NoError(err, "Error setting git repo")

	err = retryOperation(func() error {
		return target.SetCommand("sleep", "infinity")
	}, maxRetries)
	s.Require().NoError(err, "Error setting command")

	err = retryOperation(func() error {
		return target.AddFileBytes([]byte("Hello, world!"), "/home/hello.txt", "root:root")
	}, maxRetries)
	s.Require().NoError(err, "Error adding file")

	err = retryOperation(func() error {
		return target.Commit()
	}, maxRetries)
	s.Require().NoError(err, "Error committing changes")

	s.T().Cleanup(func() {
		if err := target.Destroy(ctx); err != nil {
			s.T().Logf("Error cleaning up knuu: %v", err)
		}
	})

	err = retryOperation(func() error {
		return target.Start(ctx)
	}, maxRetries)
	s.Require().NoError(err, "Error starting instance")

	var data []byte
	err = retryOperation(func() error {
		var err error
		data, err = target.GetFileBytes(ctx, "/home/hello.txt")
		return err
	}, maxRetries)
	s.Require().NoError(err, "Error getting file bytes")

	s.Assert().Equal([]byte("Hello, world!"), data, "File bytes do not match")
}
