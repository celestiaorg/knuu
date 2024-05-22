package basic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/celestiaorg/knuu/pkg/knuu"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFile(t *testing.T) {
	t.Parallel()
	// Setup

	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	web, err := knuu.NewInstance("web")
	if err != nil {
		t.Fatalf("Error creating instance '%v':", err)
	}
	err = web.SetImage("docker.io/nginx:latest")
	if err != nil {
		t.Fatalf("Error setting image '%v':", err)
	}
	web.AddPortTCP(80)
	_, err = web.ExecuteCommand("mkdir", "-p", "/usr/share/nginx/html")
	if err != nil {
		t.Fatalf("Error executing command '%v':", err)
	}
	err = web.AddFile("resources/html/index.html", "/usr/share/nginx/html/index.html", "0:0")
	if err != nil {
		t.Fatalf("Error adding file '%v':", err)
	}
	err = web.AddVolumeWithOwner("/usr/share/nginx/html", "1Gi", 0)
	if err != nil {
		t.Fatalf("Error adding volume: %v", err)
	}
	err = web.Commit()
	if err != nil {
		t.Fatalf("Error committing instance: %v", err)
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

func TestDownloadFileFromRunningInstance(t *testing.T) {
	t.Parallel()
	// Setup

	target, err := knuu.NewInstance("target")
	require.NoError(t, err, "Error creating instance")

	require.NoError(t, target.SetImage("alpine:latest"), "Error setting image")
	require.NoError(t, target.SetArgs("tail", "-f", "/dev/null"), "Error setting args") // Keep the container running
	require.NoError(t, target.Commit(), "Error committing instance")
	require.NoError(t, target.Start(), "Error starting instance")

	t.Cleanup(func() {
		require.NoError(t, knuu.BatchDestroy(target))
	})

	// Test logic
	fileContent := "Hello World!"
	filePath := "/hello.txt"

	// Create a file in the target instance
	out, err := target.ExecuteCommand("echo", "-n", fileContent, ">", filePath)
	require.NoError(t, err, fmt.Sprintf("Error executing command: %v", out))

	gotContent, err := target.GetFileBytes(filePath)
	require.NoError(t, err, "Error getting file bytes")

	assert.Equal(t, fileContent, string(gotContent))
}

func TestMinio(t *testing.T) {
	t.Parallel()
	// Setup

	target, err := knuu.NewInstance("target")
	require.NoError(t, err, "Error creating instance")

	require.NoError(t, target.SetImage("alpine:latest"), "Error setting image")
	require.NoError(t, target.SetArgs("tail", "-f", "/dev/null"), "Error setting args") // Keep the container running
	require.NoError(t, target.Commit(), "Error committing instance")
	require.NoError(t, target.Start(), "Error starting instance")

	t.Cleanup(func() {
		require.NoError(t, knuu.BatchDestroy(target))
	})

	fileContent := "Hello World!"
	contentName := uuid.New().String()

	tmpFile, err := os.CreateTemp("", "hello.txt")
	require.NoError(t, err, "Error creating temporary file")
	defer os.Remove(tmpFile.Name())

	fmt.Printf("contentName: %v\n", contentName)

	_, err = tmpFile.WriteString(fileContent)
	require.NoError(t, err, "Error writing to temporary file")
	tmpFile.Close()

	// Write to file did not work, so need to open it again
	tmpFile, err = os.Open(tmpFile.Name())
	require.NoError(t, err, "Error opening temporary file")
	defer tmpFile.Close()

	fmt.Printf("tmpFile.Name(): %v\n", tmpFile.Name())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	err = knuu.PushFileToMinio(ctx, contentName, tmpFile)
	require.NoError(t, err, "Error pushing file to Minio")

	url, err := knuu.GetMinioURL(ctx, contentName)
	require.NoError(t, err, "Error getting Minio URL")

	resp, err := http.Get(url)
	require.NoError(t, err, "Error downloading the file from URL")
	defer resp.Body.Close()

	gotContent, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Error reading the response body")

	assert.Equal(t, fileContent, string(gotContent))
}
