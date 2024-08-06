package system

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/celestiaorg/knuu/pkg/builder"
)

func (s *Suite) TestBuildFromGit() {
	const namePrefix = "build-from-git"
	s.T().Parallel()

	// Setup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	s.T().Log("Creating new instance")
	target, err := s.Knuu.NewInstance(namePrefix)
	if err != nil {
		s.T().Fatalf("Error creating new instance: %v", err)
	}

	s.T().Log("Building the image")

	// This is a blocking call which builds the image from git repo
	err = target.SetGitRepo(ctx, builder.GitContext{
		Repo:     "https://github.com/celestiaorg/knuu.git",
		Branch:   "test/build-from-git", // This branch has a Dockerfile and is protected as to not be deleted
		Username: "",
		Password: "",
	})
	if err != nil {
		s.T().Fatalf("Error setting git repo: %v", err)
	}

	s.T().Log("Image built")

	s.T().Cleanup(func() {
		s.T().Log("Cleaning up instance")
		if err := target.Destroy(ctx); err != nil {
			s.T().Logf("Error cleaning up knuu: %v", err)
		}
	})

	s.T().Log("Committing changes")
	if err := target.Commit(); err != nil {
		s.T().Fatalf("Error committing changes: %v", err)
	}

	s.T().Log("Starting instance")
	if err := target.Start(ctx); err != nil {
		s.T().Fatalf("Error starting instance: %v", err)
	}

	s.T().Log("Instance started")

	s.T().Log("Getting file bytes")
	// The file is created by the dockerfile in the repo,
	// so to make sure it is built correctly, we check the file
	data, err := target.GetFileBytes(ctx, "/test.txt")
	if err != nil {
		s.T().Fatalf("Error getting file bytes: %v", err)
	}

	data = []byte(strings.TrimSpace(string(data)))
	if !bytes.Equal([]byte("Hello, World!"), data) {
		s.T().Fatalf("File bytes do not match. Expected 'Hello, World!', got '%s'", string(data))
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s.T().Log("Creating new instance")
	target, err := s.Knuu.NewInstance(namePrefix)
	if err != nil {
		s.T().Fatalf("Error creating new instance: %v", err)
	}

	s.T().Log("Setting git repo")
	err = retryOperation(func() error {
		return target.SetGitRepo(ctx, builder.GitContext{
			Repo:     "https://github.com/celestiaorg/knuu.git",
			Branch:   "test/build-from-git",
			Username: "",
			Password: "",
		})
	}, maxRetries)
	if err != nil {
		s.T().Fatalf("Error setting git repo: %v", err)
	}

	s.T().Log("Setting command")
	err = retryOperation(func() error {
		return target.SetCommand("sleep", "infinity")
	}, maxRetries)
	if err != nil {
		s.T().Fatalf("Error setting command: %v", err)
	}

	s.T().Log("Adding file")
	err = retryOperation(func() error {
		return target.AddFileBytes([]byte("Hello, world!"), "/home/hello.txt", "root:root")
	}, maxRetries)
	if err != nil {
		s.T().Fatalf("Error adding file: %v", err)
	}

	s.T().Log("Committing changes")
	err = retryOperation(func() error {
		return target.Commit()
	}, maxRetries)
	if err != nil {
		s.T().Fatalf("Error committing changes: %v", err)
	}

	s.T().Cleanup(func() {
		s.T().Log("Cleaning up instance")
		if err := target.Destroy(ctx); err != nil {
			s.T().Logf("Error cleaning up knuu: %v", err)
		}
	})

	s.T().Log("Starting instance")
	err = retryOperation(func() error {
		return target.Start(ctx)
	}, maxRetries)
	if err != nil {
		s.T().Fatalf("Error starting instance: %v", err)
	}

	s.T().Log("Getting file bytes")
	var data []byte
	err = retryOperation(func() error {
		var err error
		data, err = target.GetFileBytes(ctx, "/home/hello.txt")
		return err
	}, maxRetries)
	if err != nil {
		s.T().Fatalf("Error getting file bytes: %v", err)
	}

	if !bytes.Equal([]byte("Hello, world!"), data) {
		s.T().Fatalf("File bytes do not match. Expected 'Hello, world!', got '%s'", string(data))
	}

	s.T().Log("Test completed successfully")
}
