package k8s

import (
	"context"

	"github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// EnsureEndpointWithPort ensures that an endpoint for a given service exists
// with the specified IP and port. If the endpoint does not exist, it creates a new one.
// If it exists but does not have the specified port, it updates the existing endpoint to include the port.
func (c *Client) EnsureEndpointWithPort(ctx context.Context, serviceName, ip string, port int) error {
	eCli := c.clientset.CoreV1().Endpoints(c.namespace)
	endpoint, err := eCli.Get(ctx, serviceName, metav1.GetOptions{})

	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return ErrGetEndpoint.WithParams(serviceName).Wrap(err)
		}
		return c.createEndpoint(ctx, eCli, serviceName, ip, port)
	}

	return c.updateEndpointWithPort(ctx, eCli, endpoint, serviceName, ip, port)
}

func (c *Client) createEndpoint(ctx context.Context, eCli corev1.EndpointsInterface, serviceName, ip string, port int) error {
	endpoint := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: c.namespace,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{{IP: ip}},
				Ports:     []v1.EndpointPort{{Port: int32(port), Protocol: v1.ProtocolTCP}},
			},
		},
	}

	_, err := eCli.Create(ctx, endpoint, metav1.CreateOptions{})
	if err != nil {
		return ErrCreateEndpoint.WithParams(serviceName).Wrap(err)
	}

	logrus.Infof("Endpoint for service %s created with port %d.", serviceName, port)
	return nil
}

func (c *Client) updateEndpointWithPort(ctx context.Context, eCli corev1.EndpointsInterface, endpoint *v1.Endpoints, serviceName, ip string, port int) error {
	if endpointHasPort(endpoint, port) {
		logrus.Infof("Port %d already exists in endpoint for service %s, no action needed.", port, serviceName)
		return nil
	}

	if len(endpoint.Subsets) == 0 {
		endpoint.Subsets = append(endpoint.Subsets, v1.EndpointSubset{})
	}

	subset := &endpoint.Subsets[0]
	subset.Ports = append(subset.Ports, v1.EndpointPort{Port: int32(port), Protocol: v1.ProtocolTCP})
	subset.Addresses = append(subset.Addresses, v1.EndpointAddress{IP: ip})

	_, err := eCli.Update(ctx, endpoint, metav1.UpdateOptions{})
	if err != nil {
		return ErrUpdateEndpoint.WithParams(serviceName).Wrap(err)
	}

	logrus.Infof("Port %d added to the existing endpoint for service %s.", port, serviceName)
	return nil
}

func endpointHasPort(endpoint *v1.Endpoints, port int) bool {
	for _, subset := range endpoint.Subsets {
		for _, p := range subset.Ports {
			if int(p.Port) == port {
				return true
			}
		}
	}
	return false
}
