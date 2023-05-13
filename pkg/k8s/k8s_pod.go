// Package k8s provides utility functions for working with Kubernetes pods.
package k8s

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

// PodExists checks if a pod exists in the given namespace.
func PodExists(namespace, name string) bool {
	return getPod(namespace, name) != nil
}

// getPod retrieves a pod from the given namespace.
func getPod(namespace, name string) *v1.Pod {
	pod, err := Clientset.CoreV1().Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		logrus.Debug("Retrieving pod failed %s: %v", name, err)
		return nil
	}
	return pod
}

// DeployPod creates a new pod in the given namespace if it doesn't already exist.
func DeployPod(namespace, name string, labels map[string]string, image string, command, args []string, env, volumes map[string]string, init bool) *v1.Pod {
	if PodExists(namespace, name) {
		logrus.Debugf("Pod %s already exists, skipping...", name)
		return getPod(namespace, name)
	}

	pod := preparePod(namespace, name, labels, image, command, args, env, volumes, init)
	pod, err := Clientset.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		logrus.Fatalf("Creating pod %s: %v", name, err)
	}

	return pod
}

// ReplacePod replaces a pod in the given namespace.
func ReplacePod(namespace, name string, labels map[string]string, image string, command, args []string, env, volumes map[string]string) {
	logrus.Debugf("Replacing pod %s", name)

	DeletePod(namespace, name)
	for PodExists(namespace, name) {
		// Wait until the pod is deleted.
	}
	DeployPod(namespace, name, labels, image, command, args, env, volumes, false)
}

// WaitPodIsRunning waits until a pod in the given namespace is running.
func WaitPodIsRunning(namespace, name string) {
	for {
		pod := getPod(namespace, name)
		if pod.Status.Phase == v1.PodRunning {
			break
		}
	}
}

// RunCommandInPod runs a command in a container within a pod.
func RunCommandInPod(namespace, podName, containerName string, cmd []string) (string, error) {
	pod := getPod(namespace, podName)
	if pod == nil {
		return "", fmt.Errorf("could not find pod %s", podName)
	}

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

	k8sConfig, err := getClusterConfig()
	if err != nil {
		return "", fmt.Errorf("Error getting k8s config: %v", err)
	}
	exec, err := remotecommand.NewSPDYExecutor(k8sConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("Error while creating Executor: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("Error in Stream: %v", err)
	}

	if stderr.Len() != 0 {
		return "", fmt.Errorf("Error: %s", stderr.String())
	}

	return stdout.String(), nil
}

// DeletePod deletes a pod in the given namespace.
func DeletePod(namespace, name string) error {
	if !PodExists(namespace, name) {
		logrus.Debugf("Pod %s does not exist, skipping...", name)
		return nil
	}

	deleteOptions := metav1.DeleteOptions{}

	err := Clientset.CoreV1().Pods(namespace).Delete(context.Background(), name, deleteOptions)
	if err != nil {
		return fmt.Errorf("Deleting pod %s: %v", name, err)
	}
	return nil
}

// buildEnv builds the environment variable configuration for a pod.
func buildEnv(env map[string]string) []v1.EnvVar {
	var envVars []v1.EnvVar
	for k, v := range env {
		envVars = append(envVars, v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}
	return envVars
}

// buildPodVolumes builds the volume configuration for a pod.
func buildPodVolumes(name string, volumes map[string]string) []v1.Volume {
	if len(volumes) != 0 {
		podVolumes := []v1.Volume{
			{
				Name: name,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: name,
					},
				},
			},
		}
		return podVolumes
	}
	return []v1.Volume{}
}

// buildContainerVolumes builds the volume mount configuration for a container.
func buildContainerVolumes(name string, volumes map[string]string) []v1.VolumeMount {
	var containerVolumes []v1.VolumeMount
	for path, _ := range volumes {
		containerVolumes = append(containerVolumes, v1.VolumeMount{
			Name:      name,
			MountPath: path,
			SubPath:   strings.TrimLeft(path, "/"),
		})
	}
	return containerVolumes
}

// buildInitContainerVolumes builds the volume mount configuration for an init container.
func buildInitContainerVolumes(name string, volumes map[string]string) []v1.VolumeMount {
	if len(volumes) != 0 {
		containerVolumes := []v1.VolumeMount{
			{
				Name:      name,
				MountPath: "/knuu",
			},
		}
		return containerVolumes
	}
	return []v1.VolumeMount{}
}

// buildInitContainerCommand builds the command for an init container.
func buildInitContainerCommand(name string, volumes map[string]string) []string {
	if len(volumes) == 0 {
		return []string{}
	}
	var command []string = []string{"sh", "-c"}
	for path, _ := range volumes {
		command = append(command, fmt.Sprintf("mkdir -p /knuu/%s && cp -r %s/* /knuu/%s", path, path, path))
	}
	return command
}

// preparePod prepares a pod configuration.
func preparePod(namespace, name string, labels map[string]string, image string, command, args []string, env, volumes map[string]string, init bool) *v1.Pod {
	podEnv := buildEnv(env)
	podVolumes := buildPodVolumes(name, volumes)
	containerVolumes := buildContainerVolumes(name, volumes)
	initContainerVolumes := []v1.VolumeMount{}
	initContainerCommand := []string{}
	initContainers := []v1.Container{}
	if len(volumes) != 0 && init {
		initContainerVolumes = buildInitContainerVolumes(name, volumes)
		initContainerCommand = buildInitContainerCommand(name, volumes)
		initContainers = []v1.Container{
			{
				Name:         "volume-management",
				Image:        image,
				Command:      initContainerCommand,
				VolumeMounts: initContainerVolumes,
			},
		}
	}

	po := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
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

	return po
}
