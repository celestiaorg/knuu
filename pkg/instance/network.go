package instance

import (
	"context"
	"net"
	"time"

	v1 "k8s.io/api/core/v1"
)

type network struct {
	instance          *Instance
	portsTCP          []int
	portsUDP          []int
	kubernetesService *v1.Service
}

func (i *Instance) Network() *network {
	return i.network
}

// AddPortTCP adds a TCP port to the instance
// This function can be called in the states 'Preparing' and 'Committed'
func (n *network) AddPortTCP(port int) error {
	if !n.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingPortNotAllowed.WithParams(n.instance.state.String())
	}

	if err := validatePort(port); err != nil {
		return err
	}
	if n.isTCPPortRegistered(port) {
		return ErrPortAlreadyRegistered.WithParams(port)
	}

	n.portsTCP = append(n.portsTCP, port)
	n.instance.Logger.Debugf("Added TCP port '%d' to instance '%s'", port, n.instance.name)
	return nil
}

// PortForwardTCP forwards the given port to a random port on the host
// This function can only be called in the state 'Started'
func (n *network) PortForwardTCP(ctx context.Context, port int) (int, error) {
	if !n.instance.IsState(StateStarted) {
		return -1, ErrRandomPortForwardingNotAllowed.WithParams(n.instance.state.String())
	}

	if err := validatePort(port); err != nil {
		return -1, err
	}
	if !n.isTCPPortRegistered(port) {
		return -1, ErrPortNotRegistered.WithParams(port)
	}
	// Get a random port on the host
	localPort, err := getFreePortTCP()
	if err != nil {
		return -1, ErrGettingFreePort.WithParams(port)
	}

	// Forward the port
	pod, err := n.instance.K8sClient.GetFirstPodFromReplicaSet(ctx, n.instance.k8sName)
	if err != nil {
		return -1, ErrGettingPodFromReplicaSet.WithParams(n.instance.k8sName).Wrap(err)
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := n.instance.K8sClient.PortForwardPod(ctx, pod.Name, localPort, port)
		if err == nil {
			break
		}

		select {
		case <-ctx.Done():
			return -1, ErrForwardingPort.WithParams(maxRetries)
		case <-time.After(retryInterval):
			// continue
		}

		if attempt == maxRetries {
			return -1, ErrForwardingPort.WithParams(maxRetries)
		}
		n.instance.Logger.Debugf("Forwarding port %d failed, cause: %v, retrying after %v (retry %d/%d)", port, err, retryInterval, attempt, maxRetries)
	}
	return localPort, nil
}

// AddPortUDP adds a UDP port to the instance
// This function can be called in the states 'Preparing' and 'Committed'
func (n *network) AddPortUDP(port int) error {
	if !n.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingPortNotAllowed.WithParams(n.instance.state.String())
	}

	if err := validatePort(port); err != nil {
		return err
	}
	if n.isUDPPortRegistered(port) {
		return ErrUDPPortAlreadyRegistered.WithParams(port)
	}
	n.portsUDP = append(n.portsUDP, port)

	n.instance.Logger.Debugf("Added UDP port '%d' to instance '%s'", port, n.instance.k8sName)
	return nil
}

// GetIP returns the IP of the instance
// This function can only be called in the states 'Preparing' and 'Started'
func (n *network) GetIP(ctx context.Context) (string, error) {
	// Check if i.kubernetesService already has the IP
	if n.kubernetesService != nil && n.kubernetesService.Spec.ClusterIP != "" {
		return n.kubernetesService.Spec.ClusterIP, nil
	}
	// If not, proceed with the existing logic to deploy the service and get the IP
	svc, err := n.instance.K8sClient.GetService(ctx, n.instance.k8sName)
	if err != nil || svc == nil {
		// Service does not exist, so we need to deploy it
		err := n.deployService(ctx, n.portsTCP, n.portsUDP)
		if err != nil {
			return "", ErrDeployingServiceForInstance.WithParams(n.instance.k8sName).Wrap(err)
		}
		svc, err = n.instance.K8sClient.GetService(ctx, n.instance.k8sName)
		if err != nil {
			return "", ErrGettingServiceForInstance.WithParams(n.instance.k8sName).Wrap(err)
		}
	}

	ip := svc.Spec.ClusterIP
	if ip == "" {
		return "", ErrGettingServiceIP.WithParams(n.instance.k8sName)
	}

	// Update i.kubernetesService for future reference
	n.kubernetesService = svc
	return ip, nil
}

// deployService deploys the service for the instance
func (n *network) deployService(ctx context.Context, portsTCP, portsUDP []int) error {
	// a sidecar instance should use the parent instance's service
	if n.instance.sidecars.IsSidecar() {
		return ErrDeployingServiceForSidecar.WithParams(n.instance.k8sName)
	}

	var (
		serviceName    = n.instance.k8sName
		labels         = n.instance.execution.Labels()
		labelSelectors = labels
	)

	srv, err := n.instance.K8sClient.CreateService(ctx, serviceName, labels, labelSelectors, portsTCP, portsUDP)
	if err != nil {
		return ErrDeployingService.WithParams(n.instance.k8sName).Wrap(err)
	}
	n.kubernetesService = srv
	n.instance.Logger.Debugf("Started service '%s'", n.instance.k8sName)
	return nil
}

