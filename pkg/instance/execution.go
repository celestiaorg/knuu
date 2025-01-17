package instance

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/machine"
	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/inlets/cloud-provision/provision"

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
		containerName = e.instance.name
	)

	if e.instance.sidecars.isSidecar {
		instanceName = e.instance.parentInstance.name
		eErr = ErrExecutingCommandInSidecar.WithParams(command, e.instance.name, e.instance.parentInstance.name)
	} else {
		instanceName = e.instance.name
		eErr = ErrExecutingCommandInInstance.WithParams(command, e.instance.name)
	}

	pod, err := e.instance.K8sClient.GetFirstPodFromReplicaSet(ctx, instanceName)
	if err != nil {
		return "", ErrGettingPodFromReplicaSet.WithParams(e.instance.name).Wrap(err)
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
			e.instance.Logger.WithError(err).WithField("instance", e.instance.name).Error("waiting for instance to be running")
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
		return ErrStartingNotAllowed.WithParams(e.instance.name, e.instance.state.String())
	}

	accessKey := os.Getenv("SCALEWAY_ACCESS_KEY")
	if accessKey == "" {
		return fmt.Errorf("SCALEWAY_ACCESS_KEY environment variable is not set")
	}
	secretKey := os.Getenv("SCALEWAY_SECRET_KEY")
	if secretKey == "" {
		return fmt.Errorf("SCALEWAY_SECRET_KEY environment variable is not set")
	}
	projectID := os.Getenv("SCALEWAY_PROJECT_ID")
	if projectID == "" {
		return fmt.Errorf("SCALEWAY_PROJECT_ID environment variable is not set")
	}
	region := os.Getenv("SCALEWAY_REGION")
	if region == "" {
		return fmt.Errorf("SCALEWAY_REGION environment variable is not set")
	}

	provisioner, err := provision.NewScalewayProvisioner(accessKey, secretKey, projectID, region)
	if err != nil {
		return err
	}

	// doToken := os.Getenv("DIGITALOCEAN_TOKEN")
	// if doToken == "" {
	// 	return fmt.Errorf("DIGITALOCEAN_TOKEN environment variable is not set")
	// }

	// provisioner, err := provision.NewDigitalOceanProvisioner(doToken)
	// if err != nil {
	// 	return err
	// }

	// Read public and private keys from file paths
	publicKeyPath := "/Users/peter/.ssh/test-knuu-vm.pub"
	privateKeyPath := "/Users/peter/.ssh/test-knuu-vm"

	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %v", err)
	}

	// privateKey, err := os.ReadFile(privateKeyPath)
	// if err != nil {
	// 	return fmt.Errorf("failed to read private key: %v", err)
	// }

	logger := log.New(os.Stdout, "test", log.LstdFlags)

	machine, err := machine.NewMachine(
		logger,
		provisioner,
		machine.Regions.Scaleway.FR_PAR_1,
		machine.Sizes.Scaleway.DEV1S,
		e.instance.Scope+"-"+e.instance.name,
		machine.OS.Scaleway.Ubuntu2404,
		[]string{
			"#!/bin/bash",
			// // Grant Polkit permissions for the default ubuntu user (NEEDED?)
			// "echo 'polkit.addRule(function(action, subject) { if (subject.isInGroup(\"sudo\")) { return polkit.Result.YES; } });' | sudo tee /etc/polkit-1/rules.d/10-nopasswd.rules",
			// // Reload Polkit to apply changes (NEEDED?)
			// "sudo systemctl restart polkit",
			// Setup SSH access for the default ubuntu user
			"mkdir -p /home/ubuntu/.ssh",
			fmt.Sprintf("echo '%s' >> /home/ubuntu/.ssh/authorized_keys", string(publicKey)),
			// Disable password authentication for SSH
			"sudo sed -i 's/^#PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config",
			"sudo systemctl restart sshd",
			// Update package list and install Podman (including needed dependencies)
			"sudo apt-get update",
			"sudo apt-get -y install podman uidmap slirp4netns --no-install-recommends",
			// Set the runtime directory and DBUS session address
			"export XDG_RUNTIME_DIR=/run/user/$(id -u ubuntu)",
			"export DBUS_SESSION_BUS_ADDRESS=unix:path=${XDG_RUNTIME_DIR}/bus",

			// Enable and start podman.socket for the user
			"runuser -l ubuntu -c \"systemctl --user enable podman.socket\"",
			"runuser -l ubuntu -c \"systemctl --user start podman.socket\"",
		},
	)
	if err != nil {
		return err
	}

	err = machine.WaitForCreation()
	if err != nil {
		return err
	}

	ip := machine.GetIP()
	logger.Printf("IP: %s\n", ip)

	// Wait until the Podman socket is available with a timeout
	uri := "ssh://ubuntu@" + ip + ":22/run/user/1000/podman/podman.sock"
	timeout := time.After(5 * time.Minute)
	tick := time.NewTicker(10 * time.Second)

	var connText context.Context
	socketAvailable := false
	for !socketAvailable {
		select {
		case <-timeout:
			return fmt.Errorf("timeout reached while waiting for Podman socket to become available")
		case <-tick.C:
			e.instance.Logger.WithField("instance", e.instance.name).Infof("attempting to connect to Podman socket")
			connText, err = bindings.NewConnectionWithIdentity(ctx, uri, privateKeyPath, true)
			if err == nil {
				e.instance.Logger.WithField("instance", e.instance.name).Infof("connected to Podman socket")
				socketAvailable = true
			} else {
				e.instance.Logger.WithField("instance", e.instance.name).Infof("podman socket not available yet, retrying...")
			}
		}
	}

	// Image pull
	pullOptions := &images.PullOptions{}
	_, err = images.Pull(connText, e.instance.build.imageName, pullOptions)
	if err != nil {
		return fmt.Errorf("failed to pull image: %v", err)
	}

	// Image list
	imageListOptions := &images.ListOptions{}
	imageList, err := images.List(connText, imageListOptions)
	if err != nil {
		return fmt.Errorf("failed to list images: %v", err)
	}
	e.instance.Logger.WithField("instance", e.instance.name).Infof("image list: %v", imageList)

	// Container create
	s := specgen.NewSpecGenerator(e.instance.build.imageName, false)
	s.Name = e.instance.name
	if e.instance.build.command != nil {
		s.Entrypoint = e.instance.build.command
	}
	if e.instance.build.args != nil {
		s.Command = e.instance.build.args
	}
	if e.instance.build.env != nil {
		s.Env = e.instance.build.env
	}
	// TODO: add port mappings
	// if e.instance.network.portsTCP != nil {
	// 	s.PortMappings = e.instance.network.portsTCP
	// }
	// if e.instance.network.portsUDP != nil {
	// 	s.PortMappings = e.instance.network.portsUDP
	// }
	createOptions := &containers.CreateOptions{}
	r, err := containers.CreateWithSpec(connText, s, createOptions)
	if err != nil {
		return fmt.Errorf("failed to create container: %v", err)
	}

	// Container start
	startOptions := &containers.StartOptions{}
	err = containers.Start(connText, r.ID, startOptions)
	if err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	// Wait for container to be running
	waitOptions := &containers.WaitOptions{
		Condition: []define.ContainerStatus{define.ContainerStateRunning},
	}
	_, err = containers.Wait(connText, r.ID, waitOptions)
	if err != nil {
		return fmt.Errorf("failed to wait for container: %v", err)
	}

	// Container list
	listOptions := &containers.ListOptions{}
	containerLatestList, err := containers.List(connText, listOptions)
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}
	e.instance.Logger.WithField("instance", e.instance.name).Infof("latest container is %s", containerLatestList[0].Names[0])

	time.Sleep(1 * time.Hour)

	machine.Remove(ctx)

	return nil

	// if err := e.instance.sidecars.verifySidecarsStates(); err != nil {
	// 	return err
	// }
	// err := e.instance.sidecars.applyFunctionToSidecars(
	// 	func(sc SidecarManager) error {
	// 		if !sc.Instance().IsInState(StateCommitted, StateStopped) {
	// 			return ErrStartingNotAllowedForSidecar.
	// 				WithParams(sc.Instance().Name(), sc.Instance().state.String())
	// 		}
	// 		return nil
	// 	})
	// if err != nil {
	// 	return err
	// }

	// if e.instance.sidecars.isSidecar {
	// 	return ErrStartingSidecarNotAllowed
	// }

	// if e.instance.state == StateCommitted {
	// 	if err := e.deployResourcesForCommittedState(ctx); err != nil {
	// 		return ErrDeployingResourcesForInstance.WithParams(e.instance.name).Wrap(err)
	// 	}
	// }

	// if err := e.deployPod(ctx); err != nil {
	// 	return ErrDeployingPodForInstance.WithParams(e.instance.name).Wrap(err)
	// }

	// e.instance.SetState(StateStarted)
	// e.instance.sidecars.setStateForSidecars(StateStarted)
	// return nil
}

