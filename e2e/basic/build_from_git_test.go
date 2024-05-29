package basic

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

// This test is just an example to show how to
// setup the test instance to be built from a git repo
func TestBuildFromGit(t *testing.T) {
	t.Parallel()
	// Setup

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	// The default image builder is kaniko here
	kn, err := knuu.New(ctx)
	require.NoError(t, err, "Error creating knuu")

	sampleInstance, err := kn.NewInstance("git-builder")
	require.NoError(t, err, "Error creating instance")

	// This is a blocking call which builds the image from git repo
	err = sampleInstance.SetGitRepo(ctx, builder.GitContext{
		Repo:   "https://github.com/celestiaorg/celestia-app.git",
		Branch: "main",
		// Commit:   "5ce94f4f010e366df301d25cd5d797c3147ff884",
		Username: "",
		Password: "",
	})
	require.NoError(t, err, "Error setting git repo")

	require.NoError(t, sampleInstance.SetCommand("sleep", "infinity"), "Error setting command")

	err = sampleInstance.AddFileBytes([]byte("Hello, world!"), "/home/hello.txt", "root:root")
	require.NoError(t, err, "Error adding file")

	require.NoError(t, sampleInstance.Commit(), "Error committing instance")

	t.Cleanup(func() {
		require.NoError(t, instance.BatchDestroy(ctx, sampleInstance))
	})

	require.NoError(t, sampleInstance.Start(ctx), "Error starting instance")

	data, err := sampleInstance.GetFileBytes(ctx, "/home/hello.txt")
	require.NoError(t, err, "Error getting file bytes")

	require.Equal(t, []byte("Hello, world!"), data, "File bytes do not match")
}
