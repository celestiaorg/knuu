package k8s

import (
	"context"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateNamespace(ctx context.Context, name string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	_, err := c.clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return ErrCreatingNamespace.WithParams(name).Wrap(err)
		}
		logrus.Debugf("Namespace %s already exists, continuing.\n", name)
	}
	logrus.Debugf("Namespace %s created.\n", name)

	return nil
}

func (c *Client) DeleteNamespace(ctx context.Context, name string) error {
	err := c.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return ErrDeletingNamespace.WithParams(name).Wrap(err)
	}
	return nil
}

func (c *Client) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	namespace, err := c.clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, ErrGettingNamespace.WithParams(name).Wrap(err)
	}
	return namespace, nil
}

func (c *Client) NamespaceExists(ctx context.Context, name string) bool {
	_, err := c.GetNamespace(ctx, name)
	if err != nil {
		logrus.Debugf("Namespace %s does not exist, err: %v", name, err)
		return false
	}
	return true
}
