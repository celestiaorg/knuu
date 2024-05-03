package basic

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

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
		err := assertCleanupInstance(t, instance)
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

	assert.Equal(t, "<!DOCTYPE html>\n<html>\n<head>\n<title>Welcome to nginx!</title>\n<style>\nhtml { color-scheme: light dark; }\nbody { width: 35em; margin: 0 auto;\nfont-family: Tahoma, Verdana, Arial, sans-serif; }\n</style>\n</head>\n<body>\n<h1>Welcome to nginx!</h1>\n<p>If you see this page, the nginx web server is successfully installed and\nworking. Further configuration is required.</p>\n\n<p>For online documentation and support please refer to\n<a href=\"http://nginx.org/\">nginx.org</a>.<br/>\nCommercial support is available at\n<a href=\"http://nginx.com/\">nginx.com</a>.</p>\n\n<p><em>Thank you for using nginx.</em></p>\n</body>\n</html>\n", wget)
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
		err := assertCleanupInstance(t, instance)
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

	assert.Equal(t, "<!DOCTYPE html>\n<html>\n<head>\n<title>Welcome to nginx!</title>\n<style>\nhtml { color-scheme: light dark; }\nbody { width: 35em; margin: 0 auto;\nfont-family: Tahoma, Verdana, Arial, sans-serif; }\n</style>\n</head>\n<body>\n<h1>Welcome to nginx!</h1>\n<p>If you see this page, the nginx web server is successfully installed and\nworking. Further configuration is required.</p>\n\n<p>For online documentation and support please refer to\n<a href=\"http://nginx.org/\">nginx.org</a>.<br/>\nCommercial support is available at\n<a href=\"http://nginx.com/\">nginx.com</a>.</p>\n\n<p><em>Thank you for using nginx.</em></p>\n</body>\n</html>\n", wget)
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
		_, err = instance.ExecuteCommand("mkdir", "-p", "/usr/share/nginx/html")
		if err != nil {
			t.Fatalf("Error executing command for '%v': %v", instanceName, err)
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
			err = instance.AddFile("resources/file_cm_to_folder/test_1", "/usr/share/nginx/html/index.html", "0:0")
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
			err = instance.AddFile("resources/file_cm_to_folder/test_1", "/usr/share/nginx/html/index.html", "0:0")
			if err != nil {
				t.Fatalf("Error adding file to '%v': %v", instanceName, err)
			}
		}(i, instance)
	}
	wgFolders.Wait()

	t.Cleanup(func() {
		// Cleanup
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
			err = instance.AddFile("resources/file_cm_to_folder/test_1", "/usr/share/nginx/html/index.html", "0:0")
			if err != nil {
				t.Fatalf("Error adding file to '%v': %v", instanceName, err)
			}
			err = instance.AddFile("resources/file_cm_to_folder/test_2", "/usr/share/nginx/html/index-2.html", "0:0")
			if err != nil {
				t.Fatalf("Error adding file to '%v': %v", instanceName, err)
			}
		}(i, instance)
	}
	wgFolders.Wait()

	t.Cleanup(func() {
		// Cleanup
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

		wgetIndex, err := executor.ExecuteCommand("wget", "-q", "-O", "-", webIP)
		if err != nil {
			t.Fatalf("Error executing command: %v", err)
		}

		webIP2 := webIP + "/index-2.html"
		wgetIndex2, err := executor.ExecuteCommand("wget", "-q", "-O", "-", webIP2)
		if err != nil {
			t.Fatalf("Error executing command: %v", err)
		}

		assert.Equal(t, "hello from 1", wgetIndex)
		assert.Equal(t, "hello from 2", wgetIndex2)
	}
}
