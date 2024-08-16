package instance

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/celestiaorg/knuu/pkg/k8s"

	"github.com/sirupsen/logrus"
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
)

type execution struct {
	instance *Instance
}

func (i *Instance) Execution() *execution {
	return i.execution
}

// ExecuteCommand executes the given command in the instance
// This function can only be called in the states 'Started'
func (e *execution) ExecuteCommand(ctx context.Context, command ...string) (string, error) {
	if e.instance.state != StateStarted {
		return "", ErrExecutingCommandNotAllowed.WithParams(e.instance.state.String())
	}

	var (
		instanceName  string
		eErr          *Error
		containerName = e.instance.k8sName
	)

	if e.instance.sidecars.isSidecar {
		instanceName = e.instance.parentInstance.k8sName
		eErr = ErrExecutingCommandInSidecar.WithParams(command, e.instance.k8sName, e.instance.parentInstance.k8sName)
	} else {
		instanceName = e.instance.k8sName
		eErr = ErrExecutingCommandInInstance.WithParams(command, e.instance.k8sName)
	}

	pod, err := e.instance.K8sClient.GetFirstPodFromReplicaSet(ctx, instanceName)
	if err != nil {
		return "", ErrGettingPodFromReplicaSet.WithParams(e.instance.k8sName).Wrap(err)
	}

	commandWithShell := []string{"/bin/sh", "-c", strings.Join(command, " ")}
	output, err := e.instance.K8sClient.RunCommandInPod(ctx, pod.Name, containerName, commandWithShell)
	if err != nil {
		return "", eErr.Wrap(err)
	}
	return output, nil
}

// StartWithCallback starts the instance asynchronously and calls a callback function when the instance is running
// This function can only be called in the state 'Committed' or 'Stopped'
func (e *execution) StartWithCallback(ctx context.Context, callback func()) error {
	if err := e.StartAsync(ctx); err != nil {
		return err
	}
	go func() {
		err := e.WaitInstanceIsRunning(ctx)
		if err != nil {
			e.instance.Logger.Errorf("Error waiting for instance '%s' to be running: %s", e.instance.k8sName, err)
			return
		}
		callback()
	}()
	return nil
}

// StartAsync starts the instance without waiting for it to be ready
// This function can only be called in the state 'Committed' or 'Stopped'
func (e *execution) StartAsync(ctx context.Context) error {
	if !e.instance.IsInState(StateCommitted, StateStopped) {
		return ErrStartingNotAllowed.WithParams(e.instance.k8sName, e.instance.state.String())
	}

	if err := e.instance.sidecars.verifySidecarsStates(); err != nil {
		return err
	}
	err := e.instance.sidecars.applyFunctionToSidecars(
		func(sc SidecarManager) error {
			if !sc.Instance().IsInState(StateCommitted, StateStopped) {
				return ErrStartingNotAllowedForSidecar.
					WithParams(sc.Instance().Name(), sc.Instance().state.String())
			}
			return nil
		})
	if err != nil {
		return err
	}

	if e.instance.sidecars.isSidecar {
		return ErrStartingSidecarNotAllowed
	}

	if e.instance.state == StateCommitted {
		if err := e.deployResourcesForCommittedState(ctx); err != nil {
			return ErrDeployingResourcesForInstance.WithParams(e.instance.k8sName).Wrap(err)
		}
	}

	if err := e.deployPod(ctx); err != nil {
		return ErrDeployingPodForInstance.WithParams(e.instance.k8sName).Wrap(err)
	}

	e.instance.state = StateStarted
	e.instance.sidecars.setStateForSidecars(StateStarted)
	e.instance.Logger.Debugf("Set state of instance '%s' to '%s'", e.instance.k8sName, e.instance.state.String())

	return nil
}

// Start starts the instance and waits for it to be ready
// This function can only be called in the state 'Committed' and 'Stopped'
func (e *execution) Start(ctx context.Context) error {
	if err := e.StartAsync(ctx); err != nil {
		return err
	}

	if err := e.WaitInstanceIsRunning(ctx); err != nil {
		return ErrWaitingForInstanceRunning.WithParams(e.instance.k8sName).Wrap(err)
	}
	return nil
}

