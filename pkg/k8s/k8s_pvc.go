package k8s

import (
	"context"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeployPersistentVolumeClaim deploys a PersistentVolumeClaim if it does not exist.
func DeployPersistentVolumeClaim(namespace, name string, labels map[string]string, size resource.Quantity, accessModes []string) {
	if PersistentVolumeClaimExists(namespace, name) {
		logrus.Debugf("PersistentVolume %s already exists, skipping...", name)
		return
	}

	pvc := preparePersistentVolumeClaim(namespace, name, labels, size, accessModes)

	_, err := Clientset.CoreV1().PersistentVolumeClaims(namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		logrus.Fatalf("Error creating PersistentVolume %s: %v", name, err)
	}
}

// DeletePersistentVolumeClaim deletes a PersistentVolumeClaim if it exists.
func DeletePersistentVolumeClaim(namespace, name string) {
	if !PersistentVolumeClaimExists(namespace, name) {
		logrus.Debugf("PersistentVolumeClaim %s does not exist, skipping...", name)
		return
	}

	err := Clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		logrus.Fatalf("Error deleting PersistentVolumeClaim %s: %v", name, err)
	}
	logrus.Debugf("PersistentVolumeClaim %s deleted", name)
}

// getPersistentVolumeClaim retrieves a PersistentVolumeClaim.
func getPersistentVolumeClaim(namespace, name string) *v1.PersistentVolumeClaim {
	pv, err := Clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil
	}
	return pv
}

// PersistentVolumeClaimExists checks if a PersistentVolumeClaim exists.
func PersistentVolumeClaimExists(namespace, name string) bool {
	return getPersistentVolumeClaim(namespace, name) != nil
}

// preparePersistentVolumeClaim prepares a PersistentVolumeClaim configuration.
func preparePersistentVolumeClaim(namespace, name string, labels map[string]string, size resource.Quantity, accessModes []string) *v1.PersistentVolumeClaim {
	pv := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.PersistentVolumeAccessMode(accessModes[0])},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: size,
				},
			},
		},
	}

	return pv
}
