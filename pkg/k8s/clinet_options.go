package k8s

import "k8s.io/client-go/rest"

type ClientOptions struct {
	clusterDomain string
	clusterHost   string
	authToken     string
	cert          string
	clusterConfig *rest.Config
}

type Option func(*ClientOptions)

func WithClusterDomain(clusterDomain string) Option {
	return func(o *ClientOptions) {
		o.clusterDomain = clusterDomain
	}
}

func WithAuthToken(host, cert, authToken string) Option {
	return func(o *ClientOptions) {
		o.clusterHost = host
		o.authToken = authToken
		o.cert = cert
	}
}

func WithClusterConfig(clusterConfig *rest.Config) Option {
	return func(o *ClientOptions) {
		o.clusterConfig = clusterConfig
	}
}

func getAppliedOptions(options ...Option) *ClientOptions {
	opts := &ClientOptions{
		clusterDomain: defaultClusterDomain,
	}
	for _, opt := range options {
		opt(opts)
	}
	return opts
}
