package k8splus

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Clientset struct {
	*kubernetes.Clientset
}

func NewForConfig(c *rest.Config) (*Clientset, error) {
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	return &Clientset{Clientset: clientset}, nil
}
