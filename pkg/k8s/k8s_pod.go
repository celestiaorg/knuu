// Package k8s provides utility functions for working with Kubernetes pods.
package k8s

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// getPod retrieves a pod from the given namespace and logs any errors.
func getPod(namespace, name string) (*v1.Pod, error) {
	// Use context.Background() to generate an empty context.Context instance
	pod, err := Clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
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

	// Try to create the pod
	createdPod, err := Clientset.CoreV1().Pods(podConfig.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create pod: %v", err)
	}

	return createdPod, nil
}

// PodConfig contains the specifications for creating a new Pod object
type PodConfig struct {
	Namespace string            // Kubernetes namespace of the Pod
	Name      string            // Name to assign to the Pod
	Labels    map[string]string // Labels to apply to the Pod
	Image     string            // Name of the Docker image to use for the container
	Command   []string          // Command to run in the container
	Args      []string          // Arguments to pass to the command in the container
	Env       map[string]string // Environment variables to set in the container
	Volumes   map[string]string // Volumes to mount in the Pod
}

// ReplacePod replaces a pod in the given namespace and returns the new Pod object.
func ReplacePod(podConfig PodConfig) (*v1.Pod, error) {
	// Log a debug message to indicate that we are replacing a pod
	logrus.Debugf("Replacing pod %s", podConfig.Name)

	// Delete the existing pod (if any)
	if err := DeletePod(podConfig.Namespace, podConfig.Name); err != nil {
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

// WaitPodIsRunning waits until a pod in the given namespace is running.
func WaitPodIsRunning(namespace, name string) error {
	for {
		// Get the pod from Kubernetes API server
		pod, err := getPod(namespace, name)
		if err != nil { // Handle errors while getting the pod
			return fmt.Errorf("failed to get pod: %v", err)
		}

		// Check if the pod is running
		if pod.Status.Phase == v1.PodRunning {
			break
		}

		time.Sleep(100 * time.Millisecond) // Wait for 1 second before checking again (to avoid spamming API server)
	}

	return nil
}

// RunCommandInPod runs a command in a container within a pod.
func RunCommandInPod(namespace, podName, containerName string, cmd []string) (string, error) {
	// Get the pod object
	_, err := getPod(namespace, podName)
	if err != nil {
		return "", fmt.Errorf("failed to get pod: %v", err)
	}

	// Construct the request for executing the command in the specified container
	req := Clientset.CoreV1().RESTClient().Post().
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

	// Execute the command and capture the output and error streams
	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
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

// DeletePod deletes a pod with the given name in the specified namespace.
func DeletePod(namespace, name string) error {
	// Get the Pod object from the API server
	_, err := getPod(namespace, name)
	if err != nil {
		return fmt.Errorf("failed to get pod %s: %v", name, err)
	}

	// Delete the pod using the Kubernetes client API
	if err = Clientset.CoreV1().Pods(namespace).Delete(context.Background(), name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete pod %s: %v", name, err)
	}

	logrus.Debugf("Pod %s deleted in namespace %s", name, namespace)
	return nil
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
func buildPodVolumes(name string, volumesAmount int) ([]v1.Volume, error) {
	if volumesAmount == 0 {
		return []v1.Volume{}, nil
	}

	podVolume := v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: name,
			},
		},
	}

	return []v1.Volume{podVolume}, nil
}

// buildContainerVolumes generates a volume mount configuration for a container based on the given name and volumes.
func buildContainerVolumes(name string, volumes map[string]string) ([]v1.VolumeMount, error) {
	var containerVolumes []v1.VolumeMount

	if len(volumes) == 0 {
		return containerVolumes, nil // return empty slice if no volumes are specified
	}

	// iterate over the volumes map, add each volume to the containerVolumes
	for path, _ := range volumes {
		containerVolumes = append(containerVolumes, v1.VolumeMount{
			Name:      name,
			MountPath: path,
			SubPath:   strings.TrimLeft(path, "/"),
		})
	}

	return containerVolumes, nil
}

// buildInitContainerVolumes generates a volume mount configuration for an init container based on the given name and volumes.
func buildInitContainerVolumes(name string, volumes map[string]string) ([]v1.VolumeMount, error) {
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
func buildInitContainerCommand(name string, volumes map[string]string) ([]string, error) {
	if len(volumes) == 0 {
		return []string{}, nil // return empty slice if no volumes are specified
	}

	var command []string = []string{"sh", "-c"} // initialize the command slice with the required shell interpreter
	for path := range volumes {                 // use _ as the blank identifier since we're not using the value of the map element
		cmd := fmt.Sprintf("mkdir -p /knuu/%s && cp -r %s/* /knuu/%s", path, path, path)
		command = append(command, cmd) // add each command to the command slice
	}

	return command, nil
}

// preparePod prepares a pod configuration.
func preparePod(spec PodConfig, init bool) (*v1.Pod, error) {
	namespace := spec.Namespace
	name := spec.Name
	labels := spec.Labels
	image := spec.Image
	command := spec.Command
	args := spec.Args
	env := spec.Env
	volumes := spec.Volumes

	// Build environment variables from the given map
	podEnv := buildEnv(env)

	// Build pod volumes from the given map
	podVolumes, err := buildPodVolumes(name, len(volumes))
	if err != nil {
		return nil, fmt.Errorf("failed to create pod volumes: %w", err)
	}

	// Build container volumes from the given map
	containerVolumes, err := buildContainerVolumes(name, volumes)
	if err != nil {
		return nil, fmt.Errorf("failed to create container volumes: %w", err)
	}

	var initContainers []v1.Container
	if len(volumes) > 0 && init {
		// Build init containers volumes and command from the given map
		initContainerVolumes, err := buildInitContainerVolumes(name, volumes)
		if err != nil {
			return nil, fmt.Errorf("failed to create init container volumes: %w", err)
		}
		initContainerCommand, err := buildInitContainerCommand(name, volumes)
		if err != nil {
			return nil, fmt.Errorf("failed to create init container command: %w", err)
		}

		initContainers = []v1.Container{
			{
				Name:         "volume-whatever",
				Image:        image,
				Command:      initContainerCommand,
				VolumeMounts: initContainerVolumes,
			},
		}
	}

	// Construct the Pod object using the above data
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: v1.PodSpec{
			InitContainers: initContainers,
			Containers: []v1.Container{
				{
					Name:         name,
					Image:        image,
					Command:      command,
					Args:         args,
					Env:          podEnv,
					VolumeMounts: containerVolumes,
				},
			},
			Volumes: podVolumes,
		},
	}

	logrus.Debugf("Prepared pod %s in namespace %s", name, namespace)

	return pod, nil
}
