package k8s

import (
	"context"
	"io"

	v1 "k8s.io/api/core/v1"
)

func (c *Client) GetLogStream(ctx context.Context, replicaSetName string, containerName string) (io.ReadCloser, error) {
	logOptions := &v1.PodLogOptions{}
	if containerName != "" {
		logOptions.Container = containerName
	}

	pod, err := c.GetFirstPodFromReplicaSet(ctx, replicaSetName)
	if err != nil {
		return nil, err
	}

	req := c.Clientset().CoreV1().Pods(c.Namespace()).GetLogs(pod.Name, logOptions)
	return req.Stream(ctx)
}
