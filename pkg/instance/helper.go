package instance

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

const (
	labelAppKey         = "app"
	labelManagedByKey   = "k8s.kubernetes.io/managed-by"
	labelScopeKey       = "knuu.sh/scope"
	labelTestStartedKey = "knuu.sh/test-started"
	labelNameKey        = "knuu.sh/name"
	labelK8sNameKey     = "knuu.sh/k8s-name"
	labelTypeKey        = "knuu.sh/type"
	labelKnuuValue      = "knuu"

	buildDirBase = "/tmp/knuu"
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
		return ErrPortNumberOutOfRange.WithParams(port)
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
		labelAppKey:         i.k8sName,
		labelManagedByKey:   labelKnuuValue,
		labelScopeKey:       i.TestScope,
		labelTestStartedKey: i.StartTime,
		labelNameKey:        i.name,
		labelK8sNameKey:     i.k8sName,
		labelTypeKey:        i.instanceType.String(),
	}

}

// Labels returns the labels for the instance
func (i *Instance) Labels() map[string]string {
	return i.getLabels()
}

// deployService deploys the service for the instance
func (i *Instance) deployService(ctx context.Context, portsTCP, portsUDP []int) error {
	// a sidecar instance should use the parent instance's service
	if i.isSidecar {
		return ErrDeployingServiceForSidecar.WithParams(i.k8sName)
	}

	var (
		serviceName    = i.k8sName
		labels         = i.getLabels()
		labelSelectors = labels
	)

	srv, err := i.K8sClient.CreateService(ctx, serviceName, labels, labelSelectors, portsTCP, portsUDP)
	if err != nil {
		return ErrDeployingService.WithParams(i.k8sName).Wrap(err)
	}
	i.kubernetesService = srv
	i.Logger.Debugf("Started service '%s'", i.k8sName)
	return nil
}

// patchService patches the service for the instance
func (i *Instance) patchService(ctx context.Context, portsTCP, portsUDP []int) error {
	// a sidecar instance should use the parent instance's service
	if i.isSidecar {
		return ErrPatchingServiceForSidecar.WithParams(i.k8sName)
	}

	var (
		serviceName    = i.k8sName
		labels         = i.getLabels()
		labelSelectors = labels
	)

	srv, err := i.K8sClient.PatchService(ctx, serviceName, labels, labelSelectors, portsTCP, portsUDP)
	if err != nil {
		return ErrPatchingService.WithParams(serviceName).Wrap(err)
	}
	i.kubernetesService = srv
	i.Logger.Debugf("Patched service '%s'", serviceName)
	return nil
}

// destroyService destroys the service for the instance
func (i *Instance) destroyService(ctx context.Context) error {
	return i.K8sClient.DeleteService(ctx, i.k8sName)
}

