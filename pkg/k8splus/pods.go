package k8splus

import (
	"context"
	"sync"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// ExtendedPodInterface is the interface for the Pods method
// This interface extends the PodInterface to include the CreateBatch method
type ExtendedPodInterface interface {
	kv1.PodInterface
	CreateBatchPods(ctx context.Context, pods []*v1.Pod, opts metav1.CreateOptions) ([]*v1.Pod, error)
}

// pods implements ExtendedPodInterface
type pods struct {
	corev1.PodInterface
}

// CreateBatchPods creates multiple pods in a single request
// It's a wrapper around the Create method and adds the CreateBatchPods method
func (c *pods) CreateBatchPods(ctx context.Context, pods []*v1.Pod, opts metav1.CreateOptions) ([]*v1.Pod, error) {
	// TODO @jrmanes: This is not what we want, as it makes several API calls
	createdPods := make([]*v1.Pod, len(pods))
	errors := make(chan error, len(pods))
	var wg sync.WaitGroup

	for i, pod := range pods {
		wg.Add(1)
		go func(pod *v1.Pod, index int) {
			defer wg.Done()
			createdPod, err := c.PodInterface.Create(ctx, pod, opts)
			if err != nil {
				errors <- err
				return
			}
			createdPods[index] = createdPod
		}(pod, i)
	}

	wg.Wait()
	close(errors)

	if len(errors) > 0 {
		return nil, <-errors // Return the first error encountered
	}

	return createdPods, nil
}
