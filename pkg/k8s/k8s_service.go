package k8s

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (c *Client) GetService(ctx context.Context, name string) (*v1.Service, error) {
	svc, err := c.clientset.CoreV1().Services(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, ErrGettingService.WithParams(name).Wrap(err)
	}
	return svc, nil
}

func (c *Client) CreateService(
	ctx context.Context,
	name string,
	labels,
	selectorMap map[string]string,
	portsTCP,
	portsUDP []int,
) (*v1.Service, error) {
	svc, err := prepareService(c.namespace, name, labels, selectorMap, portsTCP, portsUDP)
	if err != nil {
		return nil, ErrPreparingService.WithParams(name).Wrap(err)
	}

	serv, err := c.clientset.CoreV1().Services(c.namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		return nil, ErrCreatingService.WithParams(name).Wrap(err)
	}
	logrus.Debugf("Service %s created in namespace %s", name, c.namespace)
	return serv, nil
}

func (c *Client) PatchService(
	ctx context.Context,
	name string,
	labels,
	selectorMap map[string]string,
	portsTCP,
	portsUDP []int,
) error {
	svc, err := prepareService(c.namespace, name, labels, selectorMap, portsTCP, portsUDP)
	if err != nil {
		return ErrPreparingService.WithParams(name).Wrap(err)
	}

	_, err = c.clientset.CoreV1().Services(c.namespace).Update(ctx, svc, metav1.UpdateOptions{})
	if err != nil {
		return ErrPatchingService.WithParams(name).Wrap(err)
	}

	logrus.Debugf("Service %s patched in namespace %s", name, c.namespace)
	return nil
}

func (c *Client) DeleteService(ctx context.Context, name string) error {
	_, err := c.GetService(ctx, name)
	if err != nil {
		return ErrGettingService.WithParams(name).Wrap(err)
	}

	err = c.clientset.CoreV1().Services(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return ErrDeletingService.WithParams(name).Wrap(err)
	}

	logrus.Debugf("Service %s deleted in namespace %s", name, c.namespace)
	return nil
}

func (c *Client) GetServiceIP(ctx context.Context, name string) (string, error) {
	svc, err := c.GetService(ctx, name)
	if err != nil {
		return "", ErrGettingService.WithParams(name).Wrap(err)
	}
	return svc.Spec.ClusterIP, nil
}

func buildPorts(tcpPorts, udpPorts []int) []v1.ServicePort {
	ports := make([]v1.ServicePort, 0, len(tcpPorts)+len(udpPorts))
	for _, port := range tcpPorts {
		ports = append(ports, v1.ServicePort{
			Name:       fmt.Sprintf("tcp-%d", port),
			Protocol:   v1.ProtocolTCP,
			Port:       int32(port),
			TargetPort: intstr.FromInt(port),
		})
	}
	for _, port := range udpPorts {
		ports = append(ports, v1.ServicePort{
			Name:       fmt.Sprintf("udp-%d", port),
			Protocol:   v1.ProtocolUDP,
			Port:       int32(port),
			TargetPort: intstr.FromInt(port),
		})
	}
	return ports
}

func prepareService(
	namespace, name string,
	labels, selectorMap map[string]string,
	tcpPorts, udpPorts []int,
) (*v1.Service, error) {
	if namespace == "" {
		return nil, ErrNamespaceRequired
	}
	if name == "" {
		return nil, ErrServiceNameRequired
	}
	if labels == nil {
		labels = make(map[string]string)
	}
	if selectorMap == nil {
		selectorMap = make(map[string]string)
	}

	servicePorts := buildPorts(tcpPorts, udpPorts)
	if len(servicePorts) == 0 {
		return nil, ErrNoPortsSpecified.WithParams(name)
	}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		},
		Spec: v1.ServiceSpec{
			Ports:    servicePorts,
			Selector: selectorMap,
			Type:     v1.ServiceTypeClusterIP,
		},
	}
	return svc, nil
}
