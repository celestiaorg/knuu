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
	"k8s.io/utils/ptr"
)

// the loops that keep checking something and wait for it to be done
const (
	// knuuPath is the path where the knuu volume is mounted
	knuuPath = "/knuu"

	podFilesConfigmapNameSuffix = "-config"

	initContainerNameSuffix = "-init"
	initContainerImage      = "nicolaka/netshoot"
	defaultContainerUser    = 0
)

type ContainerConfig struct {
	Name            string              // Name to assign to the Container
	Image           string              // Name of the container image to use for the container
	InitImageName   string              // InitImageName to use for the Pod
	ImagePullPolicy v1.PullPolicy       // Image pull policy for the container
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
	TCPPorts        []int               // TCP ports to expose on the Pod
	UDPPorts        []int               // UDP ports to expose on the Pod
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
	NodeSelector       map[string]string // NodeSelector to apply to the Pod
}

// DeployPod creates a new pod in the namespace that k8s client is initiate with if it doesn't already exist.
func (c *Client) DeployPod(ctx context.Context, podConfig PodConfig, init bool) (*v1.Pod, error) {
	if c.terminated {
		return nil, ErrClientTerminated
	}
	if err := validatePodConfig(podConfig); err != nil {
		return nil, err
	}

	pod := c.preparePod(podConfig, init)
	createdPod, err := c.clientset.CoreV1().Pods(c.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, ErrCreatingPod.Wrap(err)
	}

	return createdPod, nil
}

func (c *Client) ReplacePodWithGracePeriod(ctx context.Context, podConfig PodConfig, gracePeriod *int64) (*v1.Pod, error) {
	c.logger.WithField("name", podConfig.Name).Debug("replacing pod")

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
			c.logger.WithField("name", name).Error("context cancelled while waiting for pod to delete")
			return ctx.Err()
		case <-time.After(retryInterval):
			_, err := c.getPod(ctx, name)
			if err != nil {
				if apierrs.IsNotFound(err) {
					c.logger.WithField("name", name).Debug("pod successfully deleted")
					return nil
				}
				return ErrWaitingForPodDeletion.WithParams(name).Wrap(err)
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
	if err := validatePodName(podName); err != nil {
		return "", err
	}
	if err := validateContainerName(containerName); err != nil {
		return "", err
	}
	if err := validateCommand(cmd); err != nil {
		return "", err
	}

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
		return "", ErrExecutingCommand.WithParams(stdout.String(), stderr.String()).Wrap(err)
	}

	// Check if there were any errors on the error stream
	if stderr.Len() != 0 {
		return "", ErrCommandExecution.WithParams(stdout.String(), stderr.String())
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
	if err := validatePodName(podName); err != nil {
		return err
	}
	if err := validatePort(localPort); err != nil {
		return err
	}
	if err := validatePort(remotePort); err != nil {
		return err
	}

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
	c.logger.WithFields(logrus.Fields{
		"local_port":  localPort,
		"remote_port": remotePort,
		"stdout":      stdout.String(),
	}).Debug("port forwarding")

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
		c.logger.WithFields(logrus.Fields{
			"local_port":  localPort,
			"remote_port": remotePort,
		}).Debug("port forwarding ready")
	case err := <-errChan:
		// if there's an error, return it
		return ErrForwardingPorts.Wrap(err)
	case <-time.After(waitRetry * 2):
		return ErrPortForwardingTimeout
	}

	return nil
}

func (c *Client) getPod(ctx context.Context, name string) (*v1.Pod, error) {
	if c.terminated {
		return nil, ErrClientTerminated
	}
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
func buildPodVolumes(name string, volumes []*Volume, files []*File) []v1.Volume {
	var podVolumes []v1.Volume

	if len(volumes) != 0 {
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

	if len(files) == 0 {
		return podVolumes
	}

	uniqueDirs := uniqueDirs(files)
	for dir := range uniqueDirs {
		cmName := PrepareConfigMapName(name, dir)
		podFiles := v1.Volume{
			Name: cmName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: cmName,
					},
					DefaultMode: ptr.To[int32](0644),
				},
			},
		}
		podVolumes = append(podVolumes, podFiles)
	}

	return podVolumes
}

