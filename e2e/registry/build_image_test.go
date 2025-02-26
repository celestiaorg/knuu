package system

import (
	"context"
	"strings"
	"time"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/instance"
)

const (
	gitRepo   = "https://github.com/celestiaorg/knuu.git"
	gitBranch = "test/build-from-git" // This branch has a Dockerfile and is protected as to not be deleted
)

func (s *Suite) TestBuildFromGit() {
	const namePrefix = "registry-build-from-git"

	// Setup
	ctx := context.Background()

	target, err := s.createAndStartBuildFromGitInstance(ctx, namePrefix)
	s.Require().NoError(err)

	s.T().Log("Getting file bytes")
	// The file is created by the dockerfile in the repo,
	// so to make sure it is built correctly, we check the file
	data, err := target.Storage().GetFileBytes(ctx, "/test.txt")
	s.Require().NoError(err)

	data = []byte(strings.TrimSpace(string(data)))
	s.Assert().Equal([]byte("Hello, World!"), data, "file bytes do not match.")
}

func (s *Suite) TestRegistryCacheWithBuildFromGit() {
	const namePrefix = "cache-registry-build-from-git"

	// Setup
	ctx := context.Background()

	_, err := s.createAndStartBuildFromGitInstance(ctx, namePrefix)
	s.Require().NoError(err)

	startTime := time.Now()

	_, err = s.createAndStartBuildFromGitInstance(ctx, "2nd-"+namePrefix)
	s.Require().NoError(err)

	duration := time.Since(startTime)
	s.T().Logf("Time taken: %s", duration)
}

func (s *Suite) createAndStartBuildFromGitInstance(ctx context.Context, namePrefix string) (*instance.Instance, error) {
	s.T().Logf("Creating new instance %s", namePrefix)
	target, err := s.Knuu.NewInstance(namePrefix)
	s.Require().NoError(err)

	s.T().Log("Building the image")

	// This is a blocking call which builds the image from git repo
	err = target.Build().SetGitRepo(ctx, builder.GitContext{
		Repo:     gitRepo,
		Branch:   gitBranch,
		Username: "",
		Password: "",
	})
	s.Require().NoError(err)
	s.T().Log("Image built")

	s.Require().NoError(target.Build().Commit(ctx))

	s.T().Logf("Starting instance %s", namePrefix)
	s.Require().NoError(target.Execution().Start(ctx))

	s.T().Log("Instance started")
	return target, nil
}