// IsRunning returns true if the instance is running
// This function can only be called in the state 'Started'
func (e *execution) IsRunning(ctx context.Context) (bool, error) {
	if !e.instance.IsInState(StateStarted, StateStopped) {
		return false, ErrCheckingIfInstanceRunningNotAllowed.WithParams(e.instance.state.String())
	}

	return e.instance.K8sClient.IsReplicaSetRunning(ctx, e.instance.k8sName)
}

// WaitInstanceIsRunning waits until the instance is running
// This function can only be called in the state 'Started'
func (e *execution) WaitInstanceIsRunning(ctx context.Context) error {
	if !e.instance.IsInState(StateStarted) {
		return ErrWaitingForInstanceNotAllowed.WithParams(e.instance.state.String())
	}

	for {
		running, err := e.IsRunning(ctx)
		if err != nil {
			return ErrCheckingIfInstanceRunning.WithParams(e.instance.k8sName).Wrap(err)
		}
		if running {
			return nil
		}

		select {
		case <-ctx.Done():
			return ErrWaitingForInstanceTimeout.
				WithParams(e.instance.k8sName).Wrap(ctx.Err())
		case <-time.After(waitForInstanceRetry):
			continue
		}
	}
}

// WaitInstanceIsStopped waits until the instance is not running anymore
// This function can only be called in the state 'Stopped'
func (e *execution) WaitInstanceIsStopped(ctx context.Context) error {
	if !e.instance.IsInState(StateStopped) {
		return ErrWaitingForInstanceStoppedNotAllowed.WithParams(e.instance.state.String())
	}
	for {
		running, err := e.IsRunning(ctx)
		if !running {
			break
		}
		if err != nil {
			return ErrCheckingIfInstanceStopped.WithParams(e.instance.k8sName).Wrap(err)
		}

		select {
		case <-ctx.Done():
			return ErrWaitingForInstanceTimeout.
				WithParams(e.instance.k8sName).Wrap(ctx.Err())
		case <-time.After(waitForInstanceRetry):
			continue
		}
	}

	return nil
}

// Stop stops the instance
// CAUTION: In order to keep data of the instance, you need to use AddVolume() before.
// This function can only be called in the state 'Started'
func (e *execution) Stop(ctx context.Context) error {
	if !e.instance.IsInState(StateStarted) {
		return ErrStoppingNotAllowed.WithParams(e.instance.state.String())

	}

	if err := e.destroyPod(ctx); err != nil {
		return ErrDestroyingPod.WithParams(e.instance.k8sName).Wrap(err)
	}
	e.instance.state = StateStopped
	e.instance.sidecars.setStateForSidecars(StateStopped)
	e.instance.Logger.Debugf("Set state of instance '%s' to '%s'", e.instance.k8sName, e.instance.state.String())

	return nil
}

// Labels returns the labels for the instance
func (e *execution) Labels() map[string]string {
	return map[string]string{
		labelAppKey:         e.instance.k8sName,
		labelManagedByKey:   labelKnuuValue,
		labelScopeKey:       e.instance.Scope,
		labelTestStartedKey: e.instance.StartTime,
		labelNameKey:        e.instance.name,
		labelK8sNameKey:     e.instance.k8sName,
		labelTypeKey:        e.instance.instanceType.String(),
	}
}

