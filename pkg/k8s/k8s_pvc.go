package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePersistentVolumeClaim deploys a PersistentVolumeClaim if it does not exist.
func (c *Client) CreatePersistentVolumeClaim(
	ctx context.Context,
	name string,
	labels map[string]string,
	size resource.Quantity,
) error {
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: c.namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: size,
				},
			},
		},
	}

	if _, err := c.clientset.CoreV1().PersistentVolumeClaims(c.namespace).Create(ctx, pvc, metav1.CreateOptions{}); err != nil {
		return ErrCreatingPersistentVolumeClaim.WithParams(name).Wrap(err)
	}

	c.logger.Debugf("PersistentVolumeClaim %s created", name)
	return nil
}

func (c *Client) DeletePersistentVolumeClaim(ctx context.Context, name string) error {
	_, err := c.getPersistentVolumeClaim(ctx, name)
	if err != nil {
		// If the pvc does not exist, skip and return without error
		return nil
	}

	if err := c.clientset.CoreV1().PersistentVolumeClaims(c.namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return ErrDeletingPersistentVolumeClaim.WithParams(name).Wrap(err)
	}

	c.logger.Debugf("PersistentVolumeClaim %s deleted", name)
	return nil
}

func (c *Client) getPersistentVolumeClaim(ctx context.Context, name string) (*v1.PersistentVolumeClaim, error) {
	return c.clientset.CoreV1().PersistentVolumeClaims(c.namespace).Get(ctx, name, metav1.GetOptions{})
}
