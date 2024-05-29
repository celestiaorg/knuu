package system

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/knuu/e2e"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

// TestOneVolumeNoFiles tests the scenario where we have one volume and no files.
// the initContainer command that it generates looks like:
// no initContainer command, as there is no volumes, nor files.
func TestNoVolumesNoFiles(t *testing.T) {
	t.Parallel()
	// Setup

	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	instanceName := fmt.Sprintf("web-1")
	instance, err := knuu.NewInstance(instanceName)
	if err != nil {
		t.Fatalf("Error creating instance '%v': %v", instanceName, err)
	}
	err = instance.SetImage("docker.io/nginx:latest")
	if err != nil {
		t.Fatalf("Error setting image for '%v': %v", instanceName, err)
	}
	err = instance.AddPortTCP(80)
	if err != nil {
		t.Fatalf("Error adding port for '%v': %v", instanceName, err)
	}
	err = instance.Commit()
	if err != nil {
		t.Fatalf("Error committing instance '%v': %v", instanceName, err)
	}

	// Cleanup
	t.Cleanup(func() {
		err := e2e.AssertCleanupInstance(t, instance)
		if err != nil {
			t.Fatalf("Error cleaning up: %v", err)
		}
	})

	// Test logic
	err = instance.StartAsync()
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}

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

	assert.Contains(t, wget, "Welcome to nginx!")
}

// TestOneVolumeNoFiles tests the scenario where we have one volume and no files.
// the initContainer command that it generates looks like:
// mkdir -p /knuu && if [ -d /opt/vol1 ] && [ \"$(ls -A /opt/vol1)\" ]; then cp -r /opt/vol1/* /knuu//opt/vol1 && chown -R 0:0 /knuu/* ;fi
func TestOneVolumeNoFiles(t *testing.T) {
	t.Parallel()
	// Setup

	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	instanceName := fmt.Sprintf("web-1")
	instance, err := knuu.NewInstance(instanceName)
	if err != nil {
		t.Fatalf("Error creating instance '%v': %v", instanceName, err)
	}
	err = instance.SetImage("docker.io/nginx:latest")
	if err != nil {
		t.Fatalf("Error setting image for '%v': %v", instanceName, err)
	}
	err = instance.AddPortTCP(80)
	if err != nil {
		t.Fatalf("Error adding port for '%v': %v", instanceName, err)
	}
	err = instance.AddVolumeWithOwner("/opt/vol1", "1Gi", 0)
	if err != nil {
		t.Fatalf("Error adding volume: %v", err)
	}
	err = instance.Commit()
	if err != nil {
		t.Fatalf("Error committing instance '%v': %v", instanceName, err)
	}

	// Cleanup
	t.Cleanup(func() {
		err := e2e.AssertCleanupInstance(t, instance)
		if err != nil {
			t.Fatalf("Error cleaning up: %v", err)
		}
	})

	// Test logic
	err = instance.StartAsync()
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}

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

	assert.Contains(t, wget, "Welcome to nginx!")
}

