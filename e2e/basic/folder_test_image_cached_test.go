package basic

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

func TestFolderCached(t *testing.T) {
	t.Parallel()

	// Setup
	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	// Test logic
	const numberOfInstances = 10
	instances := make([]*knuu.Instance, numberOfInstances)

	for i := 0; i < numberOfInstances; i++ {
		instanceName := fmt.Sprintf("web%d", i+1)
		instance, err := knuu.NewInstance(instanceName)
		if err != nil {
			t.Fatalf("Error creating instance '%v': %v", instanceName, err)
		}
		err = instance.SetImage("docker.io/nginx:latest")
		if err != nil {
			t.Fatalf("Error setting image for '%v': %v", instanceName, err)
		}
		instance.AddPortTCP(80)
		_, err = instance.ExecuteCommand("mkdir", "-p", "/usr/share/nginx/html")
		if err != nil {
			t.Fatalf("Error executing command for '%v': %v", instanceName, err)
		}
		err = instance.AddVolumeWithOwner("/usr/share/nginx/html", "1Gi", 0)
		if err != nil {
			t.Fatalf("Error adding volume: %v", err)
		}
		err = instance.Commit()
		if err != nil {
			t.Fatalf("Error committing instance '%v': %v", instanceName, err)
		}

		instances[i] = instance
	}

	var wgFolders sync.WaitGroup
	for i, instance := range instances {
		wgFolders.Add(1)
		go func(i int, instance *knuu.Instance) {
			defer wgFolders.Done()
			instanceName := fmt.Sprintf("web%d", i+1)
			// adding the folder after the Commit, it will help us to use a cached image.
			err := instance.AddFolder("resources/html", "/usr/share/nginx/html", "0:0")
			if err != nil {
				t.Fatalf("Error adding file to '%v': %v", instanceName, err)
			}
		}(i, instance)
	}
	wgFolders.Wait()

	// Cleanup
	t.Cleanup(func() {
		err := assertCleanupInstances(t, executor, instances)
		if err != nil {
			t.Fatalf("Error cleaning up: %v", err)
		}
	})

	// Test logic
	for _, instance := range instances {
		err = instance.StartAsync()
		if err != nil {
			t.Fatalf("Error waiting for instance to be running: %v", err)
		}
	}

	for _, instance := range instances {
		webIP, err := instance.GetIP()
		if err != nil {
			t.Fatalf("Error getting IP: %v", err)
		}

		err = instance.WaitInstanceIsRunning()
		if err != nil {
			t.Fatalf("Error waiting for instance to be running: %v", err)
		}

		wget, err := executor.ExecuteCommand("wget", "-q", "-O", "-", webIP)
		if err != nil {
			t.Fatalf("Error executing command: %v", err)
		}

		assert.Contains(t, "Hello World!", wget)
	}
}
