package basic

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

func TestBasic(t *testing.T) {
	t.Parallel()
	// Setup

	instance, err := knuu.NewInstance("alpine")
	if err != nil {
		t.Fatalf("Error creating instance '%v':", err)
	}
	err = instance.SetImage("docker.io/alpine:latest")
	if err != nil {
		t.Fatalf("Error setting image: %v", err)
	}
	err = instance.SetCommand("sleep", "infinity")
	if err != nil {
		t.Fatalf("Error setting command: %v", err)
	}
	err = instance.Commit()
	if err != nil {
		t.Fatalf("Error committing instance: %v", err)
	}

	t.Cleanup(func() {
		// Cleanup
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		err = instance.Destroy()
		if err != nil {
			t.Fatalf("Error destroying instance: %v", err)
		}
	})

	// Test logic

	err = instance.Start()
	if err != nil {
		t.Fatalf("Error starting instance: %v", err)
	}
	err = instance.WaitInstanceIsRunning()
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}
	wget, err := instance.ExecuteCommand("echo", "Hello World!")
	if err != nil {
		t.Fatalf("Error executing command '%v':", err)
	}

	assert.Equal(t, wget, "Hello World!\n")
}
