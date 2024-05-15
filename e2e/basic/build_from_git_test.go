package basic

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

// This test is just an example to show how to
// setup the test instance to be built from a git repo
func TestBuildFromGit(t *testing.T) {
	t.Parallel()
	// Setup

	// This code is a bit dirty due to the current limitations of knuu
	// After refactoring knuu, this test must be either removed or updated
	require.NoError(t, os.Setenv("KNUU_BUILDER", "kubernetes"), "Error setting KNUU_BUILDER Env")
	require.NoError(t, knuu.CleanUp(), "Error cleaning up knuu")
	require.NoError(t, knuu.Initialize(), "Error initializing knuu")

	instance, err := knuu.NewInstance("my-instance")
	require.NoError(t, err, "Error creating instance")

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	// This is a blocking call which builds the image from git repo
	err = instance.SetGitRepo(ctx, builder.GitContext{
		Repo:   "https://github.com/celestiaorg/celestia-app.git",
		Branch: "main",
		// Commit:   "5ce94f4f010e366df301d25cd5d797c3147ff884",
		Username: "",
		Password: "",
	})
	require.NoError(t, err, "Error setting git repo")

	require.NoError(t, instance.SetCommand("sleep", "infinity"), "Error setting command")

	err = instance.AddFileBytes([]byte("Hello, world!"), "/home/hello.txt", "root:root")
	require.NoError(t, err, "Error adding file")

	require.NoError(t, instance.Commit(), "Error committing instance")

	t.Cleanup(func() {
		require.NoError(t, knuu.BatchDestroy(instance))
	})

	// Test logic

	require.NoError(t, instance.Start(), "Error starting instance")

	data, err := instance.GetFileBytes("/home/hello.txt")
	require.NoError(t, err, "Error getting file bytes")

	require.Equal(t, []byte("Hello, world!"), data, "File bytes do not match")
}
