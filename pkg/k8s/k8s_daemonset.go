package k8s

import (
	"context"

	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) DaemonSetExists(ctx context.Context, name string) (bool, error) {
	_, err := c.clientset.AppsV1().DaemonSets(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrs.IsNotFound(err) {
			return false, nil
		}
		return false, ErrGettingDaemonset.WithParams(name).Wrap(err)
	}
	return true, nil
}

func (c *Client) GetDaemonSet(ctx context.Context, name string) (*appv1.DaemonSet, error) {
	ds, err := c.clientset.AppsV1().DaemonSets(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, ErrGettingDaemonset.WithParams(name).Wrap(err)
	}
	return ds, nil
}

func (c *Client) CreateDaemonSet(
	ctx context.Context,
	name string,
	labels map[string]string,
	initContainers []v1.Container,
	containers []v1.Container,
) (*appv1.DaemonSet, error) {
	ds, err := prepareDaemonSet(c.namespace, name, labels, initContainers, containers)
	if err != nil {
		return nil, err
	}

	created, err := c.clientset.AppsV1().DaemonSets(c.namespace).Create(ctx, ds, metav1.CreateOptions{})
	if err != nil {
		return nil, ErrCreatingDaemonset.WithParams(name).Wrap(err)
	}
	c.logger.Debugf("DaemonSet %s created in namespace %s", name, c.namespace)
	return created, nil
}

func (c *Client) UpdateDaemonSet(ctx context.Context,
	name string,
	labels map[string]string,
	initContainers []v1.Container,
	containers []v1.Container,
) (*appv1.DaemonSet, error) {
	ds, err := prepareDaemonSet(c.namespace, name, labels, initContainers, containers)
	if err != nil {
		return nil, err
	}

	updated, err := c.clientset.AppsV1().DaemonSets(c.namespace).Update(ctx, ds, metav1.UpdateOptions{})
	if err != nil {
		return nil, ErrUpdatingDaemonset.WithParams(name).Wrap(err)
	}
	c.logger.Debugf("DaemonSet %s updated in namespace %s", name, c.namespace)
	return updated, nil
}

func (c *Client) DeleteDaemonSet(ctx context.Context, name string) error {
	if err := c.clientset.AppsV1().DaemonSets(c.namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return ErrDeletingDaemonset.WithParams(name).Wrap(err)
	}
	c.logger.Debugf("DaemonSet %s deleted in namespace %s", name, c.namespace)
	return nil
}

func prepareDaemonSet(
	namespace, name string,
	labels map[string]string,
	initContainers,
	containers []v1.Container,
) (*appv1.DaemonSet, error) {
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
