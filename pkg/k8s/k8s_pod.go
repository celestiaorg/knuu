package k8s

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
)

// getPod retrieves a pod from the given namespace and logs any errors.
func getPod(namespace, name string) (*v1.Pod, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	pod, err := Clientset().CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s: %w", name, err)
	}

	return pod, nil
}

// DeployPod creates a new pod in the given namespace if it doesn't already exist.
func DeployPod(podConfig PodConfig, init bool) (*v1.Pod, error) {
	// Prepare the pod
	pod, err := preparePod(podConfig, init)
	if err != nil {
		return nil, fmt.Errorf("error preparing pod: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Try to create the pod
	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	createdPod, err := Clientset().CoreV1().Pods(podConfig.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create pod: %v", err)
	}

	return createdPod, nil
}

// Volume represents a volume.
type Volume struct {
	Path  string
	Size  string
	Owner int64
}

// File represents a file.
type File struct {
	Source string
	Dest   string
}

// NewVolume creates a new volume with the given path, size and owner.
func NewVolume(path, size string, owner int64) *Volume {
	return &Volume{
		Path:  path,
		Size:  size,
		Owner: owner,
	}
}

// NewFile creates a new file with the given source and destination.
func NewFile(source, dest string) *File {
	return &File{
		Source: source,
		Dest:   dest,
	}
}

// ContainerConfig contains the specifications for creating a new Container object
type ContainerConfig struct {
	Name           string            // Name to assign to the Container
	Image          string            // Name of the container image to use for the container
	Command        []string          // Command to run in the container
	Args           []string          // Arguments to pass to the command in the container
	Env            map[string]string // Environment variables to set in the container
	Volumes        []*Volume         // Volumes to mount in the Pod
	MemoryRequest  string            // Memory request for the container
	MemoryLimit    string            // Memory limit for the container
	CPURequest     string            // CPU request for the container
	LivenessProbe  *v1.Probe         // Liveness probe for the container
	ReadinessProbe *v1.Probe         // Readiness probe for the container
	StartupProbe   *v1.Probe         // Startup probe for the container
	Files          []*File           // Files to add to the Pod
}

// PodConfig contains the specifications for creating a new Pod object
type PodConfig struct {
	Namespace          string            // Kubernetes namespace of the Pod
	Name               string            // Name to assign to the Pod
	Labels             map[string]string // Labels to apply to the Pod
	ServiceAccountName string            // ServiceAccount to assign to Pod
	FsGroup            int64             // FSGroup to apply to the Pod
	ContainerConfig    ContainerConfig   // ContainerConfig for the Pod
	SidecarConfigs     []ContainerConfig // SideCarConfigs for the Pod
}

// ReplacePodWithGracePeriod replaces a pod in the given namespace and returns the new Pod object with a grace period.
func ReplacePodWithGracePeriod(podConfig PodConfig, gracePeriod *int64) (*v1.Pod, error) {
	// Log a debug message to indicate that we are replacing a pod
	logrus.Debugf("Replacing pod %s", podConfig.Name)

	// Delete the existing pod (if any)
	if err := DeletePodWithGracePeriod(podConfig.Namespace, podConfig.Name, gracePeriod); err != nil {
		return nil, fmt.Errorf("failed to delete pod: %v", err)
	}

	// Wait for the pod to be fully deleted
	for {
		_, err := getPod(podConfig.Namespace, podConfig.Name)
		if err != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Deploy the new pod
	pod, err := DeployPod(podConfig, false)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy pod: %v", err)
	}

	// Return the newly created pod
	return pod, nil
}

// ReplacePod replaces a pod in the given namespace and returns the new Pod object.
func ReplacePod(podConfig PodConfig) (*v1.Pod, error) {
	return ReplacePodWithGracePeriod(podConfig, nil)
}

// IsPodRunning returns true if all containers in the pod are running.
func IsPodRunning(namespace, name string) (bool, error) {
	// Get the pod from Kubernetes API server
	pod, err := getPod(namespace, name)
	if err != nil {
		return false, fmt.Errorf("failed to get pod: %v", err)
	}

	// Check if all container are running
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			return false, nil
		}
	}

	return true, nil
}

// RunCommandInPod runs a command in a container within a pod.
func RunCommandInPod(
	namespace,
	podName,
	containerName string,
	cmd []string,
) (string, error) {
	// Get the pod object
	_, err := getPod(namespace, podName)
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %v", err)
	}

	// Construct the request for executing the command in the specified container
	if !IsInitialized() {
		return "", fmt.Errorf("knuu is not initialized")
	}
	req := Clientset().CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
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
		return "", fmt.Errorf("failed to get k8s config: %v", err)
	}
	exec, err := remotecommand.NewSPDYExecutor(k8sConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to create Executor: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute the command and capture the output and error streams
	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %v", err)
	}

	// Check if there were any errors on the error stream
	if stderr.Len() != 0 {
		return "", fmt.Errorf("error while executing command: %s", stderr.String())
	}

	return stdout.String(), nil
}

