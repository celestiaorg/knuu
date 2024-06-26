// Package k8s provides utility functions for working with Kubernetes clusters.
package k8s

import (
	"context"
	"time"

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

	// CustomQPS is the QPS to use for the Kubernetes client, DefaultQPS: 5
	CustomQPS = 100

	// CustomBurst is the Burst to use for the Kubernetes client, DefaultBurst: 10.
	CustomBurst = 200
)

type Client struct {
	clientset       kubernetes.Interface
	discoveryClient discovery.DiscoveryInterface
	dynamicClient   dynamic.Interface
	namespace       string
}

var _ KubeManager = &Client{}

func NewClient(ctx context.Context, namespace string) (*Client, error) {
	config, err := getClusterConfig()
	if err != nil {
		return nil, ErrRetrievingKubernetesConfig.Wrap(err)
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, ErrCreatingClientset.Wrap(err)
	}

	dc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, ErrCreatingDiscoveryClient.Wrap(err)
	}

	dC, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, ErrCreatingDynamicClient.Wrap(err)
	}
	return NewClientCustom(ctx, cs, dc, dC, namespace)
}

func NewClientCustom(
	ctx context.Context,
	cs kubernetes.Interface,
	dc discovery.DiscoveryInterface,
	dC dynamic.Interface,
	namespace string,
) (*Client, error) {
	kc := &Client{
		clientset:       cs,
		discoveryClient: dc,
		dynamicClient:   dC,
		namespace:       namespace,
	}
	namespace = SanitizeName(namespace)
	kc.namespace = namespace
	if err := kc.CreateNamespace(ctx, namespace); err != nil {
		return nil, ErrCreatingNamespace.WithParams(namespace).Wrap(err)
	}
	return kc, nil
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