// buildContainerVolumes generates a volume mount configuration for a container based on the given name and volumes.
func buildContainerVolumes(name string, volumes []*Volume, files []*File) []v1.VolumeMount {
	var containerVolumes []v1.VolumeMount
	for _, volume := range volumes {
		containerVolumes = append(containerVolumes, v1.VolumeMount{
			Name:      name,
			MountPath: volume.Path,
		})
	}

	if len(files) == 0 {
		return containerVolumes
	}

	mountedDirs := make(map[string]bool)
	for _, file := range files {
		dir := filepath.Dir(file.Dest)
		cmName := PrepareConfigMapName(name, dir)

		// Since k8s is not allowed to mount a configmap to a critical dir (throws readonly file system error),
		// we need to mount the configmap to the file
		if isCriticalDir(dir) {
			containerVolumes = append(containerVolumes, v1.VolumeMount{
				Name:      cmName,
				MountPath: file.Dest,
				SubPath:   filepath.Base(file.Dest),
			})
			continue
		}

		// if the dir is not in a critical dir, we need to mount the configmap to the dir
		if _, processed := mountedDirs[dir]; processed {
			continue
		}

		containerVolumes = append(containerVolumes, v1.VolumeMount{
			Name:      cmName,
			MountPath: dir,
		})
		mountedDirs[dir] = true
	}

	return containerVolumes
}

// buildInitContainerVolumes generates a volume mount configuration for an init container based on the given name and volumes.
func buildInitContainerVolumes(name string, volumes []*Volume, files []*File) []v1.VolumeMount {
	if len(volumes) == 0 && len(files) == 0 {
		return []v1.VolumeMount{} // return empty slice if no volumes are specified
	}

	var containerVolumes []v1.VolumeMount
	// if the user want do add volumes, we need to mount the knuu path
	if len(volumes) != 0 {
		containerVolumes = append(containerVolumes, v1.VolumeMount{
			Name:      name,
			MountPath: knuuPath,
		})
	}
	// if the user don't want to add volumes, but want to add files, we need to mount the knuu path for the init container
	if len(volumes) == 0 && len(files) != 0 {
		uniquePaths := make(map[string]bool)
		for _, file := range files {
			uniquePaths[filepath.Dir(file.Dest)] = true
		}
		for path := range uniquePaths {
			containerVolumes = append(containerVolumes, v1.VolumeMount{
				Name:      name,
				MountPath: filepath.Join(knuuPath, path),
				SubPath:   filepath.Base(path),
			})
		}
	}

	var containerFiles []v1.VolumeMount
	for _, file := range files {
		containerFiles = append(containerFiles, v1.VolumeMount{
			Name:      name + podFilesConfigmapNameSuffix,
			MountPath: file.Dest,
			SubPath:   filepath.Base(file.Dest),
		})
	}

	return append(containerVolumes, containerFiles...)
}

