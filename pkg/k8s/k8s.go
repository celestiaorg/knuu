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

	// CustomQPS is the QPS to use for the Kubernetes client, DefaultQPS: 5
	CustomQPS = 100

	// CustomBurst is the Burst to use for the Kubernetes client, DefaultBurst: 10.
	CustomBurst = 200

	// retryInterval is the interval to wait between retries
	retryInterval = 100 * time.Millisecond

	// if any pod is pending for more than this duration, a warning is logged
	defaultMaxPendingDuration = 60 * time.Second

	// defaultClusterDomain is the default cluster domain
	defaultClusterDomain = "cluster.local"

	// FieldManager is the field manager to use for the Kubernetes client
	FieldManager = "knuu"
)

type Client struct {
	clientset       kubernetes.Interface
	discoveryClient discovery.DiscoveryInterface
	dynamicClient   dynamic.Interface
	namespace       string
	clusterDomain   string
	logger          *logrus.Logger
	terminated      bool // This flag is used to indicate that the process has been terminated by the user
	// max duration for any pod to be in pending state, otherwise it triggers a notice to be shown
	maxPendingDuration time.Duration
}

type ClientOptions struct {
	clusterDomain string
}

type Option func(*ClientOptions)

func WithClusterDomain(clusterDomain string) Option {
	return func(o *ClientOptions) {
		o.clusterDomain = clusterDomain
	}
}

var _ KubeManager = &Client{}

func NewClient(ctx context.Context, namespace string, logger *logrus.Logger, options ...Option) (*Client, error) {
	config, err := getClusterConfig()
	if err != nil {
		return nil, ErrRetrievingKubernetesConfig.Wrap(err)
	}

	// Set custom QPS and Burst to avoid client rate limit errors
	config.QPS = CustomQPS
	config.Burst = CustomBurst

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

	return NewClientCustom(ctx, cs, dc, dC, namespace, logger, options...)
}

func NewClientCustom(
	ctx context.Context,
	cs kubernetes.Interface,
	dc discovery.DiscoveryInterface,
	dC dynamic.Interface,
	namespace string,
	logger *logrus.Logger,
	options ...Option,
) (*Client, error) {
	opts := &ClientOptions{
		clusterDomain: defaultClusterDomain,
	}
	for _, opt := range options {
		opt(opts)
	}

	if err := validateDNS1123Subdomain(
		opts.clusterDomain,
		ErrInvalidClusterDomain.WithParams(opts.clusterDomain),
	); err != nil {
		return nil, err
	}

	kc := &Client{
		clientset:          cs,
		discoveryClient:    dc,
		dynamicClient:      dC,
		clusterDomain:      opts.clusterDomain,
		logger:             logger,
		terminated:         false,
		maxPendingDuration: defaultMaxPendingDuration,
	}
	kc.namespace = SanitizeName(namespace)
	if err := kc.CreateNamespace(ctx, kc.namespace); err != nil {
		return nil, ErrCreatingNamespace.WithParams(kc.namespace).Wrap(err)
	}
	kc.startPendingPodsWarningMonitor(ctx)
	return kc, nil
}

func (c *Client) Terminate() {
	c.terminated = true
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

func (c *Client) SetMaxPendingDuration(duration time.Duration) {
	c.maxPendingDuration = duration
}