// Destroy destroys the instance
// This function can only be called in the state 'Started' or 'Destroyed'
func (e *execution) Destroy(ctx context.Context) error {
	if e.instance.state == StateDestroyed {
		return nil
	}

	if !e.instance.IsInState(StateStarted, StateStopped) {
		return ErrDestroyingNotAllowed.WithParams(e.instance.state.String())
	}

	if err := e.destroyPod(ctx); err != nil {
		return ErrDestroyingPod.WithParams(e.instance.k8sName).Wrap(err)
	}
	if err := e.instance.resources.destroyResources(ctx); err != nil {
		return ErrDestroyingResourcesForInstance.WithParams(e.instance.k8sName).Wrap(err)
	}

	err := e.instance.sidecars.applyFunctionToSidecars(
		func(sidecar SidecarManager) error {
			e.instance.Logger.Debugf("Destroying sidecar resources from '%s'", sidecar.Instance().k8sName)
			return sidecar.Instance().resources.destroyResources(ctx)
		})
	if err != nil {
		return ErrDestroyingResourcesForSidecars.WithParams(e.instance.k8sName).Wrap(err)
	}

	e.instance.state = StateDestroyed
	e.instance.sidecars.setStateForSidecars(StateDestroyed)
	e.instance.Logger.Debugf("Set state of instance '%s' to '%s'", e.instance.k8sName, e.instance.state.String())

	return nil
}

func (e *execution) UpgradeImage(ctx context.Context, image string) error {
	return e.UpgradeImageWithGracePeriod(ctx, image, 0)
}

func (e *execution) UpgradeImageWithGracePeriod(ctx context.Context, image string, gracePeriod time.Duration) error {
	if !e.instance.IsInState(StateStarted) {
		return ErrUpgradingImageNotAllowed.WithParams(e.instance.state.String())
	}

	return e.instance.build.setImageWithGracePeriod(ctx, image, gracePeriod)
}

// BatchDestroy destroys a list of instances.
func BatchDestroy(ctx context.Context, instances ...*Instance) error {
	if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
		logrus.Info("Skipping cleanup")
		return nil
	}

	for _, i := range instances {
		if i == nil {
			continue
		}
		if err := i.Execution().Destroy(ctx); err != nil {
			return err
		}
	}
	return nil
}

// deployResourcesForCommittedState handles resource deployment for instances in the 'Committed' state
func (e *execution) deployResourcesForCommittedState(ctx context.Context) error {
	if err := e.instance.resources.deployResources(ctx); err != nil {
		return ErrDeployingResourcesForInstance.WithParams(e.instance.k8sName).Wrap(err)
	}
	err := e.instance.sidecars.applyFunctionToSidecars(func(sc SidecarManager) error {
		if err := sc.PreStart(ctx); err != nil {
			return err
		}
		return sc.Instance().resources.deployResources(ctx)
	})
	if err != nil {
		return ErrDeployingResourcesForSidecars.WithParams(e.instance.k8sName).Wrap(err)
	}

	return nil
}

// deployPod deploys the pod for the instance
func (e *execution) deployPod(ctx context.Context) error {
	// Get labels for the pod
	labels := e.Labels()

	// create a service account for the pod
	if err := e.instance.K8sClient.CreateServiceAccount(ctx, e.instance.k8sName, labels); err != nil {
		return ErrFailedToCreateServiceAccount.Wrap(err)
	}

	// create a role and role binding for the pod if there are policy rules
	if len(e.instance.security.policyRules) > 0 {
		if err := e.instance.K8sClient.CreateRole(ctx, e.instance.k8sName, labels, e.instance.security.policyRules); err != nil {
			return ErrFailedToCreateRole.Wrap(err)
		}
		if err := e.instance.K8sClient.CreateRoleBinding(ctx, e.instance.k8sName, labels, e.instance.k8sName, e.instance.k8sName); err != nil {
			return ErrFailedToCreateRoleBinding.Wrap(err)
		}
	}

	// Deploy the statefulSet
	replicaSet, err := e.instance.K8sClient.CreateReplicaSet(ctx, e.prepareReplicaSetConfig(), true)
	if err != nil {
		return ErrFailedToDeployPod.Wrap(err)
	}

	// Set the state of the instance to started
	e.instance.kubernetesReplicaSet = replicaSet

	// Log the deployment of the pod
	e.instance.Logger.Debugf("Started statefulSet '%s'", e.instance.k8sName)
	e.instance.Logger.Debugf("Set state of instance '%s' to '%s'", e.instance.k8sName, e.instance.state.String())

	return nil
}

