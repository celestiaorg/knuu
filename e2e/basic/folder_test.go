package basic

import (
	"testing"

	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFolder(t *testing.T) {
	t.Parallel()
	// Setup

	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	// Create and commit the instance
	web := assertCreateInstanceNginxWithVolumeOwner(t, "web")
	err = web.AddFile("resources/html/index.html", "/usr/share/nginx/html/index.html", "0:0")
	if err != nil {
		t.Fatalf("Error adding file to '%v': %v", "web", err)
	}

	t.Cleanup(func() {
		require.NoError(t, knuu.BatchDestroy(executor.Instance, web))
	})

	// Test logic
	webIP, err := web.GetIP()
	if err != nil {
		t.Fatalf("Error getting IP '%v':", err)
	}

	err = web.Start()
	if err != nil {
		t.Fatalf("Error starting instance: %v", err)
	}
	err = web.WaitInstanceIsRunning()
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}

	wget, err := executor.ExecuteCommand("wget", "-q", "-O", "-", webIP)
	if err != nil {
		t.Fatalf("Error executing command '%v':", err)
	}

	assert.Contains(t, wget, "Hello World!")
}
