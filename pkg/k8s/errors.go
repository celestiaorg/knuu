package k8s

import (
	"fmt"
)

type Error struct {
	Code    string
	Message string
	Err     error
	Params  []interface{}
}

func (e *Error) Error() string {
	msg := fmt.Sprintf(e.Message, e.Params...)
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", msg, e.Err)
	}
	return msg
}

func (e *Error) Wrap(err error) error {
	e.Err = err
	return e
}

func (e *Error) WithParams(params ...interface{}) *Error {
	e.Params = params
	return e
}

var (
	ErrBuildingContainerVolumes        = &Error{Code: "ErrorBuildingContainerVolumes", Message: "failed to build container volumes"}
	ErrBuildingInitContainerCommand    = &Error{Code: "ErrorBuildingInitContainerCommand", Message: "failed to build init container command"}
	ErrBuildingInitContainerVolumes    = &Error{Code: "ErrorBuildingInitContainerVolumes", Message: "failed to build init container volumes"}
	ErrBuildingPodVolumes              = &Error{Code: "ErrorBuildingPodVolumes", Message: "failed to build pod volumes"}
	ErrBuildingResources               = &Error{Code: "ErrorBuildingResources", Message: "failed to build resources"}
	ErrCommandExecution                = &Error{Code: "ErrorCommandExecution", Message: "error while executing command"}
	ErrConfigmapAlreadyExists          = &Error{Code: "ConfigmapAlreadyExists", Message: "configmap %s already exists"}
	ErrConfigmapDoesNotExist           = &Error{Code: "ConfigmapDoesNotExist", Message: "configmap %s does not exist"}
	ErrCreatingClientset               = &Error{Code: "CreatingClientset", Message: "creating clientset for Kubernetes"}
	ErrCreatingConfigmap               = &Error{Code: "ErrorCreatingConfigmap", Message: "error creating configmap %s"}
	ErrCreatingDaemonset               = &Error{Code: "ErrorCreatingDaemonset", Message: "error creating daemonset %s"}
	ErrCreatingExecutor                = &Error{Code: "ErrorCreatingExecutor", Message: "failed to create Executor"}
	ErrCreatingNamespace               = &Error{Code: "ErrorCreatingNamespace", Message: "error creating namespace %s"}
	ErrCreatingNetworkPolicy           = &Error{Code: "ErrorCreatingNetworkPolicy", Message: "error creating network policy %s"}
	ErrCreatingPersistentVolumeClaim   = &Error{Code: "ErrorCreatingPersistentVolumeClaim", Message: "error creating PersistentVolumeClaim"}
	ErrCreatingPod                     = &Error{Code: "ErrorCreatingPod", Message: "failed to create pod"}
	ErrCreatingPodSpec                 = &Error{Code: "ErrorCreatingPodSpec", Message: "failed to create pod spec"}
	ErrCreatingPortForwarder           = &Error{Code: "ErrorCreatingPortForwarder", Message: "failed to create port forwarder"}
	ErrCreatingReplicaSet              = &Error{Code: "ErrorCreatingReplicaSet", Message: "failed to create ReplicaSet"}
	ErrCreatingRoundTripper            = &Error{Code: "ErrorCreatingRoundTripper", Message: "failed to create round tripper"}
	ErrCreatingService                 = &Error{Code: "ErrorCreatingService", Message: "error creating service %s"}
	ErrDeletingConfigmap               = &Error{Code: "ErrorDeletingConfigmap", Message: "error deleting configmap %s"}
	ErrDeletingDaemonset               = &Error{Code: "ErrorDeletingDaemonset", Message: "error deleting daemonset %s"}
	ErrDeletingNamespace               = &Error{Code: "ErrorDeletingNamespace", Message: "error deleting namespace %s"}
	ErrDeletingNetworkPolicy           = &Error{Code: "ErrorDeletingNetworkPolicy", Message: "error deleting network policy %s"}
	ErrDeletingPersistentVolumeClaim   = &Error{Code: "ErrorDeletingPersistentVolumeClaim", Message: "error deleting PersistentVolumeClaim %s"}
	ErrDeletingPod                     = &Error{Code: "ErrorDeletingPod", Message: "failed to delete pod"}
	ErrDeletingPodFailed               = &Error{Code: "ErrorDeletingPodFailed", Message: "failed to delete pod %s"}
	ErrDeletingReplicaSet              = &Error{Code: "ErrorDeletingReplicaSet", Message: "failed to delete ReplicaSet %s"}
	ErrDeletingService                 = &Error{Code: "ErrorDeletingService", Message: "error deleting service %s"}
	ErrDeployingPod                    = &Error{Code: "ErrorDeployingPod", Message: "failed to deploy pod"}
	ErrDeployingReplicaSet             = &Error{Code: "ErrorDeployingReplicaSet", Message: "failed to deploy ReplicaSet"}
	ErrExecutingCommand                = &Error{Code: "ErrorExecutingCommand", Message: "failed to execute command"}
	ErrExternalIPsNotSet               = &Error{Code: "ExternalIPsNotSet", Message: "external IPs not set for service %s"}
	ErrFailedToConnect                 = &Error{Code: "FailedToConnect", Message: "failed to connect to %s"}
	ErrForwardingPorts                 = &Error{Code: "ErrorForwardingPorts", Message: "error forwarding ports"}
	ErrGettingClusterConfig            = &Error{Code: "ErrorGettingClusterConfig", Message: "failed to get cluster config"}
	ErrGettingConfigmap                = &Error{Code: "ErrorGettingConfigmap", Message: "error getting configmap %s"}
	ErrGettingDaemonset                = &Error{Code: "ErrorGettingDaemonset", Message: "error getting daemonset %s"}
	ErrGettingK8sConfig                = &Error{Code: "ErrorGettingK8sConfig", Message: "failed to get k8s config"}
	ErrGettingNamespace                = &Error{Code: "ErrorGettingNamespace", Message: "error getting namespace %s"}
	ErrGettingNetworkPolicy            = &Error{Code: "ErrorGettingNetworkPolicy", Message: "error getting network policy %s"}
	ErrGettingNodes                    = &Error{Code: "ErrorGettingNodes", Message: "error getting nodes"}
	ErrGettingPod                      = &Error{Code: "ErrorGettingPod", Message: "failed to get pod %s"}
	ErrGettingReplicaSet               = &Error{Code: "ErrorGettingReplicaSet", Message: "failed to get ReplicaSet %s"}
	ErrGettingService                  = &Error{Code: "ErrorGettingService", Message: "error getting service %s"}
	ErrGettingServiceEndpoint          = &Error{Code: "ErrorGettingServiceEndpoint", Message: "error getting service endpoint %s"}
	ErrKnuuNotInitialized              = &Error{Code: "KnuuNotInitialized", Message: "knuu is not initialized"}
	ErrLoadBalancerIPNotAvailable      = &Error{Code: "LoadBalancerIPNotAvailable", Message: "load balancer IP not available for service %s"}
	ErrListingPodsForReplicaSet        = &Error{Code: "ErrorListingPodsForReplicaSet", Message: "failed to list pods for ReplicaSet %s"}
	ErrNamespaceRequired               = &Error{Code: "NamespaceRequired", Message: "namespace is required"}
	ErrNoNodesFound                    = &Error{Code: "NoNodesFound", Message: "no nodes found"}
	ErrNoPodsForReplicaSet             = &Error{Code: "NoPodsForReplicaSet", Message: "no pods found for ReplicaSet %s"}
	ErrNoPortsSpecified                = &Error{Code: "NoPortsSpecified", Message: "no ports specified for service %s"}
	ErrNodePortNotSet                  = &Error{Code: "NodePortNotSet", Message: "node port not set for service %s"}
	ErrParsingCPURequest               = &Error{Code: "ErrorParsingCPURequest", Message: "failed to parse CPU request quantity '%s'"}
	ErrParsingMemoryLimit              = &Error{Code: "ErrorParsingMemoryLimit", Message: "failed to parse memory limit quantity '%s'"}
	ErrParsingMemoryRequest            = &Error{Code: "ErrorParsingMemoryRequest", Message: "failed to parse memory request quantity '%s'"}
	ErrPatchingService                 = &Error{Code: "ErrorPatchingService", Message: "error patching service %s"}
	ErrPortForwarding                  = &Error{Code: "ErrorPortForwarding", Message: "failed to port forward: %v"}
	ErrPortForwardingTimeout           = &Error{Code: "ErrorPortForwardingTimeout", Message: "timed out waiting for port forwarding to be ready"}
	ErrPreparingInitContainer          = &Error{Code: "ErrorPreparingInitContainer", Message: "failed to prepare init container"}
	ErrPreparingMainContainer          = &Error{Code: "ErrorPreparingMainContainer", Message: "failed to prepare main container"}
	ErrPreparingPod                    = &Error{Code: "ErrorPreparingPod", Message: "error preparing pod"}
	ErrPreparingPodSpec                = &Error{Code: "ErrorPreparingPodSpec", Message: "failed to prepare pod spec"}
	ErrPreparingPodVolumes             = &Error{Code: "ErrorPreparingPodVolumes", Message: "failed to prepare pod volumes"}
	ErrPreparingService                = &Error{Code: "ErrorPreparingService", Message: "error preparing service %s"}
	ErrPreparingSidecarContainer       = &Error{Code: "ErrorPreparingSidecarContainer", Message: "failed to prepare sidecar container"}
	ErrPreparingSidecarVolumes         = &Error{Code: "ErrorPreparingSidecarVolumes", Message: "failed to prepare sidecar volumes"}
	ErrRetrievingKubernetesConfig      = &Error{Code: "RetrievingKubernetesConfig", Message: "retrieving the Kubernetes config"}
	ErrServiceNameRequired             = &Error{Code: "ServiceNameRequired", Message: "service name is required"}
	ErrTimeoutWaitingForServiceReady   = &Error{Code: "TimeoutWaitingForServiceReady", Message: "timed out waiting for service %s to be ready"}
	ErrUpdatingDaemonset               = &Error{Code: "ErrorUpdatingDaemonset", Message: "error updating daemonset %s"}
	ErrWaitingForDeployment            = &Error{Code: "ErrorWaitingForDeployment", Message: "error waiting for Deployment %s to be ready"}
	ErrWaitingForReplicaSet            = &Error{Code: "ErrorWaitingForReplicaSet", Message: "error waiting for ReplicaSet to delete"}
	ErrClusterRoleAlreadyExists        = &Error{Code: "ClusterRoleAlreadyExists", Message: "ClusterRole %s already exists"}
	ErrClusterRoleBindingAlreadyExists = &Error{Code: "ClusterRoleBindingAlreadyExists", Message: "ClusterRoleBinding %s already exists"}
)