// deployPod deploys the pod for the instance
func (i *Instance) deployPod(ctx context.Context) error {
	// Get labels for the pod
	labels := i.getLabels()

	// create a service account for the pod
	if err := i.K8sClient.CreateServiceAccount(ctx, i.k8sName, labels); err != nil {
		return ErrFailedToCreateServiceAccount.Wrap(err)
	}

	// create a role and role binding for the pod if there are policy rules
	if len(i.policyRules) > 0 {
		if err := i.K8sClient.CreateRole(ctx, i.k8sName, labels, i.policyRules); err != nil {
			return ErrFailedToCreateRole.Wrap(err)
		}
		if err := i.K8sClient.CreateRoleBinding(ctx, i.k8sName, labels, i.k8sName, i.k8sName); err != nil {
			return ErrFailedToCreateRoleBinding.Wrap(err)
		}
	}

	// Deploy the statefulSet
	replicaSet, err := i.K8sClient.CreateReplicaSet(ctx, i.prepareReplicaSetConfig(), true)
	if err != nil {
		return ErrFailedToDeployPod.Wrap(err)
	}

	// Set the state of the instance to started
	i.kubernetesReplicaSet = replicaSet

	// Log the deployment of the pod
	i.Logger.Debugf("Started statefulSet '%s'", i.k8sName)
	i.Logger.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// destroyPod destroys the pod for the instance (no grace period)
// Skips if the pod is already destroyed
func (i *Instance) destroyPod(ctx context.Context) error {
	err := i.K8sClient.DeleteReplicaSetWithGracePeriod(ctx, i.k8sName, nil)
	if err != nil {
		return ErrFailedToDeletePod.Wrap(err)
	}

	// Delete the service account for the pod
	if err := i.K8sClient.DeleteServiceAccount(ctx, i.k8sName); err != nil {
		return ErrFailedToDeleteServiceAccount.Wrap(err)
	}

	// Delete the role and role binding for the pod if there are policy rules
	if len(i.policyRules) == 0 {
		return nil
	}

	if err := i.K8sClient.DeleteRole(ctx, i.k8sName); err != nil {
		return ErrFailedToDeleteRole.Wrap(err)
	}
	if err := i.K8sClient.DeleteRoleBinding(ctx, i.k8sName); err != nil {
		return ErrFailedToDeleteRoleBinding.Wrap(err)
	}

	return nil
}

// deployService deploys the service for the instance
func (i *Instance) deployOrPatchService(ctx context.Context, portsTCP, portsUDP []int) error {
	if len(portsTCP) == 0 && len(portsUDP) == 0 {
		return nil
	}

	i.Logger.Debugf("Ports not empty, deploying service for instance '%s'", i.k8sName)
	svc, _ := i.K8sClient.GetService(ctx, i.k8sName)
	if svc == nil {
		if err := i.deployService(ctx, portsTCP, portsUDP); err != nil {
			return ErrDeployingServiceForInstance.WithParams(i.k8sName).Wrap(err)
		}
		return nil
	}

	if err := i.patchService(ctx, portsTCP, portsUDP); err != nil {
		return ErrPatchingServiceForInstance.WithParams(i.k8sName).Wrap(err)
	}
	return nil
}

// deployVolume deploys the volume for the instance
func (i *Instance) deployVolume(ctx context.Context) error {
	totalSize := resource.Quantity{}
	for _, volume := range i.volumes {
		totalSize.Add(volume.Size)
	}
	i.K8sClient.CreatePersistentVolumeClaim(ctx, i.k8sName, i.getLabels(), totalSize)
	i.Logger.Debugf("Deployed persistent volume '%s'", i.k8sName)

	return nil
}

// destroyVolume destroys the volume for the instance
func (i *Instance) destroyVolume(ctx context.Context) error {
	err := i.K8sClient.DeletePersistentVolumeClaim(ctx, i.k8sName)
	if err != nil {
		return ErrFailedToDeletePersistentVolumeClaim.Wrap(err)
	}
	i.Logger.Debugf("Destroyed persistent volume '%s'", i.k8sName)
	return nil
}

// deployFiles deploys the files for the instance
func (i *Instance) deployFiles(ctx context.Context) error {
	data := map[string]string{}

	for i, file := range i.files {
		// read out file content and assign to variable
		srcFile, err := os.Open(file.Source)
		if err != nil {
			return ErrFailedToOpenFile.Wrap(err)
		}
		defer srcFile.Close()

		fileContentBytes, err := io.ReadAll(srcFile)
		if err != nil {
			return ErrFailedToReadFile.Wrap(err)
		}

		var (
			fileContent = string(fileContentBytes)
			keyName     = fmt.Sprintf("%d", i)
		)

		data[keyName] = fileContent
	}

	// create configmap
	if _, err := i.K8sClient.CreateConfigMap(ctx, i.k8sName, i.getLabels(), data); err != nil {
		return ErrFailedToCreateConfigMap.Wrap(err)
	}

	i.Logger.Debugf("Deployed configmap '%s'", i.k8sName)

	return nil
}

// destroyFiles destroys the files for the instance
func (i *Instance) destroyFiles(ctx context.Context) error {
	if err := i.K8sClient.DeleteConfigMap(ctx, i.k8sName); err != nil {
		return ErrFailedToDeleteConfigMap.Wrap(err)
	}

	i.Logger.Debugf("Destroyed configmap '%s'", i.k8sName)
	return nil
}

// deployResources deploys the resources for the instance
func (i *Instance) deployResources(ctx context.Context) error {
	// only a non-sidecar instance should deploy a service, all sidecars will use the parent instance's service
	if !i.isSidecar {
		portsTCP := i.portsTCP
		portsUDP := i.portsUDP
		for _, sidecar := range i.sidecars {
			portsTCP = append(portsTCP, sidecar.portsTCP...)
			portsUDP = append(portsUDP, sidecar.portsUDP...)
		}
		if len(portsTCP) != 0 || len(portsUDP) != 0 {
			if err := i.deployOrPatchService(ctx, portsTCP, portsUDP); err != nil {
				return ErrFailedToDeployOrPatchService.Wrap(err)
			}
		}
	}
	if len(i.volumes) != 0 {
		if err := i.deployVolume(ctx); err != nil {
			return ErrDeployingVolumeForInstance.WithParams(i.k8sName).Wrap(err)
		}
	}
	if len(i.files) == 0 {
		return nil
	}

	if err := i.deployFiles(ctx); err != nil {
		return ErrDeployingFilesForInstance.WithParams(i.k8sName).Wrap(err)
	}
	return nil
}

// destroyResources destroys the resources for the instance
func (i *Instance) destroyResources(ctx context.Context) error {
	if len(i.volumes) != 0 {
		if err := i.destroyVolume(ctx); err != nil {
			return ErrDestroyingVolumeForInstance.WithParams(i.k8sName).Wrap(err)
		}
	}
	if len(i.files) != 0 {
		err := i.destroyFiles(ctx)
		if err != nil {
			return ErrDestroyingFilesForInstance.WithParams(i.k8sName).Wrap(err)
		}
	}
	if i.kubernetesService != nil {
		err := i.destroyService(ctx)
		if err != nil {
			return ErrDestroyingServiceForInstance.WithParams(i.k8sName).Wrap(err)
		}
	}

	// disable network only for non-sidecar instances
	if !i.isSidecar {
		// enable network when network is disabled
		if err := i.enableNetworkIfDisabled(ctx); err != nil {
			return ErrEnablingNetworkForInstance.WithParams(i.k8sName).Wrap(err)
		}
	}

	return nil
}

func (i *Instance) enableNetworkIfDisabled(ctx context.Context) error {
	disableNetwork, err := i.NetworkIsDisabled(ctx)
	if err != nil {
		i.Logger.Errorf("error checking network status for instance")
		return ErrCheckingNetworkStatusForInstance.WithParams(i.k8sName).Wrap(err)
	}
	if !disableNetwork {
		return nil
	}
	if err := i.EnableNetwork(ctx); err != nil {
		i.Logger.Errorf("error enabling network for instance")
		return ErrEnablingNetworkForInstance.WithParams(i.k8sName).Wrap(err)
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
		kubernetesReplicaSet:  i.kubernetesReplicaSet,
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
		tsharkCollectorConfig: i.tsharkCollectorConfig,
		securityContext:       &clonedSecurityContext,
		BitTwister:            &clonedBitTwister,
		SystemDependencies:    i.SystemDependencies,
	}
}

// getFreePort returns a free port
func getFreePortTCP() (int, error) {
	// Get a random port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, ErrGettingFreePort.Wrap(err)
	}
	defer listener.Close()

	// Get the port from the listener
	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}

