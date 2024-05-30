package basic

import (
	"context"
	"log"
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
	if err != nil {
		t.Fatalf("Error creating knuu: %v", err)
	}

	scope := knuu.Scope()
	log.Println(scope)

	instance, err := knuu.NewInstance("alpine")
	if err != nil {
		t.Fatalf("Error creating instance '%v':", err)
	}
	err = instance.SetImage(ctx, "docker.io/alpine:latest")
	if err != nil {
		t.Fatalf("Error setting image: %v", err)
	}
	err = instance.SetCommand("sleep", "infinity")
	if err != nil {
		t.Fatalf("Error setting command: %v", err)
	}
	err = instance.EnableTsharkCollector(
		"10Gi", // Example volume size
		os.Getenv("S3_ACCESS_KEY"),
		os.Getenv("S3_SECRET_KEY"),
		os.Getenv("S3_REGION"),
		os.Getenv("S3_BUCKET_NAME"),
		"tshark/"+scope,
	)
	if err != nil {
		t.Fatalf("Error enabling tshark collector: %v", err)
	}
	err = instance.Commit()
	if err != nil {
		t.Fatalf("Error committing instance: %v", err)
	}

	t.Cleanup(func() {
		require.NoError(t, instance.Destroy(ctx))
	})

	// Test logic

	err = instance.Start(ctx)
	if err != nil {
		t.Fatalf("Error starting instance: %v", err)
	}
	err = instance.WaitInstanceIsRunning(ctx)
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}
	wget, err := instance.ExecuteCommand(ctx, "echo", "Hello World!")
	if err != nil {
		t.Fatalf("Error executing command '%v':", err)
	}

	// wait for 10 minutes
	time.Sleep(10 * time.Minute)

	assert.Contains(t, wget, "Hello World!")
}