// Start starts the instance and waits for it to be ready
// This function can only be called in the state 'Committed' and 'Stopped'
func (e *execution) Start(ctx context.Context) error {
	if err := e.StartAsync(ctx); err != nil {
		return err
	}

	if err := e.WaitInstanceIsRunning(ctx); err != nil {
		return ErrWaitingForInstanceRunning.WithParams(e.instance.name).Wrap(err)
	}
	return nil
}

// IsRunning returns true if the instance is running
// This function can only be called in the state 'Started'
func (e *execution) IsRunning(ctx context.Context) (bool, error) {
	if !e.instance.IsInState(StateStarted, StateStopped) {
		return false, ErrCheckingIfInstanceRunningNotAllowed.WithParams(e.instance.state.String())
	}

	return e.instance.K8sClient.IsReplicaSetRunning(ctx, e.instance.name)
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
			return ErrCheckingIfInstanceRunning.WithParams(e.instance.name).Wrap(err)
		}
		if running {
			return nil
		}

		select {
		case <-ctx.Done():
			return ErrWaitingForInstanceTimeout.
				WithParams(e.instance.name).Wrap(ctx.Err())
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
			return ErrCheckingIfInstanceStopped.WithParams(e.instance.name).Wrap(err)
		}

		select {
		case <-ctx.Done():
			return ErrWaitingForInstanceTimeout.
				WithParams(e.instance.name).Wrap(ctx.Err())
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
		return ErrDestroyingPod.WithParams(e.instance.name).Wrap(err)
	}

	e.instance.SetState(StateStopped)
	e.instance.sidecars.setStateForSidecars(StateStopped)
	return nil
}