// getBuildDir returns the build directory for the instance
func (i *Instance) getBuildDir() string {
	return filepath.Join(buildDirBase, i.k8sName)
}

// validateFileArgs validates the file arguments
func (i *Instance) validateFileArgs(src, dest, chown string) error {
	if src == "" {
		return ErrSrcMustBeSet
	}
	if dest == "" {
		return ErrDestMustBeSet
	}
	if chown == "" {
		return ErrChownMustBeSet
	}

	// validate chown format
	if !strings.Contains(chown, ":") || len(strings.Split(chown, ":")) != 2 {
		return ErrChownMustBeInFormatUserGroup
	}
	return nil
}

// addFileToBuilder adds a file to the builder
func (i *Instance) addFileToBuilder(src, dest, chown string) error {
	_ = src
	// dest is the same as src here, as we copy the file to the build dir with the subfolder structure of dest
	err := i.builderFactory.AddToBuilder(dest, dest, chown)
	if err != nil {
		return ErrAddingFileToInstance.WithParams(dest, i.name).Wrap(err)
	}
	return nil
}

// prepareSecurityContext creates a v1.SecurityContext from a given SecurityContext.
func prepareSecurityContext(config *SecurityContext) *v1.SecurityContext {
	securityContext := &v1.SecurityContext{}
	if config == nil {
		return securityContext
	}

	if config.privileged {
		securityContext.Privileged = &config.privileged
	}

	if len(config.capabilitiesAdd) == 0 {
		return securityContext
	}

	capabilities := make([]v1.Capability, len(config.capabilitiesAdd))
	for i, cap := range config.capabilitiesAdd {
		capabilities[i] = v1.Capability(cap)
	}
	securityContext.Capabilities = &v1.Capabilities{
		Add: capabilities,
	}

	return securityContext
}

// prepareConfig prepares the config for the instance
func (i *Instance) prepareReplicaSetConfig() k8s.ReplicaSetConfig {
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

	podConfig := k8s.PodConfig{
		Namespace:          i.K8sClient.Namespace(),
		Name:               i.k8sName,
		Labels:             i.getLabels(),
		ServiceAccountName: i.k8sName,
		FsGroup:            i.fsGroup,
		ContainerConfig:    containerConfig,
		SidecarConfigs:     sidecarConfigs,
	}

	return k8s.ReplicaSetConfig{
		Namespace: i.K8sClient.Namespace(),
		Name:      i.k8sName,
		Labels:    i.getLabels(),
		Replicas:  1,
		PodConfig: podConfig,
	}
}

// setImageWithGracePeriod sets the image of the instance with a grace period
func (i *Instance) setImageWithGracePeriod(ctx context.Context, imageName string, gracePeriod *int64) error {
	i.imageName = imageName

	_, err := i.K8sClient.ReplaceReplicaSetWithGracePeriod(ctx, i.prepareReplicaSetConfig(), gracePeriod)
	if err != nil {
		return ErrReplacingPod.Wrap(err)
	}

	if err := i.WaitInstanceIsRunning(ctx); err != nil {
		return ErrWaitingInstanceIsRunning.Wrap(err)
	}

	return nil
}

