package basic

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

func TestTshark(t *testing.T) {
	t.Parallel()
	// Setup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	knuu, err := knuu.New(ctx)
	require.NoError(t, err, "Error creating knuu")

	scope := knuu.Scope()
	t.Log(scope)

	instance, err := knuu.NewInstance("alpine")
	require.NoError(t, err, "Error creating instance")

	err = instance.SetImage(ctx, "docker.io/alpine:latest")
	require.NoError(t, err, "Error setting image")

	err = instance.SetCommand("sleep", "infinity")
	require.NoError(t, err, "Error setting command")

	err = instance.EnableTsharkCollector(
		"10Gi", // Example volume size
		os.Getenv("S3_ACCESS_KEY"),
		os.Getenv("S3_SECRET_KEY"),
		os.Getenv("S3_REGION"),
		os.Getenv("S3_BUCKET_NAME"),
		"tshark/"+scope,
	)
	require.NoError(t, err, "Error enabling tshark collector")

	err = instance.Commit()
	require.NoError(t, err, "Error committing instance")

	t.Cleanup(func() {
		require.NoError(t, instance.Destroy(ctx))
	})

	// Test logic

	err = instance.Start(ctx)
	require.NoError(t, err, "Error starting instance")

	err = instance.WaitInstanceIsRunning(ctx)
	require.NoError(t, err, "Error waiting for instance to be running")

	wget, err := instance.ExecuteCommand(ctx, "echo", "Hello World!")
	require.NoError(t, err, "Error executing command")

	// wait for 2 minutes to upload network traces to s3
	time.Sleep(2 * time.Minute)

	assert.Contains(t, wget, "Hello World!")
}
