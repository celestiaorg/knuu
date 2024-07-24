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
	const namePrefix = "build-from-git-with-modifications"
	s.T().Parallel()

	// Setup
	target, err := s.Knuu.NewInstance(namePrefix)
	s.Require().NoError(err)

	ctx := context.Background()
	// This is a blocking call which builds the image from git repo
	err = target.SetGitRepo(ctx, builder.GitContext{
		Repo:     "https://github.com/celestiaorg/knuu.git",
		Branch:   "test/build-from-git", // This branch has a Dockerfile and is protected as to not be deleted
		Username: "",
		Password: "",
	})
	s.Require().NoError(err)

	s.Require().NoError(target.SetCommand("sleep", "infinity"))

	err = target.AddFileBytes([]byte("Hello, world!"), "/home/hello.txt", "root:root")
	s.Require().NoError(err, "Error adding file")

	s.Require().NoError(target.Commit())

	s.T().Cleanup(func() {
		if err := target.Destroy(ctx); err != nil {
			s.T().Logf("Error cleaning up knuu: %v", err)
		}
	})

	s.Require().NoError(target.Start(ctx))

	data, err := target.GetFileBytes(ctx, "/home/hello.txt")
	s.Require().NoError(err, "Error getting file bytes")

	s.Assert().Equal([]byte("Hello, world!"), data, "File bytes do not match")
}
