package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) CreateTLSSecret(ctx context.Context, secretName string, cert, key []byte) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: c.namespace,
		},
		Data: map[string][]byte{
			"cert.pem": cert,
			"key.pem":  key,
		},
		Type: v1.SecretTypeOpaque,
	}

	_, err := c.clientset.CoreV1().Secrets(c.namespace).Create(ctx, secret, metav1.CreateOptions{})
	return err
}
