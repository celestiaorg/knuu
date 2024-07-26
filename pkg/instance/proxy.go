package instance

import (
	"context"
	"fmt"
	"time"
)

const (
	proxyWaitCheckInterval = 500 * time.Millisecond
)

func (i *Instance) AddHost(ctx context.Context, port int) (host string, err error) {
	if i.Proxy == nil {
		return "", ErrProxyNotInitialized
	}

	serviceName := i.k8sName
	if i.isSidecar {
		// The service is created for the main instance and
		// named after it which will be the parent instance for sidecars,
		// so we need to use the parent instance's service name.
		serviceName = i.parentInstance.k8sName
	}

	prefix := fmt.Sprintf("%s-%d", serviceName, port)
	if err := i.Proxy.AddHost(ctx, serviceName, prefix, port); err != nil {
		return "", ErrAddingToProxy.WithParams(serviceName).Wrap(err)
	}
	host, err = i.Proxy.URL(ctx, prefix)
	if err != nil {
		return "", ErrGettingProxyURL.WithParams(serviceName).Wrap(err)
	}
	return host, nil
}

func (i *Instance) AddHostWithReadyCheck(ctx context.Context, port int,
	checkFunc func(host string) (bool, error)) (host string, err error) {
	host, err = i.AddHost(ctx, port)
	if err != nil {
		return "", err
	}

	for {
		ok, err := checkFunc(host)
		if err != nil {
			return "", ErrCheckFailed.Wrap(err)
		}
		if ok {
			break
		}

		select {
		case <-ctx.Done():
			return "", ErrContextCanceled.Wrap(ctx.Err())
		case <-time.After(proxyWaitCheckInterval):
			// continue
		}
	}

	return host, nil
}
