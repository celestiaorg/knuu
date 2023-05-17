package knuu

import (
	"fmt"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"
)

// getTempImageRegistry returns the name of the temporary image registry
func (i *Instance) getTempImageRegistry() string {
	return fmt.Sprintf("ttl.sh/%s:1h", i.uuid.String())
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
	return map[string]string{"app": i.k8sName}
}

// deployService deploys the service for the instance
func (i *Instance) deployService() error {
	svc, _ := k8s.GetService(k8s.Namespace, i.k8sName)
	if svc != nil {
		// Service already exists, so we patch it
		err := i.patchService()
		if err != nil {
			return fmt.Errorf("error patching service '%s': %w", i.k8sName, err)
		}
	}

	labels := i.getLabels()
	selectorMap := i.getLabels()
	service, err := k8s.DeployService(k8s.Namespace, i.k8sName, labels, selectorMap, i.portsTCP, i.portsUDP)
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
		svc, err := k8s.GetService(k8s.Namespace, i.k8sName)
		if err != nil {
			return fmt.Errorf("error getting service '%s': %w", i.k8sName, err)
		}
		i.kubernetesService = svc
	}
	err := k8s.PatchService(k8s.Namespace, i.k8sName, i.kubernetesService.ObjectMeta.Labels, i.kubernetesService.Spec.Selector, i.portsTCP, i.portsUDP)
	if err != nil {
		return fmt.Errorf("error patching service '%s': %w", i.k8sName, err)
	}
	logrus.Debugf("Patched service '%s'", i.k8sName)
	return nil
}

// destroyService destroys the service for the instance
func (i *Instance) destroyService() error {
	k8s.DeleteService(k8s.Namespace, i.k8sName)

	return nil
}

// deployPod deploys the pod for the instance
func (i *Instance) deployPod() error {
	// Get labels for the pod
	labels := i.getLabels()

	// Generate the pod configuration
	podConfig := k8s.PodConfig{
		Namespace: k8s.Namespace,
		Name:      i.k8sName,
		Labels:    labels,
		Image:     i.getTempImageRegistry(), // Get temporary image registry for the pod
		Command:   i.command,
		Args:      i.args,
		Env:       i.env,
		Volumes:   i.volumes,
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
func (i *Instance) destroyPod() error {
	k8s.DeletePod(k8s.Namespace, i.k8sName)

	return nil
}

// deployVolume deploys the volume for the instance
func (i *Instance) deployVolume() error {
	size := resource.Quantity{}
	for _, volumeSize := range i.volumes {
		size.Add(resource.MustParse(volumeSize))
	}
	k8s.DeployPersistentVolumeClaim(k8s.Namespace, i.k8sName, i.getLabels(), size, []string{"ReadWriteOnce"})
	logrus.Debugf("Deployed persistent volume '%s'", i.k8sName)

	return nil
}

// destroyVolume destroys the volume for the instance
func (i *Instance) destroyVolume() error {
	k8s.DeletePersistentVolumeClaim(k8s.Namespace, i.k8sName)
	logrus.Debugf("Destroyed persistent volume '%s'", i.k8sName)

	return nil
}

// cloneWithSuffix clones the instance with a suffix
func (i *Instance) cloneWithSuffix(suffix string) *Instance {
	return &Instance{
		uuid:              i.uuid,
		name:              i.name + suffix,
		k8sName:           i.k8sName + suffix,
		state:             i.state,
		kubernetesService: i.kubernetesService,
		imageBuilder:      i.imageBuilder,
		buildStore:        i.buildStore,
		buildContext:      i.buildContext,
		kubernetesPod:     i.kubernetesPod,
		portsTCP:          i.portsTCP,
		portsUDP:          i.portsUDP,
		files:             i.files,
		command:           i.command,
		args:              i.args,
		env:               i.env,
		volumes:           i.volumes,
	}
}
