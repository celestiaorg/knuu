package k8s

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
)

// the loops that keep checking something and wait for it to be done
const (
	// retryInterval is the interval to wait between retries
	retryInterval = 100 * time.Millisecond

	// knuuPath is the path where the knuu volume is mounted
	knuuPath = "/knuu"
)

type ContainerConfig struct {
	Name            string              // Name to assign to the Container
	Image           string              // Name of the container image to use for the container
	Command         []string            // Command to run in the container
	Args            []string            // Arguments to pass to the command in the container
	Env             map[string]string   // Environment variables to set in the container
	Volumes         []*Volume           // Volumes to mount in the Pod
	MemoryRequest   resource.Quantity   // Memory request for the container
	MemoryLimit     resource.Quantity   // Memory limit for the container
	CPURequest      resource.Quantity   // CPU request for the container
	LivenessProbe   *v1.Probe           // Liveness probe for the container
	ReadinessProbe  *v1.Probe           // Readiness probe for the container
	StartupProbe    *v1.Probe           // Startup probe for the container
	Files           []*File             // Files to add to the Pod
	SecurityContext *v1.SecurityContext // Security context for the container
}

type PodConfig struct {
	Namespace          string            // Kubernetes namespace of the Pod
	Name               string            // Name to assign to the Pod
	Labels             map[string]string // Labels to apply to the Pod
	ServiceAccountName string            // ServiceAccount to assign to Pod
	FsGroup            int64             // FSGroup to apply to the Pod
	ContainerConfig    ContainerConfig   // ContainerConfig for the Pod
	SidecarConfigs     []ContainerConfig // SideCarConfigs for the Pod
	Annotations        map[string]string // Annotations to apply to the Pod
}

type Volume struct {
	Path  string
	Size  resource.Quantity
	Owner int64
}

type File struct {
	Source string
	Dest   string
}

// DeployPod creates a new pod in the namespace that k8s client is initiate with if it doesn't already exist.
func (c *Client) DeployPod(ctx context.Context, podConfig PodConfig, init bool) (*v1.Pod, error) {
	pod, err := preparePod(podConfig, init)
	if err != nil {
		return nil, ErrPreparingPod.Wrap(err)
	}
	createdPod, err := c.clientset.CoreV1().Pods(c.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, ErrCreatingPod.Wrap(err)
	}

	return createdPod, nil
}

func (c *Client) NewVolume(path string, size resource.Quantity, owner int64) *Volume {
	return &Volume{
		Path:  path,
		Size:  size,
		Owner: owner,
	}
}

func (c *Client) NewFile(source, dest string) *File {
	return &File{
		Source: source,
		Dest:   dest,
	}
}

func (c *Client) ReplacePodWithGracePeriod(ctx context.Context, podConfig PodConfig, gracePeriod *int64) (*v1.Pod, error) {
	logrus.Debugf("Replacing pod %s", podConfig.Name)

	if err := c.DeletePodWithGracePeriod(ctx, podConfig.Name, gracePeriod); err != nil {
		return nil, ErrDeletingPod.Wrap(err)
	}

	if err := c.waitForPodDeletion(ctx, podConfig.Name); err != nil {
		return nil, ErrWaitingForPodDeletion.WithParams(podConfig.Name).Wrap(err)
	}

	pod, err := c.DeployPod(ctx, podConfig, false)
	if err != nil {
		return nil, ErrDeployingPod.Wrap(err)
	}

	return pod, nil
}

func (c *Client) waitForPodDeletion(ctx context.Context, name string) error {
	for {
		select {
		case <-ctx.Done():
			logrus.Errorf("Context cancelled while waiting for pod %s to delete", name)
			return ctx.Err()
		case <-time.After(retryInterval):
			if _, err := c.getPod(ctx, name); err != nil {
				if apierrs.IsNotFound(err) {
					logrus.Debugf("Pod %s successfully deleted", name)
					return nil
				}
			}
		}
	}
}

// ReplacePod replaces a pod and returns the new Pod object.
func (c *Client) ReplacePod(ctx context.Context, podConfig PodConfig) (*v1.Pod, error) {
	return c.ReplacePodWithGracePeriod(ctx, podConfig, nil)
}

// IsPodRunning returns true if all containers in the pod are running.
func (c *Client) IsPodRunning(ctx context.Context, name string) (bool, error) {
	pod, err := c.getPod(ctx, name)
	if err != nil {
		return false, ErrGettingPod.WithParams(name).Wrap(err)
	}

	// Check if all container are running
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			return false, nil
		}
	}

	return true, nil
}

