package system

import (
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/celestiaorg/knuu/pkg/minio"
)

func TestBuildFromGit(t *testing.T) {
	t.Parallel()

	// Setup
	ctx := context.Background()

	// The default image builder is kaniko here
	kn, err := knuu.New(ctx, knuu.Options{})
	require.NoError(t, err, "Error creating knuu")

	target, err := kn.NewInstance("git-builder")
	require.NoError(t, err, "Error creating instance")

	t.Log("Building the image")

	// This is a blocking call which builds the image from git repo
	err = target.SetGitRepo(ctx, builder.GitContext{
		Repo:     "https://github.com/celestiaorg/knuu.git",
		Branch:   "test/build-from-git", // This branch has a Dockerfile and is protected as to not be deleted
		Username: "",
		Password: "",
	})
	require.NoError(t, err, "Error setting git repo")

	t.Log("Image built")

	t.Cleanup(func() {
		if err := target.Destroy(ctx); err != nil {
			t.Logf("Error destroying instance: %v", err)
		}
	})

	require.NoError(t, target.Commit())

	t.Logf("Starting instance")

	assert.NoError(t, target.Start(ctx))

	t.Logf("Instance started")

	// The file is created by the dockerfile in the repo,
	// so to make sure it is built correctly, we check the file
	data, err := target.GetFileBytes(ctx, "/test.txt")
	require.NoError(t, err, "Error getting file bytes")

	data = []byte(strings.TrimSpace(string(data)))
	assert.Equal(t, []byte("Hello, World!"), data, "File bytes do not match")
}
func TestBuildFromGitWithModifications(t *testing.T) {
	t.Parallel()

	// Setup
	ctx := context.Background()

	k8sClient, err := k8s.NewClient(ctx, knuu.DefaultTestScope(), logrus.New())
	require.NoError(t, err, "Error creating k8s client")

	// Since we are modifying the git repo,
	// we need to setup minio to allow the builder to push the changes
	minioClient, err := minio.New(ctx, k8sClient)
	require.NoError(t, err, "Error creating minio client")

	// The default image builder is kaniko here
	kn, err := knuu.New(ctx, knuu.Options{
		K8sClient:   k8sClient,
		MinioClient: minioClient,
	})
	require.NoError(t, err, "Error creating knuu")

	sampleInstance, err := kn.NewInstance("git-builder")
	require.NoError(t, err, "Error creating instance")

	// This is a blocking call which builds the image from git repo
	err = sampleInstance.SetGitRepo(ctx, builder.GitContext{
		Repo:     "https://github.com/celestiaorg/knuu.git",
		Branch:   "test/build-from-git", // This branch has a Dockerfile and is protected as to not be deleted
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