// DeletePodWithGracePeriod deletes a pod with the given name in the specified namespace.
func DeletePodWithGracePeriod(namespace, name string, gracePeriodSeconds *int64) error {
	// Get the Pod object from the API server
	_, err := getPod(namespace, name)
	if err != nil {
		// If the pod does not exist, skip and return without error
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Delete the pod using the Kubernetes client API
	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: gracePeriodSeconds,
	}
	if err := Clientset().CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions); err != nil {
		return fmt.Errorf("failed to delete pod %s: %v", name, err)
	}

	return nil
}

// DeletePod deletes a pod with the given name in the specified namespace.
func DeletePod(namespace, name string) error {
	return DeletePodWithGracePeriod(namespace, name, nil)
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
func buildPodVolumes(name string, volumesAmount, filesAmount int) ([]v1.Volume, error) {
	// return empty slice if no volumes or files are specified
	if volumesAmount == 0 && filesAmount == 0 {
		return []v1.Volume{}, nil
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

	return podVolumes, nil
}

// buildContainerVolumes generates a volume mount configuration for a container based on the given name and volumes.
func buildContainerVolumes(name string, volumes []*Volume, files []*File) ([]v1.VolumeMount, error) {
	var containerVolumes []v1.VolumeMount
	var containerFiles []v1.VolumeMount

	// return empty slice if no volumes or files are specified
	if len(volumes) == 0 && len(files) == 0 {
		return containerVolumes, nil
	}

	if len(volumes) != 0 {
		// iterate over the volumes map, add each volume to the containerVolumes
		for _, volume := range volumes {
			containerVolumes = append(containerVolumes, v1.VolumeMount{
				Name:      name,
				MountPath: volume.Path,
				SubPath:   strings.TrimLeft(volume.Path, "/"),
			})
		}
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

	return append(containerVolumes, containerFiles...), nil
}

// buildInitContainerVolumes generates a volume mount configuration for an init container based on the given name and volumes.
func buildInitContainerVolumes(name string, volumes []*Volume) ([]v1.VolumeMount, error) {
	if len(volumes) == 0 {
		return []v1.VolumeMount{}, nil // return empty slice if no volumes are specified
	}

	containerVolumes := []v1.VolumeMount{
		{
			Name:      name,
			MountPath: "/knuu", // set the path to "/knuu" as per the requirements
		},
	}

	return containerVolumes, nil
}

// buildInitContainerCommand generates a command for an init container based on the given name and volumes.
func buildInitContainerCommand(name string, volumes []*Volume) ([]string, error) {
	if len(volumes) == 0 {
		return []string{}, nil // return empty slice if no volumes are specified
	}

	var command = []string{"sh", "-c"} // initialize the command slice with the required shell interpreter
	for _, volume := range volumes {
		cmd := fmt.Sprintf("mkdir -p /knuu/%s && cp -r %s/* /knuu/%s && chown -R %d:%d /knuu/*", volume.Path, volume.Path, volume.Path, volume.Owner, volume.Owner)
		command = append(command, cmd) // add each command to the command slice
	}

	return command, nil
}

// buildResources generates a resource configuration for a container based on the given CPU and memory requests and limits.
func buildResources(memoryRequest string, memoryLimit string, cpuRequest string) (v1.ResourceRequirements, error) {
	resources := v1.ResourceRequirements{}

	memoryRequestQuantity, err := resource.ParseQuantity(memoryRequest)
	if err != nil {
		if memoryRequest != "" {
			return resources, fmt.Errorf("failed to parse memory request quantity '%s': %v", memoryRequest, err)
		}
	}
	memoryLimitQuantity, err := resource.ParseQuantity(memoryLimit)
	if err != nil {
		if memoryLimit != "" {
			return resources, fmt.Errorf("failed to parse memory limit quantity '%s': %v", memoryLimit, err)
		}
	}
	cpuRequestQuantity, err := resource.ParseQuantity(cpuRequest)
	if err != nil {
		if cpuRequest != "" {
			return resources, fmt.Errorf("failed to parse CPU request quantity '%s': %v", cpuRequest, err)
		}
	}

	// If a resource is not set it will use the default value of 0 which is the same as not setting it at all.
	resources = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceMemory: memoryRequestQuantity,
			v1.ResourceCPU:    cpuRequestQuantity,
		},
		Limits: v1.ResourceList{
			v1.ResourceMemory: memoryLimitQuantity,
		},
	}

	return resources, nil
}

// prepareContainer creates a v1.Container from a given ContainerConfig.
func prepareContainer(config ContainerConfig) (v1.Container, error) {
	// Build environment variables from the given map
	podEnv := buildEnv(config.Env)

	// Build container volumes from the given map
	containerVolumes, err := buildContainerVolumes(config.Name, config.Volumes, config.Files)
	if err != nil {
		return v1.Container{}, fmt.Errorf("failed to build container volumes: %v", err)
	}

	resources, err := buildResources(config.MemoryRequest, config.MemoryLimit, config.CPURequest)
	if err != nil {
		return v1.Container{}, fmt.Errorf("failed to build resources: %v", err)
	}

	return v1.Container{
		Name:           config.Name,
		Image:          config.Image,
		Command:        config.Command,
		Args:           config.Args,
		Env:            podEnv,
		VolumeMounts:   containerVolumes,
		Resources:      resources,
		LivenessProbe:  config.LivenessProbe,
		ReadinessProbe: config.ReadinessProbe,
		StartupProbe:   config.StartupProbe,
	}, nil
}

// prepareInitContainers creates a slice of v1.Container as init containers.
func prepareInitContainers(config ContainerConfig, init bool) ([]v1.Container, error) {
	if !init || len(config.Volumes) == 0 {
		return nil, nil
	}

	initContainerVolumes, err := buildInitContainerVolumes(config.Name, config.Volumes)
	if err != nil {
		return nil, fmt.Errorf("failed to build init container volumes: %v", err)
	}
	initContainerCommand, err := buildInitContainerCommand(config.Name, config.Volumes)
	if err != nil {
		return nil, fmt.Errorf("failed to build init container command: %v", err)
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
		return nil, fmt.Errorf("failed to build pod volumes: %v", err)
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
		return v1.PodSpec{}, fmt.Errorf("failed to prepare main container: %w", err)
	}

	// Prepare init containers
	initContainers, err := prepareInitContainers(spec.ContainerConfig, init)
	if err != nil {
		return v1.PodSpec{}, fmt.Errorf("failed to prepare init containers: %w", err)
	}

	// Prepare volumes
	podVolumes, err := preparePodVolumes(spec.ContainerConfig)
	if err != nil {
		return v1.PodSpec{}, fmt.Errorf("failed to prepare pod volumes: %w", err)
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
			return v1.PodSpec{}, fmt.Errorf("failed to prepare sidecar container: %w", err)
		}

		sidecarVolumes, err := preparePodVolumes(sidecarConfig)
		if err != nil {
			return v1.PodSpec{}, fmt.Errorf("failed to prepare sidecar volumes: %w", err)
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
		return nil, fmt.Errorf("failed to create pod spec: %w", err)
	}

	// Construct the Pod object using the above data
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: podSpec,
	}

	logrus.Debugf("Prepared pod %s in namespace %s", name, namespace)

	return pod, nil
}

