package k8s

import (
	"context"
	"fmt"
	"time"

	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sirupsen/logrus"
)

// getReplicaSet retrieves replicaSet from the given namespace and logs any errors.
func getReplicaSet(ctx context.Context, namespace, name string) (*appv1.ReplicaSet, error) {
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
func DeployReplicaSet(ctx context.Context, replicaSetConfig ReplicaSetConfig, init bool) (*appv1.ReplicaSet, error) {
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
	Labels    map[string]string // Labels to apply to the ReplicaSet, key/value represents the name/value of the label
	Replicas  int32             // Replicas is the number of replicas
	PodConfig PodConfig         // PodConfig represents the pod configuration
}

// ReplaceReplicaSetWithGracePeriod replaces a ReplicaSet in the given namespace and returns the new ReplicaSet object with a grace period.
func ReplaceReplicaSetWithGracePeriod(ReplicaSetConfig ReplicaSetConfig, gracePeriod *int64) (*appv1.ReplicaSet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Log a debug message to indicate that we are replacing a ReplicaSet
	logrus.Debugf("Replacing ReplicaSet %s", ReplicaSetConfig.Name)

	// Delete the existing ReplicaSet (if any)
	if err := DeleteReplicaSetWithGracePeriod(ctx, ReplicaSetConfig.Namespace, ReplicaSetConfig.Name, gracePeriod); err != nil {
		return nil, fmt.Errorf("failed to delete ReplicaSet: %v", err)
	}

	// Wait for the ReplicaSet to be fully deleted
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	deleted := false
	for !deleted {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			_, err := getReplicaSet(ctx, ReplicaSetConfig.Namespace, ReplicaSetConfig.Name)
			if errors.IsNotFound(err) {
				// ReplicaSet has been deleted
				deleted = true
			} else if err != nil {
				return nil, fmt.Errorf("error waiting for ReplicaSet to delete: %v", err)
			}
			// If ReplicaSet still exists, wait for the next tick
		}
	}

	// Deploy the new ReplicaSet
	ReplicaSet, err := DeployReplicaSet(ctx, ReplicaSetConfig, false)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy ReplicaSet: %v", err)
	}

	// Return the newly created ReplicaSet
	return ReplicaSet, nil
}

// ReplaceReplicaSet replaces a ReplicaSet in the given namespace and returns the new ReplicaSet object.
func ReplaceReplicaSet(ReplicaSetConfig ReplicaSetConfig) (*appv1.ReplicaSet, error) {
	return ReplaceReplicaSetWithGracePeriod(ReplicaSetConfig, nil)
}

// IsReplicaSetRunning returns true if the ReplicaSet is running.
func IsReplicaSetRunning(namespace, name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Get the ReplicaSet from Kubernetes API server
	ReplicaSet, err := getReplicaSet(ctx, namespace, name)
	if err != nil {
		return false, fmt.Errorf("failed to get pod: %v", err)
	}

	// Check if the ReplicaSet is running
	return ReplicaSet.Status.ReadyReplicas == *ReplicaSet.Spec.Replicas, nil
}

// DeleteReplicaSetWithGracePeriod deletes a ReplicaSet with the given name in the specified namespace.
func DeleteReplicaSetWithGracePeriod(ctx context.Context, namespace, name string, gracePeriodSeconds *int64) error {
	// Get the ReplicaSet object from the API server
	_, err := getReplicaSet(ctx, namespace, name)
	if err != nil {
		// If the ReplicaSet does not exist, skip and return without error
		return nil
	}

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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return DeleteReplicaSetWithGracePeriod(ctx, namespace, name, nil)
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
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
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

// GetFirstPodFromReplicaSet returns the first pod of a Replicaset.
func GetFirstPodFromReplicaSet(namespace, name string) (*v1.Pod, error) {
	podName := fmt.Sprintf("%s-0", name)
	return getPod(namespace, podName)
}
