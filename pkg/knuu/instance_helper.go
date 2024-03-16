package knuu

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
	imageName := fmt.Sprintf("ttl.sh/%s:24h", uuid.String())
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
		"knuu.sh/test-run-id":          identifier,
		"knuu.sh/test-started":         startTime,
		"knuu.sh/name":                 i.name,
		"knuu.sh/k8s-name":             i.k8sName,
		"knuu.sh/type":                 i.instanceType.String(),
	}
}

// deployService deploys the service for the instance
func (i *Instance) deployService() error {
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
	return k8s.DeleteService(k8s.Namespace(), i.k8sName)
}

// deployPod deploys the pod for the instance
func (i *Instance) deployPod() error {
	// Get labels for the pod
	labels := i.getLabels()

	// create a service account for the pod
	if err := k8s.CreateServiceAccount(k8s.Namespace(), i.k8sName, labels); err != nil {
		return fmt.Errorf("failed to create service account: %v", err)
	}

	// create a role and role binding for the pod if there are policy rules
	if len(i.policyRules) > 0 {
		if err := k8s.CreateRole(k8s.Namespace(), i.k8sName, labels, i.policyRules); err != nil {
			return fmt.Errorf("failed to create role: %v", err)
		}
		if err := k8s.CreateRoleBinding(k8s.Namespace(), i.k8sName, labels, i.k8sName, i.k8sName); err != nil {
			return fmt.Errorf("failed to create role binding: %v", err)
		}
	}

	statefulSetConfig := i.prepareStatefulSetConfig()

	// Deploy the statefulSet
	statefulSet, err := k8s.DeployStatefulSet(statefulSetConfig, true)
	if err != nil {
		return fmt.Errorf("failed to deploy pod: %v", err)
	}

	// Set the state of the instance to started
	i.kubernetesStatefulSet = statefulSet

	// Log the deployment of the pod
	logrus.Debugf("Started statefulSet '%s'", i.k8sName)
	logrus.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// destroyPod destroys the pod for the instance (no grace period)
// Skips if the pod is already destroyed
func (i *Instance) destroyPod() error {
	grace := int64(0)
	err := k8s.DeleteStatefulSetWithGracePeriod(k8s.Namespace(), i.k8sName, &grace)
	if err != nil {
		return fmt.Errorf("failed to delete pod: %v", err)
	}

	// Delete the service account for the pod
	if err := k8s.DeleteServiceAccount(k8s.Namespace(), i.k8sName); err != nil {
		return fmt.Errorf("failed to delete service account: %v", err)
	}
	// Delete the role and role binding for the pod if there are policy rules
	if len(i.policyRules) > 0 {
		if err := k8s.DeleteRole(k8s.Namespace(), i.k8sName); err != nil {
			return fmt.Errorf("failed to delete role: %v", err)
		}
		if err := k8s.DeleteRoleBinding(k8s.Namespace(), i.k8sName); err != nil {
			return fmt.Errorf("failed to delete role binding: %v", err)
		}
	}

	return nil
}

// deployService deploys the service for the instance
func (i *Instance) deployOrPatchService() error {
	if len(i.portsTCP) != 0 || len(i.portsUDP) != 0 {
		logrus.Debugf("Ports not empty, deploying service for instance '%s'", i.k8sName)
		svc, _ := k8s.GetService(k8s.Namespace(), i.k8sName)
		if svc == nil {
			err := i.deployService()
			if err != nil {
				return fmt.Errorf("error deploying service for instance '%s': %w", i.k8sName, err)
			}
		} else if svc != nil {
			err := i.patchService()
			if err != nil {
				return fmt.Errorf("error patching service for instance '%s': %w", i.k8sName, err)
			}
		}
	}
	return nil
}

// deployVolume deploys the volume for the instance
func (i *Instance) deployVolume() error {
	size := resource.Quantity{}
	for _, volume := range i.volumes {
		size.Add(resource.MustParse(volume.Size))
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

// deployFiles deploys the files for the instance
func (i *Instance) deployFiles() error {
	data := map[string]string{}

	n := 0

	for _, file := range i.files {
		// read out file content and assign to variable
		srcFile, err := os.Open(file.Source)
		if err != nil {
			return fmt.Errorf("failed to open file: %v", err)
		}
		fileContentBytes, err := io.ReadAll(srcFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}
		srcFile.Close()
		fileContent := string(fileContentBytes)

		keyName := fmt.Sprintf("%d", n)

		data[keyName] = fileContent

		n++
	}

	// create configmap
	if _, err := k8s.CreateConfigMap(k8s.Namespace(), i.k8sName, i.getLabels(), data); err != nil {
		return fmt.Errorf("failed to create configmap: %v", err)
	}

	return nil
}

// destroyFiles destroys the files for the instance
func (i *Instance) destroyFiles() error {
	if err := k8s.DeleteConfigMap(k8s.Namespace(), i.k8sName); err != nil {
		return fmt.Errorf("failed to delete configmap: %v", err)
	}
	return nil
}

// deployResources deploys the resources for the instance
func (i *Instance) deployResources() error {
	if len(i.portsTCP) != 0 || len(i.portsUDP) != 0 {
		if err := i.deployOrPatchService(); err != nil {
			return fmt.Errorf("failed to deploy or patch service: %v", err)
		}
	}
	if len(i.volumes) != 0 {
		if err := i.deployVolume(); err != nil {
			return fmt.Errorf("error deploying volume for instance '%s': %w", i.k8sName, err)
		}
	}
	if len(i.files) != 0 {
		if err := i.deployFiles(); err != nil {
			return fmt.Errorf("error deploying files for instance '%s': %w", i.k8sName, err)
		}
	}

	return nil
}

// destroyResources destroys the resources for the instance
func (i *Instance) destroyResources() error {
	if len(i.volumes) != 0 {
		err := i.destroyVolume()
		if err != nil {
			return fmt.Errorf("error destroying volume for instance '%s': %w", i.k8sName, err)
		}
	}
	if len(i.files) != 0 {
		err := i.destroyFiles()
		if err != nil {
			return fmt.Errorf("error destroying files for instance '%s': %w", i.k8sName, err)
		}
	}
	if i.kubernetesService != nil {
		err := i.destroyService()
		if err != nil {
			return fmt.Errorf("error destroying service for instance '%s': %w", i.k8sName, err)
		}
	}

	// disable network only for non-sidecar instances
	if !i.isSidecar {
		// enable network when network is disabled
		disableNetwork, err := i.NetworkIsDisabled()
		if err != nil {
			logrus.Debugf("error checking network status for instance")
			return fmt.Errorf("error checking network status for instance '%s': %w", i.k8sName, err)
		}
		if disableNetwork {
			err := i.EnableNetwork()
			if err != nil {
				logrus.Debugf("error enabling network for instance")
				return fmt.Errorf("error enabling network for instance '%s': %w", i.k8sName, err)
			}
		}
	}

	return nil
}

// cloneWithSuffix clones the instance with a suffix
func (i *Instance) cloneWithSuffix(suffix string) *Instance {
	clonedSidecars := make([]*Instance, len(i.sidecars))
	for i, sidecar := range i.sidecars {
		clonedSidecars[i] = sidecar.cloneWithSuffix(suffix)
	}

	// Deep copy of securityContext to ensure cloned instance has its own copy
	clonedSecurityContext := *i.securityContext

	clonedBitTwister := *i.BitTwister
	clonedBitTwister.SetClient(nil) // reset client to avoid reusing the same client

	return &Instance{
		name:                  i.name + suffix,
		k8sName:               i.k8sName + suffix,
		imageName:             i.imageName,
		state:                 i.state,
		instanceType:          i.instanceType,
		kubernetesService:     i.kubernetesService,
		builderFactory:        i.builderFactory,
		kubernetesStatefulSet: i.kubernetesStatefulSet,
		portsTCP:              i.portsTCP,
		portsUDP:              i.portsUDP,
		command:               i.command,
		args:                  i.args,
		env:                   i.env,
		volumes:               i.volumes,
		memoryRequest:         i.memoryRequest,
		memoryLimit:           i.memoryLimit,
		cpuRequest:            i.cpuRequest,
		policyRules:           i.policyRules,
		livenessProbe:         i.livenessProbe,
		readinessProbe:        i.readinessProbe,
		startupProbe:          i.startupProbe,
		isSidecar:             false,
		parentInstance:        nil,
		sidecars:              clonedSidecars,
		obsyConfig:            i.obsyConfig,
		securityContext:       &clonedSecurityContext,
		BitTwister:            &clonedBitTwister,
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

// getBuildDir returns the build directory for the instance
func (i *Instance) getBuildDir() string {
	return filepath.Join("/tmp", "knuu", i.k8sName)
}

// validateFileArgs validates the file arguments
func (i *Instance) validateFileArgs(src, dest, chown string) error {
	// check src
	if src == "" {
		return fmt.Errorf("src must be set")
	}
	// check dest
	if dest == "" {
		return fmt.Errorf("dest must be set")
	}
	// check chown
	if chown == "" {
		return fmt.Errorf("chown must be set")
	}
	// validate chown format
	if !strings.Contains(chown, ":") || len(strings.Split(chown, ":")) != 2 {
		return fmt.Errorf("chown must be in format 'user:group'")
	}

	return nil
}

// addFileToBuilder adds a file to the builder
func (i *Instance) addFileToBuilder(src, dest, chown string) error {
	// dest is the same as src here, as we copy the file to the build dir with the subfolder structure of dest
	err := i.builderFactory.AddToBuilder(dest, dest, chown)
	if err != nil {
		return fmt.Errorf("error adding file '%s' to instance '%s': %w", dest, i.name, err)
	}
	return nil
}

// prepareSecurityContext creates a v1.SecurityContext from a given SecurityContext.
func prepareSecurityContext(config *SecurityContext) *v1.SecurityContext {
	securityContext := &v1.SecurityContext{}

	if config != nil {
		if config.privileged {
			securityContext.Privileged = &config.privileged
		}
		if len(config.capabilitiesAdd) > 0 {
			capabilities := make([]v1.Capability, len(config.capabilitiesAdd))
			for i, cap := range config.capabilitiesAdd {
				capabilities[i] = v1.Capability(cap)
			}
			securityContext.Capabilities = &v1.Capabilities{
				Add: capabilities,
			}
		}
	}

	return securityContext
}

// prepareConfig prepares the config for the instance
func (i *Instance) prepareStatefulSetConfig() k8s.StatefulSetConfig {

	// Generate the container configuration
	containerConfig := k8s.ContainerConfig{
		Name:            i.k8sName,
		Image:           i.imageName,
		Command:         i.command,
		Args:            i.args,
		Env:             i.env,
		Volumes:         i.volumes,
		MemoryRequest:   i.memoryRequest,
		MemoryLimit:     i.memoryLimit,
		CPURequest:      i.cpuRequest,
		LivenessProbe:   i.livenessProbe,
		ReadinessProbe:  i.readinessProbe,
		StartupProbe:    i.startupProbe,
		Files:           i.files,
		SecurityContext: prepareSecurityContext(i.securityContext),
	}
	// Generate the sidecar configurations
	sidecarConfigs := make([]k8s.ContainerConfig, 0)
	for _, sidecar := range i.sidecars {
		sidecarConfigs = append(sidecarConfigs, k8s.ContainerConfig{
			Name:            sidecar.k8sName,
			Image:           sidecar.imageName,
			Command:         sidecar.command,
			Args:            sidecar.args,
			Env:             sidecar.env,
			Volumes:         sidecar.volumes,
			MemoryRequest:   sidecar.memoryRequest,
			MemoryLimit:     sidecar.memoryLimit,
			CPURequest:      sidecar.cpuRequest,
			LivenessProbe:   sidecar.livenessProbe,
			ReadinessProbe:  sidecar.readinessProbe,
			StartupProbe:    sidecar.startupProbe,
			Files:           sidecar.files,
			SecurityContext: prepareSecurityContext(sidecar.securityContext),
		})
	}
	// Generate the pod configuration
	podConfig := k8s.PodConfig{
		Namespace:          k8s.Namespace(),
		Name:               i.k8sName,
		Labels:             i.getLabels(),
		ServiceAccountName: i.k8sName,
		FsGroup:            i.fsGroup,
		ContainerConfig:    containerConfig,
		SidecarConfigs:     sidecarConfigs,
	}
	// Generate the statefulset configuration
	statefulSetConfig := k8s.StatefulSetConfig{
		Namespace: k8s.Namespace(),
		Name:      i.k8sName,
		Labels:    i.getLabels(),
		Replicas:  1,
		PodConfig: podConfig,
	}

	return statefulSetConfig
}

// setImageWithGracePeriod sets the image of the instance with a grace period
func (i *Instance) setImageWithGracePeriod(imageName string, gracePeriod *int64) error {
	i.imageName = imageName

	statefulSetConfig := i.prepareStatefulSetConfig()

	// Replace the pod with a new one, using the given image
	_, err := k8s.ReplaceStatefulSetWithGracePeriod(statefulSetConfig, gracePeriod)
	if err != nil {
		return fmt.Errorf("error replacing pod: %s", err.Error())
	}
	if err := i.WaitInstanceIsRunning(); err != nil {
		return fmt.Errorf("error waiting for instance to be running: %w", err)
	}

	return nil
}

// applyFunctionToInstances applies a function to all instances
func applyFunctionToInstances(instances []*Instance, function func(sidecar Instance) error) error {
	for _, i := range instances {
		if err := function(*i); err != nil {
			return fmt.Errorf("error applying function to instance '%s': %w", i.k8sName, err)
		}
	}
	return nil
}

func setStateForSidecars(sidecars []*Instance, state InstanceState) {
	// We don't handle errors here, as the function can't return an error
	err := applyFunctionToInstances(sidecars, func(sidecar Instance) error {
		sidecar.state = state
		return nil
	})
	if err != nil {
		return
	}
}

// isObservabilityEnabled returns true if observability is enabled
func (i *Instance) isObservabilityEnabled() bool {
	return i.obsyConfig.otlpPort != 0 || i.obsyConfig.prometheusPort != 0 || i.obsyConfig.jaegerGrpcPort != 0 || i.obsyConfig.jaegerThriftCompactPort != 0 || i.obsyConfig.jaegerThriftHttpPort != 0
}

func (i *Instance) validateStateForObsy(endpoint string) error {
	if !i.IsInState(Preparing, Committed) {
		return fmt.Errorf("setting %s is only allowed in state 'Preparing' or 'Committed'. Current state is '%s'", endpoint, i.state.String())
	}
	return nil
}

func (i *Instance) addOtelCollectorSidecar() error {
	otelSidecar, err := i.createOtelCollectorInstance()
	if err != nil {
		return fmt.Errorf("error creating otel collector instance '%s': %w", i.k8sName, err)
	}
	if err := i.AddSidecar(otelSidecar); err != nil {
		return fmt.Errorf("error adding otel collector sidecar to instance '%s': %w", i.k8sName, err)
	}
	return nil
}

func (i *Instance) createBitTwisterInstance() (*Instance, error) {
	bt, err := NewInstance("bit-twister")
	if err != nil {
		return nil, fmt.Errorf("error creating bit-twister instance: %w", err)
	}

	if err := bt.SetImage(i.BitTwister.Image()); err != nil {
		return nil, fmt.Errorf("error setting image for bit-twister instance: %w", err)
	}

	// We need to add the port here so the instance will get an IP
	if err := i.AddPortTCP(i.BitTwister.Port()); err != nil {
		return nil, fmt.Errorf("error adding BitTwister port: %w", err)
	}
	ip, err := i.GetIP()
	if err != nil {
		return nil, fmt.Errorf("error getting IP of instance '%s': %w", i.name, err)
	}
	logrus.Debugf("IP of instance '%s' is '%s'", i.name, ip)

	i.BitTwister.SetNewClientByIPAddr("http://" + ip)

	if err := bt.Commit(); err != nil {
		return nil, fmt.Errorf("error committing bit-twister instance: %w", err)
	}

	if err := bt.SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", i.BitTwister.Port())); err != nil {
		return nil, fmt.Errorf("error setting environment variable for bit-twister instance: %w", err)
	}

	return bt, nil
}

func (i *Instance) addBitTwisterSidecar() error {
	networkConfigSidecar, err := i.createBitTwisterInstance()
	if err != nil {
		return fmt.Errorf("error creating bit-twister instance '%s': %w", i.k8sName, err)
	}

	if err := networkConfigSidecar.SetPrivileged(true); err != nil {
		return fmt.Errorf("error setting privileged for bit-twister instance '%s': %w", i.k8sName, err)
	}

	if err := networkConfigSidecar.AddCapability("all"); err != nil {
		return fmt.Errorf("error adding capability for bit-twister instance '%s': %w", i.k8sName, err)
	}

	if err := networkConfigSidecar.SetUser("0"); err != nil {
		return fmt.Errorf("error setting user root for bit-twister instance '%s': %w", i.k8sName, err)
	}

	if err := i.AddSidecar(networkConfigSidecar); err != nil {
		return fmt.Errorf("error adding bit-twister sidecar to instance '%s': %w", i.k8sName, err)
	}
	return nil
}
