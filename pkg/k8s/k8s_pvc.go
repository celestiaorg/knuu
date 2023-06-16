package k8s

import (
    "context"
    "fmt"
    "time"

    "github.com/sirupsen/logrus"
    v1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/resource"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// createPersistentVolumeClaim deploys a PersistentVolumeClaim if it does not exist.
func createPersistentVolumeClaim(namespace, name string, labels map[string]string, size resource.Quantity, accessModes []v1.PersistentVolumeAccessMode) error {
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: size,
				},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	if _, err := Clientset().CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{}); err != nil {
		return err
	}

	logrus.Debugf("PersistentVolumeClaim %s created", name)
	return nil
}

// deletePersistentVolumeClaim deletes a PersistentVolumeClaim if it exists.
func deletePersistentVolumeClaim(namespace, name string) error {
	// Get the pvc object from the API server
	_, err := getPersistentVolumeClaim(namespace, name)
	if err != nil {
		// If the pvc does not exist, skip and return without error
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	if err := Clientset().CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("error deleting PersistentVolumeClaim %s: %w", name, err)
	}

	logrus.Debugf("PersistentVolumeClaim %s deleted", name)
	return nil
}

// getPersistentVolumeClaim retrieves a PersistentVolumeClaim.
func getPersistentVolumeClaim(namespace, name string) (*v1.PersistentVolumeClaim, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	pv, err := Clientset().CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return pv, nil
}

// DeployPersistentVolumeClaim creates a new PersistentVolumeClaim in the specified namespace.
func DeployPersistentVolumeClaim(namespace, name string, labels map[string]string, size resource.Quantity) {
	accessModes := []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
	if err := createPersistentVolumeClaim(namespace, name, labels, size, accessModes); err != nil {
		logrus.Fatalf("Error creating PersistentVolumeClaim %s: %v", name, err)
	}
}

// DeletePersistentVolumeClaim deletes the PersistentVolumeClaim with the specified name in the specified namespace.
func DeletePersistentVolumeClaim(namespace, name string) {
	if err := deletePersistentVolumeClaim(namespace, name); err != nil {
		logrus.Fatalf("Error deleting PersistentVolumeClaim %s: %v", name, err)
	}
}
