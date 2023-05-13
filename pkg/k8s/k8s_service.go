package k8s

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ServiceExists checks if a service exists.
func ServiceExists(namespace, name string) bool {
	return GetService(namespace, name) != nil
}

// GetService retrieves a service.
func GetService(namespace, name string) *v1.Service {
	svc, err := Clientset.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil
	}
	return svc
}

// DeployService deploys a service if it does not exist.
func DeployService(namespace, name string, labels, selectorMap map[string]string, portsTCP, portsUDP []int) *v1.Service {
	if ServiceExists(namespace, name) {
		logrus.Debugf("Service %s already exists, skipping...", name)
		return GetService(namespace, name)
	}

	svc := prepareService(namespace, name, labels, selectorMap, portsTCP, portsUDP)

	svc, err := Clientset.CoreV1().Services(namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	if err != nil {
		logrus.Fatalf("Error creating service %s: %v", name, err)
	}

	return svc
}

// PatchService patches an existing service.
func PatchService(namespace, name string, labels, selectorMap map[string]string, portsTCP, portsUDP []int) {
	svc := prepareService(namespace, name, labels, selectorMap, portsTCP, portsUDP)

	_, err := Clientset.CoreV1().Services(namespace).Update(context.Background(), svc, metav1.UpdateOptions{})
	if err != nil {
		logrus.Fatal(err)
	}
}

// DeleteService deletes a service if it exists.
func DeleteService(namespace, name string) error {
	if !ServiceExists(namespace, name) {
		logrus.Debugf("Service %s does not exist, skipping...", name)
		return nil
	}

	err := Clientset.CoreV1().Services(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("Error deleting service %s: %w", name, err)
	}
	return nil
}

// GetServiceIP retrieves the IP address of a service.
func GetServiceIP(namespace, name string) string {
	svc, err := Clientset.CoreV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		logrus.Fatal(err)
	}
	return svc.Spec.ClusterIP
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
func prepareService(namespace, name string, labels, selectorMap map[string]string, portsTCP, portsUDP []int) *v1.Service {
	servicePorts := buildPorts(portsTCP, portsUDP)

	if len(servicePorts) == 0 {
		logrus.Fatalf("Error preparing service %s: no ports specified", name)
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

	return svc
}
