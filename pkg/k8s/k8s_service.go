package k8s

import (
    "context"
    "errors"
    "fmt"
    "github.com/sirupsen/logrus"
    "time"

    v1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/intstr"
)

// GetService retrieves a service.
func GetService(namespace, name string) (*v1.Service, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	svc, err := Clientset().CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting service %s: %w", name, err)
	}
	return svc, nil
}

// DeployService deploys a service if it does not exist.
func DeployService(namespace, name string, labels, selectorMap map[string]string, portsTCP []int, portsUDP []int) (*v1.Service, error) {

	svc, err := prepareService(namespace, name, labels, selectorMap, portsTCP, portsUDP)
	if err != nil {
		return nil, fmt.Errorf("error preparing service %s: %w", name, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if !IsInitialized() {
		return nil, fmt.Errorf("knuu is not initialized")
	}
	serv, err := Clientset().CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating service %s: %w", name, err)
	}
	logrus.Debugf("Service %s deployed in namespace %s", name, namespace)
	return serv, nil
}

// PatchService patches an existing service.
func PatchService(namespace, name string, labels, selectorMap map[string]string, portsTCP, portsUDP []int) error {

	svc, err := prepareService(namespace, name, labels, selectorMap, portsTCP, portsUDP)
	if err != nil {
		return fmt.Errorf("error preparing service %s: %w", name, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	_, err = Clientset().CoreV1().Services(namespace).Update(ctx, svc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("error patching service %s: %w", name, err)
	}

	logrus.Debugf("Service %s patched in namespace %s", name, namespace)
	return nil
}

// DeleteService deletes a service if it exists.
func DeleteService(namespace, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	_, err := GetService(namespace, name)
	if err != nil {
		return fmt.Errorf("error getting service %s: %w", name, err)
	}

	if !IsInitialized() {
		return fmt.Errorf("knuu is not initialized")
	}
	err = Clientset().CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting service %s: %w", name, err)
	}

	logrus.Debugf("Service %s deleted in namespace %s", name, namespace)
	return nil
}

// GetServiceIP retrieves the IP address of a service.
func GetServiceIP(namespace, name string) (string, error) {
	svc, err := GetService(namespace, name)
	if err != nil {
		return "", fmt.Errorf("error getting service %s: %w", name, err)
	}
	return svc.Spec.ClusterIP, nil
}

// buildPorts constructs a list of ServicePort objects from the given TCP and UDP port lists.
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

// prepareService constructs a new Service object with the specified parameters.
func prepareService(namespace, name string, labels, selectorMap map[string]string,
	tcpPorts, udpPorts []int) (*v1.Service, error) {
	if namespace == "" {
		return nil, errors.New("namespace is required")
	}
	if name == "" {
		return nil, errors.New("service name is required")
	}
	if labels == nil {
		labels = make(map[string]string)
	}
	if selectorMap == nil {
		selectorMap = make(map[string]string)
	}

	servicePorts := buildPorts(tcpPorts, udpPorts)
	if len(servicePorts) == 0 {
		return nil, fmt.Errorf("no ports specified for service %s", name)
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
