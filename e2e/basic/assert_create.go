package basic

import (
	"testing"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	// nginxImage is the image used to create the instances.
	nginxImage = "docker.io/nginx:latest"
	// nginxVolume is the volume used to create the instances.
	nginxVolume = "1Gi"
	// nginxVolumeOwner is the owner of the volume used to create the instances.
	nginxVolumeOwner = 0
	// nginxPort is the port used to create the instances.
	nginxPort = 80
	// nginxPath is the path used to create the instances.
	nginxPath = "/usr/share/nginx/html"
)

// assertCreateInstanceNginxWithVolumeOwner creates and configures an instance with common settings used across tests.
func assertCreateInstanceNginxWithVolumeOwner(t *testing.T, instanceName string) *knuu.Instance {
	instance, err := knuu.NewInstance(instanceName)
	if err != nil {
		t.Fatalf("Error creating instance '%v': %v", instanceName, err)
	}
	err = instance.SetImage(nginxImage)
	if err != nil {
		t.Fatalf("Error setting image for '%v': %v", instanceName, err)
	}
	err = instance.AddPortTCP(nginxPort)
	if err != nil {
		t.Fatalf("Error adding port for '%v': %v", instanceName, err)
	}
	_, err = instance.ExecuteCommand("mkdir", "-p", nginxPath)
	if err != nil {
		t.Fatalf("Error executing command for '%v': %v", instanceName, err)
	}
	err = instance.AddVolumeWithOwner(nginxPath, nginxVolume, nginxVolumeOwner)
	if err != nil {
		t.Fatalf("Error adding volume: %v", err)
	}
	err = instance.Commit()
	if err != nil {
		t.Fatalf("Error committing instance '%v': %v", instanceName, err)
	}
	return instance
}