// RunCommandInPod runs a command in a container within a pod with a context.
func (c *Client) RunCommandInPod(
	ctx context.Context,
	podName,
	containerName string,
	cmd []string,
) (string, error) {
	_, err := c.getPod(ctx, podName)
	if err != nil {
		return "", ErrGettingPod.WithParams(podName).Wrap(err)
	}

	req := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(c.namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command:   cmd,
			Container: containerName,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	// Create an executor for the command execution
	k8sConfig, err := getClusterConfig()
	if err != nil {
		return "", ErrGettingK8sConfig.Wrap(err)
	}
	exec, err := remotecommand.NewSPDYExecutor(k8sConfig, http.MethodPost, req.URL())
	if err != nil {
		return "", ErrCreatingExecutor.Wrap(err)
	}

	// Execute the command and capture the output and error streams
	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})

	if err != nil {
		return "", ErrExecutingCommand.Wrap(err)
	}

	// Check if there were any errors on the error stream
	if stderr.Len() != 0 {
		return "", ErrCommandExecution.WithParams(stderr.String())
	}

	return stdout.String(), nil
}

func (c *Client) DeletePodWithGracePeriod(ctx context.Context, name string, gracePeriodSeconds *int64) error {
	if _, err := c.getPod(ctx, name); err != nil {
		// If the pod does not exist, skip and return without error
		if apierrs.IsNotFound(err) {
			return nil
		}
		return err
	}

	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: gracePeriodSeconds,
	}
	if err := c.clientset.CoreV1().Pods(c.namespace).Delete(ctx, name, deleteOptions); err != nil {
		return ErrDeletingPodFailed.WithParams(name).Wrap(err)
	}

	return nil
}

func (c *Client) DeletePod(ctx context.Context, name string) error {
	return c.DeletePodWithGracePeriod(ctx, name, nil)
}

// PortForwardPod forwards a local port to a port on a pod.
func (c *Client) PortForwardPod(
	ctx context.Context,
	podName string,
	localPort,
	remotePort int,
) error {
	if _, err := c.getPod(ctx, podName); err != nil {
		return ErrGettingPod.WithParams(podName).Wrap(err)
	}

	restConfig, err := getClusterConfig()
	if err != nil {
		return ErrGettingClusterConfig.Wrap(err)
	}

	url := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(c.namespace).
		Name(podName).
		SubResource("portforward").
		URL()

	transport, upgrader, err := spdy.RoundTripperFor(restConfig)
	if err != nil {
		return ErrCreatingRoundTripper.Wrap(err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)
	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}

	var (
		stopChan  = make(chan struct{}, 1)
		readyChan = make(chan struct{})
		errChan   = make(chan error)
	)

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	// Create a new PortForwarder
	pf, err := portforward.New(dialer, ports, stopChan, readyChan, stdout, stderr)
	if err != nil {
		return ErrCreatingPortForwarder.Wrap(err)
	}
	if stderr.Len() > 0 {
		return ErrPortForwarding.WithParams(stderr.String())
	}
	logrus.Debugf("Port forwarding from %d to %d", localPort, remotePort)
	logrus.Debugf("Port forwarding stdout: %v", stdout)

	// Start the port forwarding
	go func() {
		if err := pf.ForwardPorts(); err != nil {
			errChan <- err
			return
		}
		close(errChan) // if there's no error, close the channel
	}()

	// Wait for the port forwarding to be ready or error to occur
	select {
	case <-readyChan:
		// Ready to forward
		logrus.Debugf("Port forwarding ready from %d to %d", localPort, remotePort)
	case err := <-errChan:
		// if there's an error, return it
		return ErrForwardingPorts.Wrap(err)
	case <-time.After(waitRetry * 2):
		return ErrPortForwardingTimeout
	}

	return nil
}

func (c *Client) getPod(ctx context.Context, name string) (*v1.Pod, error) {
	return c.clientset.CoreV1().Pods(c.namespace).Get(ctx, name, metav1.GetOptions{})
}

// buildEnv builds an environment variable configuration for a Pod based on the given map of key-value pairs.
func buildEnv(envMap map[string]string) []v1.EnvVar {
	envVars := make([]v1.EnvVar, 0, len(envMap))
	for key, val := range envMap {
		envVar := v1.EnvVar{Name: key, Value: val}
		envVars = append(envVars, envVar)
	}
	return envVars
}

