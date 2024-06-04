// Package k8s provides utility functions for working with Kubernetes clusters.
package k8s

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

const (
	// tokenPath path in the filesystem to the service account token
	tokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	// certPath path in the filesystem to the ca.crt
	certPath = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	// waitRetry is the time to wait between retries for a readiness checking
	waitRetry = 2 * time.Second
)

type Client struct {
	clientset       kubernetes.Interface
	discoveryClient discovery.DiscoveryInterface
	dynamicClient   dynamic.Interface
	namespace       string
}

var _ KubeManager = &Client{}

func New(ctx context.Context, namespace string) (*Client, error) {
	config, err := getClusterConfig()
	if err != nil {
		return nil, ErrRetrievingKubernetesConfig.Wrap(err)
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, ErrCreatingClientset.Wrap(err)
	}

	// create discovery client
	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, ErrCreatingDiscoveryClient.Wrap(err)
	}

	// Create the dynamic client
	dC, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, ErrCreatingDynamicClient.Wrap(err)
	}
	kc := &Client{clientset: cs, discoveryClient: dc, dynamicClient: dC}

	namespace = SanitizeName(namespace)
	kc.namespace = namespace
	if kc.NamespaceExists(ctx, namespace) {
		logrus.Debugf("Namespace %s already exists, continuing.\n", namespace)
		return kc, nil
	}

	if err := kc.CreateNamespace(ctx, namespace); err != nil {
		return nil, ErrCreatingNamespace.WithParams(namespace).Wrap(err)
	}

	return kc, nil
}

func NewCustom(
	cs kubernetes.Interface,
	dc discovery.DiscoveryInterface,
	dC dynamic.Interface,
	namespace string,
) *Client {
	return &Client{
		clientset:       cs,
		discoveryClient: dc,
		dynamicClient:   dC,
		namespace:       namespace,
	}
}

func (c *Client) Clientset() kubernetes.Interface {
	return c.clientset
}

func (c *Client) DynamicClient() dynamic.Interface {
	return c.dynamicClient
}

func (c *Client) Namespace() string {
	return c.namespace
}

func (c *Client) DiscoveryClient() discovery.DiscoveryInterface {
	return c.discoveryClient
}