func (b *execution) SetImage(ctx context.Context, image string) error {
	return b.instance.build.SetImage(ctx, image)
}

// Labels returns the labels for the instance
func (e *execution) Labels() map[string]string {
	return map[string]string{
		labelAppKey:         e.instance.name,
		labelManagedByKey:   labelKnuuValue,
		labelScopeKey:       e.instance.Scope,
		labelTestStartedKey: e.instance.StartTime,
		labelNameKey:        e.instance.name,
		labelK8sNameKey:     e.instance.name,
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
		return ErrDestroyingPod.WithParams(e.instance.name).Wrap(err)
	}
	if err := e.instance.resources.destroyResources(ctx); err != nil {
		return ErrDestroyingResourcesForInstance.WithParams(e.instance.name).Wrap(err)
	}

	err := e.instance.sidecars.applyFunctionToSidecars(
		func(sidecar SidecarManager) error {
			e.instance.Logger.WithFields(logrus.Fields{
				"instance": e.instance.name,
				"sidecar":  sidecar.Instance().name,
			}).Debugf("destroying sidecar resources")
			return sidecar.Instance().resources.destroyResources(ctx)
		})
	if err != nil {
		return ErrDestroyingResourcesForSidecars.WithParams(e.instance.name).Wrap(err)
	}

	e.instance.SetState(StateDestroyed)
	e.instance.sidecars.setStateForSidecars(StateDestroyed)
	return nil
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
		return ErrDeployingResourcesForInstance.WithParams(e.instance.name).Wrap(err)
	}
	err := e.instance.sidecars.applyFunctionToSidecars(func(sc SidecarManager) error {
		if err := sc.PreStart(ctx); err != nil {
			return err
		}
		return sc.Instance().resources.deployResources(ctx)
	})
	if err != nil {
		return ErrDeployingResourcesForSidecars.WithParams(e.instance.name).Wrap(err)
	}

	return nil
}