// PortForwardPod forwards a local port to a port on a pod.
func PortForwardPod(
	namespace,
	podName string,
	localPort,
	remotePort int,
) error {
	// Get the pod object
	_, err := getPod(namespace, podName)
	if err != nil {
		return fmt.Errorf("failed to get pod: %v", err)
	}

	// Get a config to talk to the apiserver
	restconfig, err := getClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get cluster config: %v", err)
	}

	// Setup the port forwarding
	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	url := Clientset().CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward").
		URL()

	transport, upgrader, err := spdy.RoundTripperFor(restconfig)
	if err != nil {
		return fmt.Errorf("failed to create round tripper: %v", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", url)

	ports := []string{fmt.Sprintf("%d:%d", localPort, remotePort)}

	stopChan := make(chan struct{}, 1)
	readyChan := make(chan struct{})

	var stdout, stderr io.Writer
	// Create a new PortForwarder
	pf, err := portforward.New(dialer, ports, stopChan, readyChan, stdout, stderr)
	if err != nil {
		return fmt.Errorf("failed to create port forwarder: %v", err)
	}
	if stderr != nil {
		return fmt.Errorf("failed to port forward: %v", stderr)
	}
	logrus.Debugf("Port forwarding from %d to %d", localPort, remotePort)
	logrus.Debugf("Port forwarding stdout: %v", stdout)

	errChan := make(chan error)

	// Start the port forwarding
	go func() {
		if err := pf.ForwardPorts(); err != nil {
			errChan <- err
		} else {
			close(errChan) // if there's no error, close the channel
		}
	}()

	// Wait for the port forwarding to be ready or error to occur
	select {
	case <-readyChan:
		// Ready to forward
		logrus.Debugf("Port forwarding ready from %d to %d", localPort, remotePort)
	case err := <-errChan:
		// if there's an error, return it
		return fmt.Errorf("error forwarding ports: %w", err)
	case <-time.After(time.Second * 5):
		return fmt.Errorf("timed out waiting for port forwarding to be ready")
	}

	return nil
}
