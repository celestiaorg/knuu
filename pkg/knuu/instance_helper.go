package knuu

import (
	"fmt"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"
	"net"
)

// getImageRegistry returns the name of the temporary image registry
func (i *Instance) getImageRegistry() (string, error) {
	if i.imageName != "" {
		return i.imageName, nil
	}
	// If not already set, generate a random name using ttl.sh
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("error generating UUID: %w", err)
	}
	imageName := fmt.Sprintf("ttl.sh/%s:1h", uuid.String())
	return imageName, nil
}

// validatePort validates the port
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number '%d' is out of range", port)
	}
	return nil
}

// isTCPPortRegistered returns true if the given port is registered
// with the instance, and false otherwise
func (i *Instance) isTCPPortRegistered(port int) bool {
	for _, p := range i.portsTCP {
		if p == port {
			return true
		}
	}
	return false
}

// isUDPPortRegistered returns true if the given port is registered
// with the instance, and false otherwise
func (i *Instance) isUDPPortRegistered(port int) bool {
	for _, p := range i.portsUDP {
		if p == port {
			return true
		}
	}
	return false
}

// getLabels returns the labels for the instance
func (i *Instance) getLabels() map[string]string {
	return map[string]string{
		"app":                          i.k8sName,
		"k8s.kubernetes.io/managed-by": "knuu",
		"test-run-id":                  identifier,
		"test-started":                 startTime,
		"name":                         i.name,
		"k8s-name":                     i.k8sName,
	}
}

// deployService deploys the service for the instance
func (i *Instance) deployService() error {
	svc, _ := k8s.GetService(k8s.Namespace(), i.k8sName)
	if svc != nil {
		// Service already exists, so we patch it
		err := i.patchService()
		if err != nil {
			return fmt.Errorf("error patching service '%s': %w", i.k8sName, err)
		}
	}

	labels := i.getLabels()
	selectorMap := i.getLabels()
	service, err := k8s.DeployService(k8s.Namespace(), i.k8sName, labels, selectorMap, i.portsTCP, i.portsUDP)
	if err != nil {
		return fmt.Errorf("error deploying service '%s': %w", i.k8sName, err)
	}
	i.kubernetesService = service
	logrus.Debugf("Started service '%s'", i.k8sName)
	return nil
}

// patchService patches the service for the instance
func (i *Instance) patchService() error {
	if i.kubernetesService == nil {
		svc, err := k8s.GetService(k8s.Namespace(), i.k8sName)
		if err != nil {
			return fmt.Errorf("error getting service '%s': %w", i.k8sName, err)
		}
		i.kubernetesService = svc
	}
	err := k8s.PatchService(k8s.Namespace(), i.k8sName, i.kubernetesService.ObjectMeta.Labels, i.kubernetesService.Spec.Selector, i.portsTCP, i.portsUDP)
	if err != nil {
		return fmt.Errorf("error patching service '%s': %w", i.k8sName, err)
	}
	logrus.Debugf("Patched service '%s'", i.k8sName)
	return nil
}

// destroyService destroys the service for the instance
func (i *Instance) destroyService() error {
	k8s.DeleteService(k8s.Namespace(), i.k8sName)

	return nil
}

// deployPod deploys the pod for the instance
func (i *Instance) deployPod() error {
	// Get labels for the pod
	labels := i.getLabels()

	imageName, err := i.getImageRegistry()
	if err != nil {
		return fmt.Errorf("failed to get image name: %v", err)
	}

	// Generate the pod configuration
	podConfig := k8s.PodConfig{
		Namespace:     k8s.Namespace(),
		Name:          i.k8sName,
		Labels:        labels,
		Image:         imageName,
		Command:       i.command,
		Args:          i.args,
		Env:           i.env,
		Volumes:       i.volumes,
		MemoryRequest: i.memoryRequest,
		MemoryLimit:   i.memoryLimit,
		CPURequest:    i.cpuRequest,
	}

	// Deploy the pod
	pod, err := k8s.DeployPod(podConfig, true)
	if err != nil {
		return fmt.Errorf("failed to deploy pod: %v", err)
	}

	// Set the state of the instance to started
	i.kubernetesPod = pod

	// Log the deployment of the pod
	logrus.Debugf("Started pod '%s'", i.k8sName)
	logrus.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// destroyPod destroys the pod for the instance
// Skips if the pod is already destroyed
func (i *Instance) destroyPod() error {
	err := k8s.DeletePod(k8s.Namespace(), i.k8sName)
	if err != nil {
		return fmt.Errorf("failed to delete pod: %v", err)
	}

	return nil
}

// deployVolume deploys the volume for the instance
func (i *Instance) deployVolume() error {
	size := resource.Quantity{}
	for _, volumeSize := range i.volumes {
		size.Add(resource.MustParse(volumeSize))
	}
	k8s.DeployPersistentVolumeClaim(k8s.Namespace(), i.k8sName, i.getLabels(), size)
	logrus.Debugf("Deployed persistent volume '%s'", i.k8sName)

	return nil
}

// destroyVolume destroys the volume for the instance
func (i *Instance) destroyVolume() error {
	k8s.DeletePersistentVolumeClaim(k8s.Namespace(), i.k8sName)
	logrus.Debugf("Destroyed persistent volume '%s'", i.k8sName)

	return nil
}

// cloneWithSuffix clones the instance with a suffix
func (i *Instance) cloneWithSuffix(suffix string) *Instance {
	return &Instance{
		name:              i.name + suffix,
		k8sName:           i.k8sName + suffix,
		imageName:         i.imageName,
		state:             i.state,
		kubernetesService: i.kubernetesService,
		builderFactory:    i.builderFactory,
		kubernetesPod:     i.kubernetesPod,
		portsTCP:          i.portsTCP,
		portsUDP:          i.portsUDP,
		files:             i.files,
		command:           i.command,
		args:              i.args,
		env:               i.env,
		volumes:           i.volumes,
		memoryRequest:     i.memoryRequest,
		memoryLimit:       i.memoryLimit,
		cpuRequest:        i.cpuRequest,
	}
}

func generateK8sName(name string) (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("error generating UUID: %w", err)
	}
	return fmt.Sprintf("%s-%s", name, uuid.String()[:8]), nil
}

// getFreePort returns a free port
func getFreePortTCP() (int, error) {
	// Get a random port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("error getting free port: %w", err)
	}
	defer listener.Close()

	// Get the port from the listener
	port := listener.Addr().(*net.TCPAddr).Port

	return port, nil
}