// TestNoVolumesOneFile tests the scenario where we have no volumes and one file.
// the initContainer command that it generates looks like:
// no initContainer command, as we do not have volumes.
func TestNoVolumesOneFile(t *testing.T) {
	t.Parallel()
	// Setup
	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	const numberOfInstances = 2
	instances := make([]*knuu.Instance, numberOfInstances)

	for i := 0; i < numberOfInstances; i++ {
		instanceName := fmt.Sprintf("web%d", i+1)
		instances[i] = e2e.AssertCreateInstanceNginxWithVolumeOwner(t, instanceName)
	}

	var wgFolders sync.WaitGroup
	errorChannel := make(chan error, len(instances))

	for i, instance := range instances {
		wgFolders.Add(1)
		go func(i int, instance *knuu.Instance) {
			defer wgFolders.Done()
			instanceName := fmt.Sprintf("web%d", i+1)
			// adding the folder after the Commit, it will help us to use a cached image.
			err = instance.AddFile("resources/file_cm_to_folder/test_1", "/usr/share/nginx/html/index.html", "0:0")
			if err != nil {
				errorChannel <- fmt.Errorf("Error adding file to '%v': %v", instanceName, err)
				return
			}
			errorChannel <- nil
		}(i, instance)
	}
	wgFolders.Wait()
	close(errorChannel)

	for err := range errorChannel {
		if err != nil {
			t.Fatalf("%v", err)
		}
	}

	// Cleanup
	t.Cleanup(func() {
		err := e2e.AssertCleanupInstances(t, executor, instances)
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
		wget = strings.TrimSpace(wget)

		assert.Equal(t, "hello from 1", wget)
	}
}

// TestOneVolumeOneFile tests the scenario where we have one volume and one file.
// the initContainer command that it generates looks like:
// mkdir -p /knuu && mkdir -p /knuu/usr/share/nginx/html && chmod -R 777 /knuu//usr/share/nginx/html && if [ -d /usr/share/nginx/html ] && [ \"$(ls -A /usr/share/nginx/html)\" ]; then cp -r /usr/share/nginx/html/* /knuu//usr/share/nginx/html && chown -R 0:0 /knuu/* ;fi
func TestOneVolumeOneFile(t *testing.T) {
	t.Parallel()
	// Setup

	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	const numberOfInstances = 2
	instances := make([]*knuu.Instance, numberOfInstances)

	for i := 0; i < numberOfInstances; i++ {
		instanceName := fmt.Sprintf("web%d", i+1)
		instances[i] = e2e.AssertCreateInstanceNginxWithVolumeOwner(t, instanceName)
	}

	var wgFolders sync.WaitGroup
	errorChannel := make(chan error, len(instances))

	for i, instance := range instances {
		wgFolders.Add(1)
		go func(i int, instance *knuu.Instance) {
			defer wgFolders.Done()
			instanceName := fmt.Sprintf("web%d", i+1)
			// adding the folder after the Commit, it will help us to use a cached image.
			err := instance.AddFile("resources/file_cm_to_folder/test_1", "/usr/share/nginx/html/index.html", "0:0")
			if err != nil {
				errorChannel <- fmt.Errorf("Error adding file to '%v': %v", instanceName, err)
				return
			}
			errorChannel <- nil
		}(i, instance)
	}
	wgFolders.Wait()
	close(errorChannel)

	for err := range errorChannel {
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		// Cleanup
		err := e2e.AssertCleanupInstances(t, executor, instances)
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
		wget = strings.TrimSpace(wget)

		assert.Equal(t, "hello from 1", wget)
	}
}

// TestOneVolumeOneFile tests the scenario where we have one volume and one file.
// the initContainer command that it generates looks like:
// mkdir -p /knuu && mkdir -p /knuu/usr/share/nginx/html && chmod -R 777 /knuu//usr/share/nginx/html && if [ -d /usr/share/nginx/html ] && [ \"$(ls -A /usr/share/nginx/html)\" ]; then cp -r /usr/share/nginx/html/* /knuu//usr/share/nginx/html && chown -R 0:0 /knuu/* ;fi
func TestOneVolumeTwoFiles(t *testing.T) {
	t.Parallel()
	// Setup

	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	const numberOfInstances = 2
	instances := make([]*knuu.Instance, numberOfInstances)

	for i := 0; i < numberOfInstances; i++ {
		instanceName := fmt.Sprintf("web%d", i+1)
		instances[i] = e2e.AssertCreateInstanceNginxWithVolumeOwner(t, instanceName)
	}

	var wgFolders sync.WaitGroup
	errorChannel := make(chan error, len(instances)*2) // Allocate space for potential errors from each file addition in each instance

	for i, instance := range instances {
		wgFolders.Add(1)
		go func(i int, instance *knuu.Instance) {
			defer wgFolders.Done()
			instanceName := fmt.Sprintf("web%d", i+1)
			// adding the folder after the Commit, it will help us to use a cached image.
			if err := instance.AddFile("resources/file_cm_to_folder/test_1", "/usr/share/nginx/html/index.html", "0:0"); err != nil {
				errorChannel <- fmt.Errorf("Error adding file test_1 to '%v': %w", instanceName, err)
				return
			}
			if err := instance.AddFile("resources/file_cm_to_folder/test_2", "/usr/share/nginx/html/index-2.html", "0:0"); err != nil {
				errorChannel <- fmt.Errorf("Error adding file test_2 to '%v': %w", instanceName, err)
				return
			}
		}(i, instance)
	}
	wgFolders.Wait()
	close(errorChannel)

	// Handle errors from the error channel
	for err := range errorChannel {
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		// Cleanup
		err := e2e.AssertCleanupInstances(t, executor, instances)
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

		wgetIndex, err := executor.ExecuteCommand("wget", "-q", "-O", "-", webIP)
		if err != nil {
			t.Fatalf("Error executing command: %v", err)
		}
		wgetIndex = strings.TrimSpace(wgetIndex)

		webIP2 := webIP + "/index-2.html"
		wgetIndex2, err := executor.ExecuteCommand("wget", "-q", "-O", "-", webIP2)
		if err != nil {
			t.Fatalf("Error executing command: %v", err)
		}
		wgetIndex2 = strings.TrimSpace(wgetIndex2)

		assert.Equal(t, "hello from 1", wgetIndex)
		assert.Equal(t, "hello from 2", wgetIndex2)
	}
}
