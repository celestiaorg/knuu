package k8s

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ServiceOptions struct {
	Labels      map[string]string
	SelectorMap map[string]string
	TCPPorts    []int
	UDPPorts    []int
	NotHeadless bool
}

func (c *Client) GetService(ctx context.Context, name string) (*v1.Service, error) {
	if c.terminated {
		return nil, ErrClientTerminated
	}
	return c.clientset.CoreV1().Services(c.namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Client) CreateService(ctx context.Context, name string, opts ServiceOptions) (*v1.Service, error) {
	if c.terminated {
		return nil, ErrClientTerminated
	}
	if err := validateServiceName(name); err != nil {
		return nil, err
	}
	if err := validateServiceOptions(opts); err != nil {
		return nil, err
	}
	svc, err := prepareService(c.namespace, name, opts)
	if err != nil {
		return nil, ErrPreparingService.WithParams(name).Wrap(err)
	}

	serv, err := c.clientset.CoreV1().Services(c.namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		return nil, ErrCreatingService.WithParams(name).Wrap(err)
	}
	c.logger.WithFields(logrus.Fields{
		"name":      name,
		"namespace": c.namespace,
	}).Debug("service created")
	return serv, nil
}

func (c *Client) PatchService(ctx context.Context, name string, opts ServiceOptions) (*v1.Service, error) {
	if c.terminated {
		return nil, ErrClientTerminated
	}
	if err := validateServiceName(name); err != nil {
		return nil, err
	}
	if err := validateServiceOptions(opts); err != nil {
		return nil, err
	}

	svc, err := prepareService(c.namespace, name, opts)
	if err != nil {
		return nil, ErrPreparingService.WithParams(name).Wrap(err)
	}

	serv, err := c.clientset.CoreV1().Services(c.namespace).Update(ctx, svc, metav1.UpdateOptions{})
	if err != nil {
		return nil, ErrPatchingService.WithParams(name).Wrap(err)
	}

	c.logger.WithFields(logrus.Fields{
		"name":      name,
		"namespace": c.namespace,
	}).Debug("service patched")
	return serv, nil
}

func (c *Client) DeleteService(ctx context.Context, name string) error {
	_, err := c.GetService(ctx, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return ErrGettingService.WithParams(name).Wrap(err)
	}

	err = c.clientset.CoreV1().Services(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return ErrDeletingService.WithParams(name).Wrap(err)
	}

	c.logger.WithFields(logrus.Fields{
		"name":      name,
		"namespace": c.namespace,
	}).Debug("service deleted")
	return nil
}

func (c *Client) GetServiceIP(ctx context.Context, name string) (string, error) {
	srv, err := c.GetService(ctx, name)
	if err != nil {
		return "", ErrGettingService.WithParams(name).Wrap(err)
	}

	if srv.Spec.Type == v1.ServiceTypeLoadBalancer {
		// Use the LoadBalancer's external IP
		if len(srv.Status.LoadBalancer.Ingress) > 0 {
			return srv.Status.LoadBalancer.Ingress[0].IP, nil
		}
		return "", ErrLoadBalancerIPNotAvailable
	}

	if srv.Spec.Type != v1.ServiceTypeNodePort {
		// Headless service does not have a cluster IP
		if srv.Spec.ClusterIP == v1.ClusterIPNone {
			return "", ErrHeadlessService.WithParams(name)
		}
		return srv.Spec.ClusterIP, nil
	}

	// Use the Node IP and NodePort
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", ErrGettingNodes.Wrap(err)
	}
	if len(nodes.Items) == 0 {
		return "", ErrNoNodesFound
	}

	// Use the first node for simplicity, you might need to handle multiple nodes
	var nodeIP string
	for _, address := range nodes.Items[0].Status.Addresses {
		if address.Type == v1.NodeExternalIP {
			nodeIP = address.Address
			break
		}
	}
	return nodeIP, nil
}

// WaitForService() works only for the services with publicly accessible IP
func (c *Client) WaitForService(ctx context.Context, name string) error {
	srv, err := c.GetService(ctx, name)
	if err != nil {
		return ErrGettingService.WithParams(name).Wrap(err)
	}

	// Since this function is called from the client,
	// we cannot use headless service as it is not accessible from outside the cluster
	if isHeadlessService(srv) {
		return ErrCannotConnectToHeadlessService.WithParams(name)
	}

	retryInterval := time.Duration(0)
	for {
		select {
		case <-ctx.Done():
			return ErrTimeoutWaitingForServiceReady.WithParams(name)
		case <-time.After(retryInterval):
			// Reset to default interval
			retryInterval = waitRetry
		}

		ready, err := c.isServiceReady(ctx, name)
		if err != nil {
			return ErrCheckingServiceReady.WithParams(name).Wrap(err)
		}
		if !ready {
			continue
		}

		// Check if service is reachable
		// the service IP and port are used to check connectivity
		endpoint, err := c.GetServiceEndpoint(ctx, name)
		if err != nil {
			return ErrGettingServiceEndpoint.WithParams(name).Wrap(err)
		}
		if err := checkServiceConnectivity(endpoint); err != nil {
			continue
		}

		// Service is reachable
		return nil
	}
}