// buildPodVolumes generates a volume configuration for a pod based on the given name.
// If the volumes amount is zero, returns an empty slice.
func buildPodVolumes(name string, volumesAmount, filesAmount int) []v1.Volume {
	// return empty slice if no volumes or files are specified
	if volumesAmount == 0 && filesAmount == 0 {
		return []v1.Volume{}
	}

	var podVolumes []v1.Volume

	if volumesAmount != 0 {
		podVolume := v1.Volume{
			Name: name,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: name,
				},
			},
		}

		podVolumes = append(podVolumes, podVolume)
	}

	// 0777 is used so that the files are usable by any user in the container without needing to change permissions
	defaultMode := int32(0777)

	if filesAmount != 0 {
		podFiles := v1.Volume{
			Name: name + "-config",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: name,
					},
					DefaultMode: &defaultMode,
				},
			},
		}

		podVolumes = append(podVolumes, podFiles)
	}

	return podVolumes
}

// buildContainerVolumes generates a volume mount configuration for a container based on the given name and volumes.
func buildContainerVolumes(name string, volumes []*Volume) []v1.VolumeMount {
	var containerVolumes []v1.VolumeMount

	// return empty slice if no volumes or files are specified
	if len(volumes) == 0 {
		return containerVolumes
	}

	// iterate over the volumes map, add each volume to the containerVolumes
	for _, volume := range volumes {
		containerVolumes = append(containerVolumes, v1.VolumeMount{
			Name:      name,
			MountPath: volume.Path,
			SubPath:   strings.TrimLeft(volume.Path, "/"),
		})
	}

	return containerVolumes
}

// buildInitContainerVolumes generates a volume mount configuration for an init container based on the given name and volumes.
func buildInitContainerVolumes(name string, volumes []*Volume, files []*File) []v1.VolumeMount {
	if len(volumes) == 0 && len(files) == 0 {
		return []v1.VolumeMount{} // return empty slice if no volumes are specified
	}

	var containerFiles []v1.VolumeMount

	containerVolumes := []v1.VolumeMount{
		{
			Name:      name,
			MountPath: knuuPath, // set the path to "/knuu" as per the requirements
		},
	}

	if len(files) != 0 {
		// iterate over the files map, add each file to the containerFiles
		n := 0
		for _, file := range files {
			containerFiles = append(containerFiles, v1.VolumeMount{
				Name:      name + "-config",
				MountPath: file.Dest,
				SubPath:   fmt.Sprintf("%d", n),
			})
			n++
		}
	}

	return append(containerVolumes, containerFiles...)
}

// buildInitContainerCommand generates a command for an init container based on the given name and volumes.
func buildInitContainerCommand(volumes []*Volume, files []*File) []string {
	var commands = []string{"sh", "-c"}
	dirsProcessed := make(map[string]bool)
	baseCmd := "set -xe && "
	createKnuuPath := fmt.Sprintf("mkdir -p %s && ", knuuPath)
	cmds := []string{baseCmd, createKnuuPath}

	// for each file, get the directory and create the parent directory if it doesn't exist
	for _, file := range files {
		// get the directory of the file
		folder := filepath.Dir(file.Dest)
		if _, processed := dirsProcessed[folder]; !processed {
			knuuFolder := fmt.Sprintf("%s%s", knuuPath, folder)
			parentDirCmd := fmt.Sprintf("mkdir -p %s && ", knuuFolder)
			cmds = append(cmds, parentDirCmd)
			dirsProcessed[folder] = true
		}
		copyFileToKnuu := fmt.Sprintf("cp %s %s && ", file.Dest, filepath.Join(knuuPath, file.Dest))
		cmds = append(cmds, copyFileToKnuu)
	}

	// for each volume, copy the contents of the volume to the knuu volume
	for i, volume := range volumes {
		knuuVolumePath := fmt.Sprintf("%s%s", knuuPath, volume.Path)
		cmd := fmt.Sprintf("if [ -d %s ] && [ \"$(ls -A %s)\" ]; then mkdir -p %s && cp -r %s/* %s && chown -R %d:%d %s", volume.Path, volume.Path, knuuVolumePath, volume.Path, knuuVolumePath, volume.Owner, volume.Owner, knuuVolumePath)
		if i < len(volumes)-1 {
			cmd += " ;fi && "
		} else {
			cmd += " ;fi"
		}
		cmds = append(cmds, cmd)
	}

	fullCommand := strings.Join(cmds, "")
	commands = append(commands, fullCommand)

	logrus.Debugf("Init container command: %s", fullCommand)
	return commands
}

// buildResources generates a resource configuration for a container based on the given CPU and memory requests and limits.
func buildResources(memoryRequest, memoryLimit, cpuRequest resource.Quantity) v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceMemory: memoryRequest,
			v1.ResourceCPU:    cpuRequest,
		},
		Limits: v1.ResourceList{
			v1.ResourceMemory: memoryLimit,
		},
	}
}

