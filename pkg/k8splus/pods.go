package k8splus

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type ExtendedPodInterface interface {
	kv1.PodInterface
	CreateBatch(ctx context.Context, pods []*v1.Pod, opts metav1.CreateOptions) ([]*v1.Pod, error)
}

// pods implements ExtendedPodInterface
type pods struct {
	corev1.PodInterface
}

func (c *pods) CreateBatch(ctx context.Context, pods []*v1.Pod, opts metav1.CreateOptions) ([]*v1.Pod, error) {
	var createdPods []*v1.Pod
	// bla bla bla

	return createdPods, nil
}