// applyFunctionToInstances applies a function to all instances
func applyFunctionToInstances(instances []*Instance, function func(sidecar *Instance) error) error {
	for _, i := range instances {
		if err := function(i); err != nil {
			return ErrApplyingFunctionToInstance.WithParams(i.k8sName).Wrap(err)
		}
	}
	return nil
}

func setStateForSidecars(sidecars []*Instance, state InstanceState) {
	_ = applyFunctionToInstances(sidecars, func(sidecar *Instance) error {
		sidecar.state = state
		return nil
	})
}

// isObservabilityEnabled returns true if observability is enabled
func (i *Instance) isObservabilityEnabled() bool {
	return i.obsyConfig.otlpPort != 0 ||
		i.obsyConfig.prometheusEndpointPort != 0 ||
		i.obsyConfig.jaegerGrpcPort != 0 ||
		i.obsyConfig.jaegerThriftCompactPort != 0 ||
		i.obsyConfig.jaegerThriftHttpPort != 0
}

func (i *Instance) validateStateForObsy(endpoint string) error {
	if !i.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingNotAllowed.WithParams(endpoint, i.state.String())
	}
	return nil
}

func (i *Instance) addOtelCollectorSidecar(ctx context.Context) error {
	otelSidecar, err := i.createOtelCollectorInstance(ctx)
	if err != nil {
		return ErrCreatingOtelCollectorInstance.WithParams(i.k8sName).Wrap(err)
	}
	if err := i.AddSidecar(otelSidecar); err != nil {
		return ErrAddingOtelCollectorSidecar.WithParams(i.k8sName).Wrap(err)
	}
	return nil
}

func (i *Instance) addTsharkCollectorSidecar(ctx context.Context) error {
	tsharkSidecar, err := i.createTsharkCollectorInstance(ctx)
	if err != nil {
		return ErrCreatingTsharkCollectorInstance.WithParams(i.k8sName).Wrap(err)
	}
	if err := i.AddSidecar(tsharkSidecar); err != nil {
		return ErrAddingTsharkCollectorSidecar.WithParams(i.k8sName).Wrap(err)
	}
	return nil
}

func (i *Instance) createBitTwisterInstance(ctx context.Context) (*Instance, error) {
	bt, err := New("bit-twister", i.SystemDependencies)
	if err != nil {
		return nil, ErrCreatingBitTwisterInstance.Wrap(err)
	}

	if err := bt.SetImage(ctx, i.BitTwister.Image()); err != nil {
		return nil, ErrSettingBitTwisterImage.Wrap(err)
	}

	// This is needed to make BT reachable
	if err := bt.AddPortTCP(i.BitTwister.Port()); err != nil {
		return nil, ErrAddingBitTwisterPort.Wrap(err)
	}
	serviceName := i.k8sName // the main instance name
	btURL, err := i.AddHost(ctx, i.BitTwister.Port())
	if err != nil {
		return nil, ErrAddingToProxy.WithParams(bt.k8sName, serviceName).Wrap(err)
	}
	i.Logger.Debugf("BitTwister URL: %s", btURL)

	i.BitTwister.SetNewClientByURL(btURL)

	if err := bt.Commit(); err != nil {
		return nil, ErrCommittingBitTwisterInstance.Wrap(err)
	}

	if err := bt.SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", i.BitTwister.Port())); err != nil {
		return nil, ErrSettingBitTwisterEnv.Wrap(err)
	}

	return bt, nil
}

func (i *Instance) addBitTwisterSidecar(ctx context.Context) error {
	networkConfigSidecar, err := i.createBitTwisterInstance(ctx)
	if err != nil {
		return ErrCreatingBitTwisterInstance.WithParams(i.k8sName).Wrap(err)
	}

	if err := networkConfigSidecar.SetPrivileged(true); err != nil {
		return ErrSettingBitTwisterPrivileged.WithParams(i.k8sName).Wrap(err)
	}

	if err := networkConfigSidecar.AddCapability("NET_ADMIN"); err != nil {
		return ErrAddingBitTwisterCapability.WithParams(i.k8sName).Wrap(err)
	}

	if err := i.AddSidecar(networkConfigSidecar); err != nil {
		return ErrAddingBitTwisterSidecar.WithParams(i.k8sName).Wrap(err)
	}
	return nil
}

// isSubFolderOfVolumes checks if the given path is a subfolder of the volumes
func (i *Instance) isSubFolderOfVolumes(path string) bool {
	for _, volume := range i.volumes {
		if strings.HasPrefix(path, volume.Path) {
			return true
		}
	}
	return false
}