// deployPod deploys the pod for the instance
func (e *execution) deployPod(ctx context.Context) error {
	// Get labels for the pod
	labels := e.Labels()

	// create a service account for the pod
	if err := e.instance.K8sClient.CreateServiceAccount(ctx, e.instance.name, labels); err != nil {
		return ErrFailedToCreateServiceAccount.Wrap(err)
	}

	// create a role and role binding for the pod if there are policy rules
	if len(e.instance.security.policyRules) > 0 {
		if err := e.instance.K8sClient.CreateRole(ctx, e.instance.name, labels, e.instance.security.policyRules); err != nil {
			return ErrFailedToCreateRole.Wrap(err)
		}
		if err := e.instance.K8sClient.CreateRoleBinding(ctx, e.instance.name, labels, e.instance.name, e.instance.name); err != nil {
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
	e.instance.Logger.WithField("instance", e.instance.name).Debugf("started statefulSet")
	return nil
}

// destroyPod destroys the pod for the instance (no grace period)
// Skips if the pod is already destroyed
func (e *execution) destroyPod(ctx context.Context) error {
	err := e.instance.K8sClient.DeleteReplicaSetWithGracePeriod(ctx, e.instance.name, nil)
	if err != nil {
		return ErrFailedToDeletePod.Wrap(err)
	}

	// Delete the service account for the pod
	if err := e.instance.K8sClient.DeleteServiceAccount(ctx, e.instance.name); err != nil {
		return ErrFailedToDeleteServiceAccount.Wrap(err)
	}

	// Delete the role and role binding for the pod if there are policy rules
	if len(e.instance.security.policyRules) == 0 {
		return nil
	}

	if err := e.instance.K8sClient.DeleteRole(ctx, e.instance.name); err != nil {
		return ErrFailedToDeleteRole.Wrap(err)
	}
	if err := e.instance.K8sClient.DeleteRoleBinding(ctx, e.instance.name); err != nil {
		return ErrFailedToDeleteRoleBinding.Wrap(err)
	}

	return nil
}

// prepareConfig prepares the config for the instance
func (e *execution) prepareReplicaSetConfig() k8s.ReplicaSetConfig {
	containerConfig := k8s.ContainerConfig{
		Name:            e.instance.name,
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
		TCPPorts:        e.instance.network.portsTCP,
		UDPPorts:        e.instance.network.portsUDP,
	}

	sidecarConfigs := make([]k8s.ContainerConfig, 0)
	for _, sidecar := range e.instance.sidecars.sidecars {
		sidecarConfigs = append(sidecarConfigs, k8s.ContainerConfig{
			Name:            sidecar.Instance().name,
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
			TCPPorts:        sidecar.Instance().network.portsTCP,
			UDPPorts:        sidecar.Instance().network.portsUDP,
		})
	}

	podConfig := k8s.PodConfig{
		Namespace:          e.instance.K8sClient.Namespace(),
		Name:               e.instance.name,
		Labels:             e.Labels(),
		ServiceAccountName: e.instance.name,
		ContainerConfig:    containerConfig,
		SidecarConfigs:     sidecarConfigs,
		NodeSelector:       e.instance.build.nodeSelector,
	}

	return k8s.ReplicaSetConfig{
		Namespace: e.instance.K8sClient.Namespace(),
		Name:      e.instance.name,
		Labels:    e.Labels(),
		Replicas:  1,
		PodConfig: podConfig,
	}
}

func (e *execution) clone() *execution {
	return &execution{instance: nil}
}
