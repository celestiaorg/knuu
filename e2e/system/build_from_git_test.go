package system

import (
	"context"
	"strings"

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
	err = target.Build().SetGitRepo(ctx, builder.GitContext{
		Repo:     "https://github.com/celestiaorg/knuu.git",
		Branch:   "test/build-from-git", // This branch has a Dockerfile and is protected as to not be deleted
		Username: "",
		Password: "",
	})
	s.Require().NoError(err)

	s.T().Log("Image built")

	s.T().Cleanup(func() {
		if err := target.Execution().Destroy(ctx); err != nil {
			s.T().Logf("Error cleaning up knuu: %v", err)
		}
	})

	s.Require().NoError(target.Build().Commit(ctx))

	s.T().Logf("Starting instance")
	s.Require().NoError(target.Execution().Start(ctx))

	s.T().Logf("Instance started")

	// The file is created by the dockerfile in the repo,
	// so to make sure it is built correctly, we check the file
	data, err := target.Storage().GetFileBytes(ctx, "/test.txt")
	s.Require().NoError(err)

	data = []byte(strings.TrimSpace(string(data)))
	s.Assert().Equal([]byte("Hello, World!"), data, "File bytes do not match")
}
func (s *Suite) TestBuildFromGitWithModifications() {
	const namePrefix = "build-from-git-with-modifications"
	s.T().Parallel()

	// Setup
	target, err := s.Knuu.NewInstance(namePrefix)
	s.Require().NoError(err)

	ctx := context.Background()
	// This is a blocking call which builds the image from git repo
	err = target.Build().SetGitRepo(ctx, builder.GitContext{
		Repo:     "https://github.com/celestiaorg/knuu.git",
		Branch:   "test/build-from-git", // This branch has a Dockerfile and is protected as to not be deleted
		Username: "",
		Password: "",
	})
	s.Require().NoError(err)

	s.Require().NoError(target.Build().SetStartCommand("sleep", "infinity"))

	err = target.Storage().AddFileBytes([]byte("Hello, world!"), "/home/hello.txt", "root:root")
	s.Require().NoError(err, "Error adding file")

	s.Require().NoError(target.Build().Commit(ctx))

	s.T().Cleanup(func() {
		if err := target.Execution().Destroy(ctx); err != nil {
			s.T().Logf("Error cleaning up knuu: %v", err)
		}
	})

	s.Require().NoError(target.Execution().Start(ctx))

	data, err := target.Storage().GetFileBytes(ctx, "/home/hello.txt")
	s.Require().NoError(err, "Error getting file bytes")

	s.Assert().Equal([]byte("Hello, world!"), data, "File bytes do not match")
}
