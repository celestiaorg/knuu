package system

import (
	"io"
	"os"
	"testing"

	"github.com/celestiaorg/knuu/pkg/knuu"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalFile(t *testing.T) {
	t.Parallel()
	// Setup

	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	server, err := knuu.NewInstance("server")
	if err != nil {
		t.Fatalf("Error creating instance '%v':", err)
	}
	err = server.SetImage("docker.io/nginx:latest")
	if err != nil {
		t.Fatalf("Error setting image '%v':", err)
	}
	server.AddPortTCP(80)
	_, err = server.ExecuteCommand("mkdir", "-p", "/usr/share/nginx/html")
	if err != nil {
		t.Fatalf("Error executing command '%v':", err)
	}
	// copy resources/html/index.html to /tmp/index.html
	srcFile, err := os.Open("resources/html/index.html")
	if err != nil {
		t.Fatalf("Error opening source file '%v':", err)
	}
	defer srcFile.Close()

	// Create the destination file
	dstFile, err := os.Create("/tmp/index.html")
	if err != nil {
		t.Fatalf("Error creating destination file '%v':", err)
	}
	defer dstFile.Close()

	// Copy the contents of the source file into the destination file
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		t.Fatalf("Error copying contents '%v':", err)
	}

	// Ensure that the copy is successful by syncing the written data to the disk
	err = dstFile.Sync()
	if err != nil {
		t.Fatalf("Error syncing data to disk '%v':", err)
	}

	err = server.AddFile("/tmp/index.html", "/usr/share/nginx/html/index.html", "0:0")
	if err != nil {
		t.Fatalf("Error adding file '%v':", err)
	}
	err = server.Commit()
	if err != nil {
		t.Fatalf("Error committing instance: %v", err)
	}

	t.Cleanup(func() {
		require.NoError(t, knuu.BatchDestroy(executor.Instance, server))
	})

	// Test logic

	serverIP, err := server.GetIP()
	if err != nil {
		t.Fatalf("Error getting IP '%v':", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("Error starting instance: %v", err)
	}
	err = server.WaitInstanceIsRunning()
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}

	wget, err := executor.ExecuteCommand("wget", "-q", "-O", "-", serverIP)
	if err != nil {
		t.Fatalf("Error executing command '%v':", err)
	}

	assert.Contains(t, wget, "Hello World!")
}
