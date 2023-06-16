package k8s

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// DaemonSetExists checks if a daemonset exists.
func DaemonSetExists(namespace, name string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !IsInitialized() {
		return false, fmt.Errorf("knuu is not initialized")
	}
	_, err := Clientset().AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("error getting daemonset %s: %w", name, err)
	}
	return true, nil
}

// GetDaemonSet retrieves a daemonset.
func GetDaemonSet(namespace, name string) (*appv1.DaemonSet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	ds, err := Clientset().AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting daemonset %s: %w", name, err)
	}
	return ds, nil
}

// CreateDaemonSet creates a new daemonset.
func CreateDaemonSet(namespace, name string, labels map[string]string, initContainers []v1.Container, containers []v1.Container) (*appv1.DaemonSet, error) {

	ds, err := prepareDaemonSet(namespace, name, labels, initContainers, containers)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	created, err := Clientset().AppsV1().DaemonSets(namespace).Create(ctx, ds, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating daemonset %s: %w", name, err)
	}
	logrus.Debugf("DaemonSet %s created in namespace %s", name, namespace)
	return created, nil
}

// UpdateDaemonSet updates an existing daemonset.
func UpdateDaemonSet(namespace, name string, labels map[string]string, initContainers []v1.Container, containers []v1.Container) (*appv1.DaemonSet, error) {

	ds, err := prepareDaemonSet(namespace, name, labels, initContainers, containers)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	updated, err := Clientset().AppsV1().DaemonSets(namespace).Update(ctx, ds, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error updating daemonset %s: %w", name, err)
	}
	logrus.Debugf("DaemonSet %s updated in namespace %s", name, namespace)
	return updated, nil
}

// DeleteDaemonSet deletes an existing daemonset.
func DeleteDaemonSet(namespace, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	if err := Clientset().AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("error deleting daemonset %s: %w", name, err)
	}
	logrus.Debugf("DaemonSet %s deleted in namespace %s", name, namespace)
	return nil
}

// prepareService constructs a new Service object with the specified parameters.
func prepareDaemonSet(namespace, name string, labels map[string]string, initContainers []v1.Container, containers []v1.Container) (*appv1.DaemonSet, error) {

	ds := &appv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					InitContainers: initContainers,
					Containers:     containers,
				},
			},
		},
	}

	return ds, nil

}
