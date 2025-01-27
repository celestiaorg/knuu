package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// CreatePersistentVolumeClaim deploys a PersistentVolumeClaim if it does not exist.
func (c *Client) CreatePersistentVolumeClaim(
	ctx context.Context,
	name string,
	labels map[string]string,
	size resource.Quantity,
) error {
	if c.terminated {
		return ErrClientTerminated
	}
	if err := validatePVCName(name); err != nil {
		return err
	}
	if err := validatePVCSize(size); err != nil {
		return err
	}
	if err := validateLabels(labels); err != nil {
		return err
	}

	pvc := corev1.PersistentVolumeClaim(name, c.namespace).
		WithLabels(labels).
		WithSpec(corev1.PersistentVolumeClaimSpec().
			WithAccessModes(v1.ReadWriteOnce).
			WithResources(corev1.VolumeResourceRequirements().
				WithRequests(v1.ResourceList{
					v1.ResourceStorage: size,
				}),
			),
		)

	_, err := c.clientset.CoreV1().PersistentVolumeClaims(c.namespace).Apply(ctx, pvc, metav1.ApplyOptions{
		FieldManager: FieldManager,
	})
	if err != nil {
		return ErrCreatingPersistentVolumeClaim.WithParams(name).Wrap(err)
	}

	c.logger.WithField("name", name).Debug("PersistentVolumeClaim applied")
	return nil
}

func (c *Client) DeletePersistentVolumeClaim(ctx context.Context, name string) error {
	_, err := c.getPersistentVolumeClaim(ctx, name)
	if err != nil {
		// If the pvc does not exist, skip and return without error
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if err := c.clientset.CoreV1().PersistentVolumeClaims(c.namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return ErrDeletingPersistentVolumeClaim.WithParams(name).Wrap(err)
	}

	c.logger.WithField("name", name).Debug("PersistentVolumeClaim deleted")
	return nil
}

func (c *Client) getPersistentVolumeClaim(ctx context.Context, name string) (*v1.PersistentVolumeClaim, error) {
	return c.clientset.CoreV1().PersistentVolumeClaims(c.namespace).Get(ctx, name, metav1.GetOptions{})
}