// buildInitContainerCommand generates a command for an init container based on the given name and volumes.
func (c *Client) buildInitContainerCommand(volumes []*Volume, files []*File) []string {
	var (
		commands       = []string{"sh", "-c"}
		dirsProcessed  = make(map[string]bool)
		baseCmd        = "set -xe && "
		createKnuuPath = fmt.Sprintf("mkdir -p %s && ", knuuPath)
		cmds           = []string{baseCmd, createKnuuPath}
	)

	// for each file, get the directory and create the parent directory if it doesn't exist
	for _, file := range files {
		// get the directory of the file
		folder := filepath.Dir(file.Dest)
		if _, processed := dirsProcessed[folder]; !processed {
			var (
				knuuFolder   = filepath.Join(knuuPath, folder)
				parentDirCmd = fmt.Sprintf("mkdir -p %s && ", knuuFolder)
			)
			cmds = append(cmds, parentDirCmd)
			dirsProcessed[folder] = true
		}
		chown := file.Chown
		permission := file.Permission
		addFileToKnuu := fmt.Sprintf("cp %s %s && ", file.Dest, filepath.Join(knuuPath, file.Dest))
		if chown != "" {
			addFileToKnuu += fmt.Sprintf("chown %s %s && ", chown, filepath.Join(knuuPath, file.Dest))
		}
		if permission != "" {
			addFileToKnuu += fmt.Sprintf("chmod %s %s && ", permission, filepath.Join(knuuPath, file.Dest))
		}
		cmds = append(cmds, addFileToKnuu)
	}

	// for each volume, copy the contents of the volume to the knuu volume
	// TODO: this code works only for one volume, need to fix it
	for _, volume := range volumes {
		knuuVolumePath := knuuPath // volume is mounted to the same path so no need to join the path here
		cmd := fmt.Sprintf("if [ -d %s ] && [ \"$(ls -A %s)\" ]; then mkdir -p %s && cp -r %s/* %s && chown -R %d:%d %s",
			volume.Path, volume.Path, knuuVolumePath, volume.Path,
			knuuVolumePath, volume.Owner, volume.Owner, knuuVolumePath)
		cmd += " ;fi && "
		cmds = append(cmds, cmd)
	}

	fullCommand := strings.Join(cmds, "")
	commands = append(commands, fullCommand)
	if strings.HasSuffix(fullCommand, " && ") {
		commands[len(commands)-1] = strings.TrimSuffix(commands[len(commands)-1], " && ")
	}

	c.logger.WithField("command", fullCommand).Debug("init container command")
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

func buildPodPorts(tcpPorts, udpPorts []int) []v1.ContainerPort {
	ports := make([]v1.ContainerPort, 0, len(tcpPorts)+len(udpPorts))
	for _, port := range tcpPorts {
		ports = append(ports, v1.ContainerPort{
			Name:          fmt.Sprintf("tcp-%d", port),
			Protocol:      v1.ProtocolTCP,
			ContainerPort: int32(port),
		})
	}
	for _, port := range udpPorts {
		ports = append(ports, v1.ContainerPort{
			Name:          fmt.Sprintf("udp-%d", port),
			Protocol:      v1.ProtocolUDP,
			ContainerPort: int32(port),
		})
	}
	return ports
}

// prepareContainer creates a v1.Container from a given ContainerConfig.
func prepareContainer(config ContainerConfig) v1.Container {
	return v1.Container{
		Name:            config.Name,
		Image:           config.Image,
		ImagePullPolicy: config.ImagePullPolicy,
		Command:         config.Command,
		Args:            config.Args,
		Env:             buildEnv(config.Env),
		VolumeMounts:    buildContainerVolumes(config.Name, config.Volumes, config.Files),
		Resources:       buildResources(config.MemoryRequest, config.MemoryLimit, config.CPURequest),
		Ports:           buildPodPorts(config.TCPPorts, config.UDPPorts),
		LivenessProbe:   config.LivenessProbe,
		ReadinessProbe:  config.ReadinessProbe,
		StartupProbe:    config.StartupProbe,
		SecurityContext: config.SecurityContext,
	}
}

// prepareInitContainers creates a slice of v1.Container as init containers.
func (c *Client) prepareInitContainers(config ContainerConfig, init bool) []v1.Container {
	if !init ||
		(len(config.Volumes) == 0 && len(config.Files) == 0) ||
		config.InitImageName == "" {
		return nil
	}

	return []v1.Container{
		{
			Name:  config.Name + initContainerNameSuffix,
			Image: config.InitImageName,
			SecurityContext: &v1.SecurityContext{
				RunAsUser: ptr.To[int64](defaultContainerUser),
			},
			Command:      c.buildInitContainerCommand(config.Volumes, config.Files),
			VolumeMounts: buildInitContainerVolumes(config.Name, config.Volumes, config.Files),
		},
	}
}

// preparePodVolumes prepares pod volumes
func preparePodVolumes(config ContainerConfig) []v1.Volume {
	return buildPodVolumes(config.Name, config.Volumes, config.Files)
}

func (c *Client) preparePodSpec(spec PodConfig, init bool) v1.PodSpec {
	podSpec := v1.PodSpec{
		ServiceAccountName: spec.ServiceAccountName,
		InitContainers:     c.prepareInitContainers(spec.ContainerConfig, init),
		Containers:         []v1.Container{prepareContainer(spec.ContainerConfig)},
		Volumes:            preparePodVolumes(spec.ContainerConfig),
		NodeSelector:       spec.NodeSelector,
	}

	// Prepare sidecar containers and append to the pod spec
	for _, sidecarConfig := range spec.SidecarConfigs {
		sidecarInitContainer := c.prepareInitContainers(sidecarConfig, true)
		sidecarContainer := prepareContainer(sidecarConfig)
		sidecarVolumes := preparePodVolumes(sidecarConfig)

		podSpec.InitContainers = append(podSpec.InitContainers, sidecarInitContainer...)
		podSpec.Containers = append(podSpec.Containers, sidecarContainer)
		podSpec.Volumes = append(podSpec.Volumes, sidecarVolumes...)
	}

	return podSpec
}

func (c *Client) preparePod(spec PodConfig, init bool) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   spec.Namespace,
			Name:        spec.Name,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
		Spec: c.preparePodSpec(spec, init),
	}

	c.logger.WithFields(logrus.Fields{
		"name":      spec.Name,
		"namespace": spec.Namespace,
	}).Debug("prepared pod")
	return pod
}

func uniqueDirs(files []*File) map[string]bool {
	uniqueDirs := make(map[string]bool)
	for _, file := range files {
		uniqueDirs[filepath.Dir(file.Dest)] = true
	}
	return uniqueDirs
}

func isCriticalDir(dir string) bool {
	criticalDirs := map[string]bool{
		"/etc":   true,
		"/bin":   true,
		"/sbin":  true,
		"/lib":   true,
		"/lib64": true,
		"/dev":   true,
		"/proc":  true,
		"/sys":   true,
		"/run":   true,
		"/boot":  true,
		"/usr":   true,
		"/var":   true,
		"/root":  true,
	}
	return criticalDirs[dir]
}
