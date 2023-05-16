package k8s

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GetService retrieves a service.
func GetService(namespace, name string) (*v1.Service, error) {
	svc, err := Clientset.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting service %s: %w", name, err)
	}
	return svc, nil
}

// DeployService deploys a service if it does not exist.
func DeployService(namespace, name string, labels, selectorMap map[string]string, portsTCP, portsUDP []int) (*v1.Service, error) {
	svc, err := prepareService(namespace, name, labels, selectorMap, portsTCP, portsUDP)
	if err != nil {
		return nil, fmt.Errorf("error preparing service %s: %w", name, err)
	}

	svc, err = Clientset.CoreV1().Services(namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating service %s: %w", name, err)
	}
	logrus.Debugf("Service %s deployed", name)
	return svc, nil
}

// PatchService patches an existing service.
func PatchService(namespace, name string, labels, selectorMap map[string]string, portsTCP, portsUDP []int) error {
	svc, err := prepareService(namespace, name, labels, selectorMap, portsTCP, portsUDP)
	if err != nil {
		return fmt.Errorf("error preparing service %s: %w", name, err)
	}

	_, err = Clientset.CoreV1().Services(namespace).Update(context.Background(), svc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error patching service %s: %w", name, err)
	}

	logrus.Debugf("Service %s patched", name)
	return nil
}

// DeleteService deletes a service if it exists.
func DeleteService(namespace, name string) error {
	_, err := GetService(namespace, name)
	if err != nil {
		return fmt.Errorf("failed to get pod %s: %v", name, err)
	}

	err = Clientset.CoreV1().Services(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("Error deleting service %s: %w", name, err)
	}
	logrus.Debugf("Service %s deleted", name)
	return nil
}

// GetServiceIP retrieves the IP address of a service.
func GetServiceIP(namespace, name string) (string, error) {
	svc, err := Clientset.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("error getting service %s: %w", name, err)
	}
	return svc.Spec.ClusterIP, nil
}

// buildPorts constructs a list of ServicePort objects from the given TCP and UDP port lists.
func buildPorts(portsTCP, portsUDP []int) []v1.ServicePort {
	if len(portsTCP) == 0 && len(portsUDP) == 0 {
		portsTCP = append(portsTCP, 80)
	}

	servicePorts := make([]v1.ServicePort, len(portsTCP)+len(portsUDP))
	for i, port := range portsTCP {
		servicePorts[i] = v1.ServicePort{
			Name:       "tcp-" + fmt.Sprint(port),
			Protocol:   v1.ProtocolTCP,
			Port:       int32(port),
			TargetPort: intstr.FromInt(port),
		}
	}
	for i, port := range portsUDP {
		servicePorts[len(portsTCP)+i] = v1.ServicePort{
			Name:       "udp-" + fmt.Sprint(port),
			Protocol:   v1.ProtocolUDP,
			Port:       int32(port),
			TargetPort: intstr.FromInt(port),
		}
	}
	return servicePorts
}

// prepareService constructs a new Service object.
func prepareService(namespace, name string, labels, selectorMap map[string]string, portsTCP, portsUDP []int) (*v1.Service, error) {
	servicePorts := buildPorts(portsTCP, portsUDP)

	if len(servicePorts) == 0 {
		return nil, fmt.Errorf("error preparing service %s: no ports specified", name)
	}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: v1.ServiceSpec{
			Ports:    servicePorts,
			Selector: selectorMap,
			Type:     v1.ServiceTypeClusterIP,
		},
	}

	return svc, nil
}
