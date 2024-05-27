package k8s

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) WaitForDeployment(ctx context.Context, name string) error {
	for {
		deployment, err := c.clientset.AppsV1().Deployments(c.namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil && deployment.Status.ReadyReplicas > 0 {
			break
		}

		select {
		case <-ctx.Done():
			return ErrWaitingForDeployment.WithParams(name).Wrap(err)
		case <-time.After(waitRetry):
			// Retry after some seconds
		}
	}

	return nil
}