func (c *Client) ServiceDNS(name string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", name, c.namespace)
}

func (c *Client) GetServiceEndpoint(ctx context.Context, name string) (string, error) {
	ip, err := c.GetServiceIP(ctx, name)
	if err != nil {
		return "", ErrGettingServiceIP.WithParams(name).Wrap(err)
	}

	port, err := c.ServicePort(ctx, name)
	if err != nil {
		return "", ErrGettingService.WithParams(name).Wrap(err)
	}

	return fmt.Sprintf("%s:%d", ip, port), nil
}

func (c *Client) ServicePort(ctx context.Context, name string) (int32, error) {
	svc, err := c.GetService(ctx, name)
	if err != nil {
		return 0, ErrGettingService.WithParams(name).Wrap(err)
	}

	if svc.Spec.Type == v1.ServiceTypeLoadBalancer {
		if len(svc.Spec.Ports) > 0 {
			return svc.Spec.Ports[0].Port, nil
		}
		return 0, ErrNoPortsFoundForLoadBalancerService.WithParams(name)
	}

	if svc.Spec.Type == v1.ServiceTypeNodePort {
		if len(svc.Spec.Ports) > 0 && svc.Spec.Ports[0].NodePort != 0 {
			return svc.Spec.Ports[0].NodePort, nil
		}
		return 0, ErrNoNodePortFoundForService.WithParams(name)
	}

	// Handle ClusterIP and other service types
	if len(svc.Spec.Ports) > 0 {
		return svc.Spec.Ports[0].Port, nil
	}

	return 0, ErrNoPortsFoundForService.WithParams(name)
}

func (c *Client) isServiceReady(ctx context.Context, name string) (bool, error) {
	service, err := c.GetService(ctx, name)
	if err != nil {
		return false, ErrGettingService.WithParams(name).Wrap(err)
	}

	if isHeadlessService(service) {
		return c.isHeadlessServiceReady(ctx, service)
	}

	switch service.Spec.Type {
	case v1.ServiceTypeNodePort:
		return c.isNodePortServiceReady(ctx, service)
	case v1.ServiceTypeLoadBalancer:
		return len(service.Status.LoadBalancer.Ingress) > 0, nil
	default:
		return len(service.Spec.ExternalIPs) > 0, nil
	}
}

func (c *Client) isHeadlessServiceReady(ctx context.Context, service *v1.Service) (bool, error) {
	// For headless services, we check if the service has any pods ready
	pods, err := c.clientset.CoreV1().Pods(service.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: service.Spec.Selector,
		}),
	})
	if err != nil {
		return false, ErrGettingPodsForService.WithParams(service.Name).Wrap(err)
	}
	// Check if at least one pod is ready
	for _, pod := range pods.Items {
		if pod.Status.Phase == v1.PodRunning {
			return true, nil
		}
	}
	return false, nil
}

func (c *Client) isNodePortServiceReady(ctx context.Context, service *v1.Service) (bool, error) {
	// Check if NodePort is valid
	if service.Spec.Ports[0].NodePort == 0 {
		return false, nil
	}

	// Check if at least one node with an ExternalIP is available
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, ErrGettingNodes.Wrap(err)
	}

	for _, node := range nodes.Items {
		for _, address := range node.Status.Addresses {
			if address.Type == v1.NodeExternalIP && address.Address != "" {
				// NodePort service is ready if we have an external IP available
				return true, nil
			}
		}
	}
	// No node with ExternalIP found
	return false, nil
}

func checkServiceConnectivity(serviceEndpoint string) error {
	conn, err := net.DialTimeout("tcp", serviceEndpoint, waitRetry)
	if err != nil {
		return ErrFailedToConnect.WithParams(serviceEndpoint).Wrap(err)
	}
	defer conn.Close()
	return nil // success
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

func prepareService(namespace, name string, opts ServiceOptions) (*v1.Service, error) {
	if namespace == "" {
		return nil, ErrNamespaceRequired
	}
	if name == "" {
		return nil, ErrServiceNameRequired
	}
	if opts.Labels == nil {
		opts.Labels = make(map[string]string)
	}
	if opts.SelectorMap == nil {
		opts.SelectorMap = make(map[string]string)
	}

	servicePorts := buildPorts(opts.TCPPorts, opts.UDPPorts)
	if len(servicePorts) == 0 {
		return nil, ErrNoPortsSpecified.WithParams(name)
	}

	clusterIP := v1.ClusterIPNone
	if opts.NotHeadless {
		clusterIP = ""
	}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    opts.Labels,
		},
		Spec: v1.ServiceSpec{
			Ports:     servicePorts,
			Selector:  opts.SelectorMap,
			Type:      v1.ServiceTypeClusterIP,
			ClusterIP: clusterIP,
		},
	}
	return svc, nil
}

func isHeadlessService(srv *v1.Service) bool {
	return srv.Spec.ClusterIP == v1.ClusterIPNone
}