// patchService patches the service for the instance
func (n *network) patchService(ctx context.Context, portsTCP, portsUDP []int) error {
	// a sidecar instance should use the parent instance's service
	if n.instance.sidecars.IsSidecar() {
		return ErrPatchingServiceForSidecar.WithParams(n.instance.k8sName)
	}

	var (
		serviceName    = n.instance.k8sName
		labels         = n.instance.execution.Labels()
		labelSelectors = labels
	)

	srv, err := n.instance.K8sClient.PatchService(ctx, serviceName, labels, labelSelectors, portsTCP, portsUDP)
	if err != nil {
		return ErrPatchingService.WithParams(serviceName).Wrap(err)
	}
	n.kubernetesService = srv
	n.instance.Logger.Debugf("Patched service '%s'", serviceName)
	return nil
}

// destroyService destroys the service for the instance
func (n *network) destroyService(ctx context.Context) error {
	return n.instance.K8sClient.DeleteService(ctx, n.instance.k8sName)
}

// isTCPPortRegistered returns true if the given port is registered
// with the instance, and false otherwise
func (n *network) isTCPPortRegistered(port int) bool {
	for _, p := range n.portsTCP {
		if p == port {
			return true
		}
	}
	return false
}

// isUDPPortRegistered returns true if the given port is registered
// with the instance, and false otherwise
func (n *network) isUDPPortRegistered(port int) bool {
	for _, p := range n.portsUDP {
		if p == port {
			return true
		}
	}
	return false
}

// validatePort validates the port
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return ErrPortNumberOutOfRange.WithParams(port)
	}
	return nil
}

// Disable disables the network of the instance
// This function can only be called in the state 'Started'
func (n *network) Disable(ctx context.Context) error {
	if !n.instance.IsInState(StateStarted) {
		return ErrDisablingNetworkNotAllowed.WithParams(n.instance.state.String())
	}

	err := n.instance.K8sClient.CreateNetworkPolicy(ctx, n.instance.k8sName, n.instance.execution.Labels(), nil, nil)
	if err != nil {
		return ErrDisablingNetwork.WithParams(n.instance.k8sName).Wrap(err)
	}
	return nil
}

// EnableNetwork enables the network of the instance
// This function can only be called in the state 'Started'
func (n *network) Enable(ctx context.Context) error {
	if !n.instance.IsInState(StateStarted) {
		return ErrEnablingNetworkNotAllowed.WithParams(n.instance.state.String())
	}

	err := n.instance.K8sClient.DeleteNetworkPolicy(ctx, n.instance.k8sName)
	if err != nil {
		return ErrEnablingNetwork.WithParams(n.instance.k8sName).Wrap(err)
	}
	return nil
}

// IsDisabled returns true if the network of the instance is disabled
// This function can only be called in the state 'Started'
func (n *network) IsDisabled(ctx context.Context) (bool, error) {
	if !n.instance.IsInState(StateStarted) {
		return false, ErrCheckingIfNetworkDisabledNotAllowed.WithParams(n.instance.state.String())
	}

	return n.instance.K8sClient.NetworkPolicyExists(ctx, n.instance.k8sName), nil
}

// deployService deploys the service for the instance
func (n *network) deployOrPatchService(ctx context.Context, portsTCP, portsUDP []int) error {
	if len(portsTCP) == 0 && len(portsUDP) == 0 {
		return nil
	}

	n.instance.Logger.Debugf("Ports not empty, deploying service for instance '%s'", n.instance.k8sName)
	svc, _ := n.instance.K8sClient.GetService(ctx, n.instance.k8sName)
	if svc == nil {
		if err := n.deployService(ctx, portsTCP, portsUDP); err != nil {
			return ErrDeployingServiceForInstance.WithParams(n.instance.k8sName).Wrap(err)
		}
		return nil
	}

	if err := n.patchService(ctx, portsTCP, portsUDP); err != nil {
		return ErrPatchingServiceForInstance.WithParams(n.instance.k8sName).Wrap(err)
	}
	return nil
}

func (n *network) enableIfDisabled(ctx context.Context) error {
	disableNetwork, err := n.IsDisabled(ctx)
	if err != nil {
		n.instance.Logger.Errorf("error checking network status for instance")
		return ErrCheckingNetworkStatusForInstance.WithParams(n.instance.k8sName).Wrap(err)
	}

	if !disableNetwork {
		return nil
	}
	if err := n.Enable(ctx); err != nil {
		n.instance.Logger.Errorf("error enabling network for instance")
		return ErrEnablingNetworkForInstance.WithParams(n.instance.k8sName).Wrap(err)
	}
	return nil
}

func (n *network) clone() *network {
	if n == nil {
		return nil
	}

	portsTCPCopy := make([]int, len(n.portsTCP))
	copy(portsTCPCopy, n.portsTCP)

	portsUDPCopy := make([]int, len(n.portsUDP))
	copy(portsUDPCopy, n.portsUDP)

	return &network{
		instance:          nil,
		portsTCP:          portsTCPCopy,
		portsUDP:          portsUDPCopy,
		kubernetesService: nil, //TODO: discuss the implementation of a clone for the service
	}
}

// getFreePort returns a free port
func getFreePortTCP() (int, error) {
	// Get a random port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, ErrGettingFreePort.Wrap(err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}