// destroyPod destroys the pod for the instance (no grace period)
// Skips if the pod is already destroyed
func (e *execution) destroyPod(ctx context.Context) error {
	err := e.instance.K8sClient.DeleteReplicaSetWithGracePeriod(ctx, e.instance.k8sName, nil)
	if err != nil {
		return ErrFailedToDeletePod.Wrap(err)
	}

	// Delete the service account for the pod
	if err := e.instance.K8sClient.DeleteServiceAccount(ctx, e.instance.k8sName); err != nil {
		return ErrFailedToDeleteServiceAccount.Wrap(err)
	}

	// Delete the role and role binding for the pod if there are policy rules
	if len(e.instance.security.policyRules) == 0 {
		return nil
	}

	if err := e.instance.K8sClient.DeleteRole(ctx, e.instance.k8sName); err != nil {
		return ErrFailedToDeleteRole.Wrap(err)
	}
	if err := e.instance.K8sClient.DeleteRoleBinding(ctx, e.instance.k8sName); err != nil {
		return ErrFailedToDeleteRoleBinding.Wrap(err)
	}

	return nil
}

// prepareConfig prepares the config for the instance
func (e *execution) prepareReplicaSetConfig() k8s.ReplicaSetConfig {
	containerConfig := k8s.ContainerConfig{
		Name:            e.instance.k8sName,
		Image:           e.instance.build.imageName,
		ImagePullPolicy: e.instance.build.imagePullPolicy,
		Command:         e.instance.build.command,
		Args:            e.instance.build.args,
		Env:             e.instance.build.env,
		Volumes:         e.instance.storage.volumes,
		MemoryRequest:   e.instance.resources.memoryRequest,
		MemoryLimit:     e.instance.resources.memoryLimit,
		CPURequest:      e.instance.resources.cpuRequest,
		LivenessProbe:   e.instance.monitoring.livenessProbe,
		ReadinessProbe:  e.instance.monitoring.readinessProbe,
		StartupProbe:    e.instance.monitoring.startupProbe,
		Files:           e.instance.storage.files,
		SecurityContext: e.instance.security.prepareSecurityContext(),
	}

	sidecarConfigs := make([]k8s.ContainerConfig, 0)
	for _, sidecar := range e.instance.sidecars.sidecars {
		sidecarConfigs = append(sidecarConfigs, k8s.ContainerConfig{
			Name:            sidecar.Instance().k8sName,
			Image:           sidecar.Instance().build.imageName,
			Command:         sidecar.Instance().build.command,
			Args:            sidecar.Instance().build.args,
			Env:             sidecar.Instance().build.env,
			Volumes:         sidecar.Instance().storage.volumes,
			MemoryRequest:   sidecar.Instance().resources.memoryRequest,
			MemoryLimit:     sidecar.Instance().resources.memoryLimit,
			CPURequest:      sidecar.Instance().resources.cpuRequest,
			LivenessProbe:   sidecar.Instance().monitoring.livenessProbe,
			ReadinessProbe:  sidecar.Instance().monitoring.readinessProbe,
			StartupProbe:    sidecar.Instance().monitoring.startupProbe,
			Files:           sidecar.Instance().storage.files,
			SecurityContext: sidecar.Instance().security.prepareSecurityContext(),
		})
	}

	podConfig := k8s.PodConfig{
		Namespace:          e.instance.K8sClient.Namespace(),
		Name:               e.instance.k8sName,
		Labels:             e.Labels(),
		ServiceAccountName: e.instance.k8sName,
		FsGroup:            e.instance.storage.fsGroup,
		ContainerConfig:    containerConfig,
		SidecarConfigs:     sidecarConfigs,
	}

	return k8s.ReplicaSetConfig{
		Namespace: e.instance.K8sClient.Namespace(),
		Name:      e.instance.k8sName,
		Labels:    e.Labels(),
		Replicas:  1,
		PodConfig: podConfig,
	}
}

func (e *execution) clone() *execution {
	return &execution{instance: nil}
}
