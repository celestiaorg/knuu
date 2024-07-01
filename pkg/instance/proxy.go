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

	prefix := fmt.Sprintf("%s-%d", i.k8sName, port)
	if err := i.Proxy.AddHost(ctx, i.k8sName, prefix, port); err != nil {
		return "", ErrAddingToProxy.WithParams(i.k8sName).Wrap(err)
	}
	host, err = i.Proxy.URL(ctx, prefix)
	if err != nil {
		return "", ErrGettingProxyURL.WithParams(i.k8sName).Wrap(err)
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
