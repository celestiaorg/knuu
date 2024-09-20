package k8s

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) WaitForDeployment(ctx context.Context, name string) error {
	for {
		if c.terminated {
			return ErrClientTerminated
		}

		deployment, err := c.clientset.AppsV1().
			Deployments(c.namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return ErrWaitingForDeployment.WithParams(name).Wrap(err)
		}

		if deployment.Status.ReadyReplicas > 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ErrWaitingForDeployment.WithParams(name).Wrap(err)
		case <-time.After(waitRetry):
			// Retry after some seconds
		}
	}
}
