package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) GetConfigMap(ctx context.Context, name string) (*v1.ConfigMap, error) {
	cm, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, ErrGettingConfigmap.WithParams(name).Wrap(err)
	}

	return cm, nil
}

func (c *Client) ConfigMapExists(ctx context.Context, name string) (bool, error) {
	_, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrs.IsNotFound(err) {
			return false, nil
		}
		return false, ErrGettingConfigmap.WithParams(name).Wrap(err)
	}

	return true, nil
}

func (c *Client) CreateConfigMap(
	ctx context.Context, name string,
	labels, data map[string]string,
) (*v1.ConfigMap, error) {
	if c.terminated {
		return nil, ErrClientTerminated
	}

	if err := validateConfigMapName(name); err != nil {
		return nil, err
	}
	if err := validateLabels(labels); err != nil {
		return nil, err
	}
	if err := validateConfigMapKeys(data); err != nil {
		return nil, err
	}

	exists, err := c.ConfigMapExists(ctx, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrConfigmapAlreadyExists.WithParams(name)
	}

	cm := prepareConfigMap(c.namespace, name, labels, data)
	created, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return nil, ErrCreatingConfigmap.WithParams(name).Wrap(err)
	}

	return created, nil
}

func (c *Client) UpdateConfigMap(
	ctx context.Context, name string,
	labels, data map[string]string,
) (*v1.ConfigMap, error) {
	if c.terminated {
		return nil, ErrClientTerminated
	}

	if err := validateConfigMapName(name); err != nil {
		return nil, err
	}
	if err := validateLabels(labels); err != nil {
		return nil, err
	}
	if err := validateConfigMapKeys(data); err != nil {
		return nil, err
	}

	cm := prepareConfigMap(c.namespace, name, labels, data)
	updated, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return nil, ErrUpdatingConfigmap.WithParams(name).Wrap(err)
	}

	return updated, nil
}

func (c *Client) CreateOrUpdateConfigMap(
	ctx context.Context, name string,
	labels, data map[string]string,
) (*v1.ConfigMap, error) {
	exists, err := c.ConfigMapExists(ctx, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return c.UpdateConfigMap(ctx, name, labels, data)
	}

	return c.CreateConfigMap(ctx, name, labels, data)
}

func (c *Client) DeleteConfigMap(ctx context.Context, name string) error {
	exists, err := c.ConfigMapExists(ctx, name)
	if err != nil {
		return err
	}
	if !exists {
		return ErrConfigmapDoesNotExist.WithParams(name)
	}

	err = c.clientset.CoreV1().ConfigMaps(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return ErrDeletingConfigmap.WithParams(name).Wrap(err)
	}

	return nil
}

func prepareConfigMap(
	namespace, name string,
	labels, data map[string]string,
) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}
}
