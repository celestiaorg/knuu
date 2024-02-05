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

// getReplicaSet retrieves replicaSet from the given namespace and logs any errors.
func getReplicaSet(namespace, name string) (*appv1.ReplicaSetSet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}

	replicaset, err := Clientset().AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ReplicaSet %s: %w", name, err)
	}

	return replicaset, nil
}

// DeployReplicaSet creates a new replicaSet in the given namespace if it doesn't already exist.
func DeployReplicaSet(replicaSetConfig ReplicaSetConfig, init bool) (*appv1.ReplicaSet, error) {
	// Prepare the pod
	replicaSet, err := prepareReplicaSet(replicaSetConfig, init)
	if err != nil {
		return nil, fmt.Errorf("error preparing pod: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Try to create the ReplicaSet
	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	createdReplicaSet, err := Clientset().AppsV1().ReplicaSets(replicaSetConfig.Namespace).Create(ctx, replicaSet, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create ReplicaSet: %v", err)
	}

	return createdReplicaSet, nil
}

// ReplicaSetConfig contains the specifications for creating a new ReplicaSet object
type ReplicaSetConfig struct {
	Name      string            // Name of the ReplicaSet
	Namespace string            // Namespace of the ReplicaSet
	Labels    map[string]string // Labels to apply to the ReplicaSet
	Replicas  int32             // Number of replicas
	PodConfig PodConfig         // Pod configuration
}

// ReplaceReplicaSetWithGracePeriod replaces a ReplicaSet in the given namespace and returns the new ReplicaSet object with a grace period.
func ReplaceReplicaSetWithGracePeriod(ReplicaSetConfig ReplicaSetConfig, gracePeriod *int64) (*appv1.ReplicaSet, error) {
	// Log a debug message to indicate that we are replacing a pod
	logrus.Debugf("Replacing ReplicaSet %s", ReplicaSetConfig.Name)

	// Delete the existing pod (if any)
	if err := DeleteReplicaSetWithGracePeriod(ReplicaSetConfig.Namespace, ReplicaSetConfig.Name, gracePeriod); err != nil {
		return nil, fmt.Errorf("failed to delete ReplicaSet: %v", err)
	}

	// Wait for the pod to be fully deleted
	for {
		_, err := getReplicaSet(ReplicaSetConfig.Namespace, ReplicaSetConfig.Name)
		if err != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Deploy the new pod
	ReplicaSet, err := DeployReplicaSet(ReplicaSetConfig, false)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy ReplicaSet: %v", err)
	}

	// Return the newly created pod
	return ReplicaSet, nil
}

// ReplaceReplicaSet replaces a ReplicaSet in the given namespace and returns the new ReplicaSet object.
func ReplaceReplicaSet(ReplicaSetConfig ReplicaSetConfig) (*appv1.ReplicaSet, error) {
	return ReplaceReplicaSetWithGracePeriod(ReplicaSetConfig, nil)
}

// IsReplicaSetRunning returns true if the ReplicaSet is running.
func IsReplicaSetRunning(namespace, name string) (bool, error) {

	// Get the ReplicaSet from Kubernetes API server
	ReplicaSet, err := getReplicaSet(namespace, name)
	if err != nil {
		return false, fmt.Errorf("failed to get pod: %v", err)
	}

	// Check if the ReplicaSet is running
	return ReplicaSet.Status.ReadyReplicas == *ReplicaSet.Spec.Replicas, nil
}

// DeleteReplicaSetWithGracePeriod deletes a ReplicaSet with the given name in the specified namespace.
func DeleteReplicaSetWithGracePeriod(namespace, name string, gracePeriodSeconds *int64) error {
	// Get the ReplicaSet object from the API server
	_, err := getReplicaSet(namespace, name)
	if err != nil {
		// If the ReplicaSet does not exist, skip and return without error
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Delete the ReplicaSet using the Kubernetes client API
	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: gracePeriodSeconds,
	}
	if err := Clientset().AppsV1().ReplicaSets(namespace).Delete(ctx, name, deleteOptions); err != nil {
		return fmt.Errorf("failed to delete ReplicaSet %s: %v", name, err)
	}

	return nil
}

// DeleteReplicaSet deletes a ReplicaSet with the given name in the specified namespace.
func DeleteReplicaSet(namespace, name string) error {
	return DeleteReplicaSetWithGracePeriod(namespace, name, nil)
}

// preparePod prepares a pod configuration.
func prepareReplicaSet(ReplicaSetConfig ReplicaSetConfig, init bool) (*appv1.ReplicaSet, error) {
	namespace := ReplicaSetConfig.Namespace
	name := ReplicaSetConfig.Name
	labels := ReplicaSetConfig.Labels
	replicas := ReplicaSetConfig.Replicas
	podConfig := ReplicaSetConfig.PodConfig

	podSpec, err := preparePodSpec(podConfig, init)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare pod spec: %w", err)
	}

	// Construct the ReplicaSet object using the above data
	ReplicaSet := &appv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: appv1.ReplicaSetSpec{
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

	logrus.Debugf("Prepared ReplicaSet %s in namespace %s", name, namespace)

	return ReplicaSet, nil
}

// GetFirstPod returns the first pod of a Replicaset.
func GetFirstPodFromReplicaSet(namespace, name string) (*v1.Pod, error) {
	podName := fmt.Sprintf("%s-0", name)
	return getPod(namespace, podName)
}