// prepareContainer creates a v1.Container from a given ContainerConfig.
func prepareContainer(config ContainerConfig) v1.Container {
	// Build environment variables from the given map
	podEnv := buildEnv(config.Env)

	// Build container volumes from the given map
	containerVolumes := buildContainerVolumes(config.Name, config.Volumes)
	resources := buildResources(config.MemoryRequest, config.MemoryLimit, config.CPURequest)

	return v1.Container{
		Name:            config.Name,
		Image:           config.Image,
		Command:         config.Command,
		Args:            config.Args,
		Env:             podEnv,
		VolumeMounts:    containerVolumes,
		Resources:       resources,
		LivenessProbe:   config.LivenessProbe,
		ReadinessProbe:  config.ReadinessProbe,
		StartupProbe:    config.StartupProbe,
		SecurityContext: config.SecurityContext,
	}, nil
}

// prepareInitContainers creates a slice of v1.Container as init containers.
func prepareInitContainers(config ContainerConfig, init bool) ([]v1.Container, error) {
	if !init || len(config.Volumes) == 0 {
		return nil, nil
	}

	initContainerVolumes, err := buildInitContainerVolumes(config.Name, config.Volumes, config.Files)
	if err != nil {
		return nil, ErrBuildingInitContainerVolumes.Wrap(err)
	}
	initContainerCommand, err := buildInitContainerCommand(config.Volumes, config.Files)
	if err != nil {
		return nil, ErrBuildingInitContainerCommand.Wrap(err)
	}

	user := int64(0)

	return []v1.Container{
		{
			Name:  config.Name + "-init",
			Image: config.Image,
			SecurityContext: &v1.SecurityContext{
				RunAsUser: &user,
			},
			Command:      initContainerCommand,
			VolumeMounts: initContainerVolumes,
		},
	}, nil
}

// preparePodVolumes prepares pod volumes
func preparePodVolumes(config ContainerConfig) ([]v1.Volume, error) {
	podVolumes, err := buildPodVolumes(config.Name, len(config.Volumes), len(config.Files))
	if err != nil {
		return nil, ErrBuildingPodVolumes.Wrap(err)
	}

	return podVolumes, nil
}

func preparePodSpec(spec PodConfig, init bool) (v1.PodSpec, error) {
	var err error

	// Prepare security context
	securityContext := v1.PodSecurityContext{
		FSGroup: &spec.FsGroup,
	}

	// Prepare main container
	mainContainer, err := prepareContainer(spec.ContainerConfig)
	if err != nil {
		return v1.PodSpec{}, ErrPreparingMainContainer.Wrap(err)
	}

	// Prepare init containers
	initContainers, err := prepareInitContainers(spec.ContainerConfig, init)
	if err != nil {
		return v1.PodSpec{}, ErrPreparingInitContainer.Wrap(err)
	}

	// Prepare volumes
	podVolumes, err := preparePodVolumes(spec.ContainerConfig)
	if err != nil {
		return v1.PodSpec{}, ErrPreparingPodVolumes.Wrap(err)
	}

	podSpec := v1.PodSpec{
		ServiceAccountName: spec.ServiceAccountName,
		SecurityContext:    &securityContext,
		InitContainers:     initContainers,
		Containers:         []v1.Container{mainContainer},
		Volumes:            podVolumes,
	}

	// Prepare sidecar containers and append to the pod spec
	for _, sidecarConfig := range spec.SidecarConfigs {
		sidecar, err := prepareContainer(sidecarConfig)
		if err != nil {
			return v1.PodSpec{}, ErrPreparingSidecarContainer.Wrap(err)
		}

		sidecarVolumes, err := preparePodVolumes(sidecarConfig)
		if err != nil {
			return v1.PodSpec{}, ErrPreparingSidecarVolumes.Wrap(err)
		}

		podSpec.Containers = append(podSpec.Containers, sidecar)
		podSpec.Volumes = append(podSpec.Volumes, sidecarVolumes...)
	}

	return podSpec, nil
}

// preparePod prepares a pod configuration.
func preparePod(spec PodConfig, init bool) (*v1.Pod, error) {
	namespace := spec.Namespace
	name := spec.Name
	labels := spec.Labels

	podSpec, err := preparePodSpec(spec, init)
	if err != nil {
		return nil, ErrCreatingPodSpec.Wrap(err)
	}

	// Construct the Pod object using the above data
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name,
			Labels:      labels,
			Annotations: spec.Annotations,
		},
		Spec: podSpec,
	}

	logrus.Debugf("Prepared pod %s in namespace %s", name, namespace)

	return pod, nil
}
