package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// getStatefulSet retrieves a statefulSet from the given namespace and logs any errors.
func getStatefulSet(namespace, name string) (*appv1.StatefulSet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}

	statefulset, err := Clientset().AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulSet %s: %w", name, err)
	}

	return statefulset, nil
}

// DeployStatefulSet creates a new statefulSet in the given namespace if it doesn't already exist.
func DeployStatefulSet(statefulSetConfig StatefulSetConfig, init bool) (*appv1.StatefulSet, error) {
	// Prepare the pod
	statefulSet, err := prepareStatefulSet(statefulSetConfig, init)
	if err != nil {
		return nil, fmt.Errorf("error preparing pod: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Try to create the statefulSet
	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	createdStatefulSet, err := Clientset().AppsV1().StatefulSets(statefulSetConfig.Namespace).Create(ctx, statefulSet, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create statefulSet: %v", err)
	}

	return createdStatefulSet, nil
}

// StatefulSetConfig contains the specifications for creating a new StatefulSet object
type StatefulSetConfig struct {
	Name      string            // Name of the statefulSet
	Namespace string            // Namespace of the statefulSet
	Labels    map[string]string // Labels to apply to the statefulSet
	Replicas  int32             // Number of replicas
	PodConfig PodConfig         // Pod configuration
}

// ReplaceStatefulSetWithGracePeriod replaces a statefulSet in the given namespace and returns the new statefulSet object with a grace period.
func ReplaceStatefulSetWithGracePeriod(statefulSetConfig StatefulSetConfig, gracePeriod *int64) (*appv1.StatefulSet, error) {
	// Log a debug message to indicate that we are replacing a pod
	logrus.Debugf("Replacing statefulSet %s", statefulSetConfig.Name)

	// Delete the existing pod (if any)
	if err := DeleteStatefulSetWithGracePeriod(statefulSetConfig.Namespace, statefulSetConfig.Name, gracePeriod); err != nil {
		return nil, fmt.Errorf("failed to delete statefulSet: %v", err)
	}

	// Wait for the pod to be fully deleted
	for {
		_, err := getStatefulSet(statefulSetConfig.Namespace, statefulSetConfig.Name)
		if err != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Deploy the new pod
	statefulSet, err := DeployStatefulSet(statefulSetConfig, false)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy statefulSet: %v", err)
	}

	// Return the newly created pod
	return statefulSet, nil
}

// ReplaceStatefulSet replaces a statefulSet in the given namespace and returns the new StatefulSet object.
func ReplaceStatefulSet(statefulSetConfig StatefulSetConfig) (*appv1.StatefulSet, error) {
	return ReplaceStatefulSetWithGracePeriod(statefulSetConfig, nil)
}

// IsStatefulSetRunning returns true if the statefulSet is running.
func IsStatefulSetRunning(namespace, name string) (bool, error) {

	// Get the statefulSet from Kubernetes API server
	statefulSet, err := getStatefulSet(namespace, name)
	if err != nil {
		return false, fmt.Errorf("failed to get pod: %v", err)
	}

	// Check if the statefulSet is running
	return statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas, nil
}

// DeleteStatefulSetWithGracePeriod deletes a statefulSet with the given name in the specified namespace.
func DeleteStatefulSetWithGracePeriod(namespace, name string, gracePeriodSeconds *int64) error {
	// Get the statefulSet object from the API server
	_, err := getStatefulSet(namespace, name)
	if err != nil {
		// If the statefulSet does not exist, skip and return without error
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Delete the statefulSet using the Kubernetes client API
	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: gracePeriodSeconds,
	}
	if err := Clientset().AppsV1().StatefulSets(namespace).Delete(ctx, name, deleteOptions); err != nil {
		return fmt.Errorf("failed to delete statefulSet %s: %v", name, err)
	}

	return nil
}

// DeleteStatefulSet deletes a statefulSet with the given name in the specified namespace.
func DeleteStatefulSet(namespace, name string) error {
	return DeleteStatefulSetWithGracePeriod(namespace, name, nil)
}

// preparePod prepares a pod configuration.
func prepareStatefulSet(statefulSetConfig StatefulSetConfig, init bool) (*appv1.StatefulSet, error) {
	namespace := statefulSetConfig.Namespace
	name := statefulSetConfig.Name
	labels := statefulSetConfig.Labels
	replicas := statefulSetConfig.Replicas
	podConfig := statefulSetConfig.PodConfig

	podSpec, err := preparePodSpec(podConfig, init)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare pod spec: %w", err)
	}

	// Construct the StatefulSet object using the above data
	statefulSet := &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appv1.StatefulSetSpec{
			Replicas:    &replicas,
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			ServiceName: name,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      name,
					Labels:    labels,
				},
				Spec: podSpec,
			},
		},
	}

	logrus.Debugf("Prepared statefulSet %s in namespace %s", name, namespace)

	return statefulSet, nil
}

// GetFirstPod returns the first pod of a statefulset.
func GetFirstPodFromStatefulSet(namespace, name string) (*v1.Pod, error) {
	podName := fmt.Sprintf("%s-0", name)
	return getPod(namespace, podName)
}
