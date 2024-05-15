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
	ErrKnuuNotInitialized              = &Error{Code: "KnuuNotInitialized", Message: "knuu is not initialized"}
	ErrGettingConfigmap                = &Error{Code: "ErrorGettingConfigmap", Message: "error getting configmap %s"}
	ErrConfigmapAlreadyExists          = &Error{Code: "ConfigmapAlreadyExists", Message: "configmap %s already exists"}
	ErrCreatingConfigmap               = &Error{Code: "ErrorCreatingConfigmap", Message: "error creating configmap %s"}
	ErrConfigmapDoesNotExist           = &Error{Code: "ConfigmapDoesNotExist", Message: "configmap %s does not exist"}
	ErrDeletingConfigmap               = &Error{Code: "ErrorDeletingConfigmap", Message: "error deleting configmap %s"}
	ErrGettingDaemonset                = &Error{Code: "ErrorGettingDaemonset", Message: "error getting daemonset %s"}
	ErrCreatingDaemonset               = &Error{Code: "ErrorCreatingDaemonset", Message: "error creating daemonset %s"}
	ErrUpdatingDaemonset               = &Error{Code: "ErrorUpdatingDaemonset", Message: "error updating daemonset %s"}
	ErrDeletingDaemonset               = &Error{Code: "ErrorDeletingDaemonset", Message: "error deleting daemonset %s"}
	ErrCreatingNamespace               = &Error{Code: "ErrorCreatingNamespace", Message: "error creating namespace %s"}
	ErrDeletingNamespace               = &Error{Code: "ErrorDeletingNamespace", Message: "error deleting namespace %s"}
	ErrGettingNamespace                = &Error{Code: "ErrorGettingNamespace", Message: "error getting namespace %s"}
	ErrCreatingNetworkPolicy           = &Error{Code: "ErrorCreatingNetworkPolicy", Message: "error creating network policy %s"}
	ErrDeletingNetworkPolicy           = &Error{Code: "ErrorDeletingNetworkPolicy", Message: "error deleting network policy %s"}
	ErrGettingNetworkPolicy            = &Error{Code: "ErrorGettingNetworkPolicy", Message: "error getting network policy %s"}
	ErrGettingPod                      = &Error{Code: "ErrorGettingPod", Message: "failed to get pod %s"}
	ErrPreparingPod                    = &Error{Code: "ErrorPreparingPod", Message: "error preparing pod"}
	ErrCreatingPod                     = &Error{Code: "ErrorCreatingPod", Message: "failed to create pod"}
	ErrDeletingPod                     = &Error{Code: "ErrorDeletingPod", Message: "failed to delete pod"}
	ErrDeployingPod                    = &Error{Code: "ErrorDeployingPod", Message: "failed to deploy pod"}
	ErrGettingK8sConfig                = &Error{Code: "ErrorGettingK8sConfig", Message: "failed to get k8s config"}
	ErrCreatingExecutor                = &Error{Code: "ErrorCreatingExecutor", Message: "failed to create Executor"}
	ErrExecutingCommand                = &Error{Code: "ErrorExecutingCommand", Message: "failed to execute command"}
	ErrCommandExecution                = &Error{Code: "ErrorCommandExecution", Message: "error while executing command"}
	ErrDeletingPodFailed               = &Error{Code: "ErrorDeletingPodFailed", Message: "failed to delete pod %s"}
	ErrParsingMemoryRequest            = &Error{Code: "ErrorParsingMemoryRequest", Message: "failed to parse memory request quantity '%s'"}
	ErrParsingMemoryLimit              = &Error{Code: "ErrorParsingMemoryLimit", Message: "failed to parse memory limit quantity '%s'"}
	ErrParsingCPURequest               = &Error{Code: "ErrorParsingCPURequest", Message: "failed to parse CPU request quantity '%s'"}
	ErrBuildingContainerVolumes        = &Error{Code: "ErrorBuildingContainerVolumes", Message: "failed to build container volumes"}
	ErrBuildingResources               = &Error{Code: "ErrorBuildingResources", Message: "failed to build resources"}
	ErrBuildingInitContainerVolumes    = &Error{Code: "ErrorBuildingInitContainerVolumes", Message: "failed to build init container volumes"}
	ErrBuildingInitContainerCommand    = &Error{Code: "ErrorBuildingInitContainerCommand", Message: "failed to build init container command"}
	ErrBuildingPodVolumes              = &Error{Code: "ErrorBuildingPodVolumes", Message: "failed to build pod volumes"}
	ErrPreparingMainContainer          = &Error{Code: "ErrorPreparingMainContainer", Message: "failed to prepare main container"}
	ErrPreparingInitContainer          = &Error{Code: "ErrorPreparingInitContainer", Message: "failed to prepare init container"}
	ErrPreparingPodVolumes             = &Error{Code: "ErrorPreparingPodVolumes", Message: "failed to prepare pod volumes"}
	ErrPreparingSidecarContainer       = &Error{Code: "ErrorPreparingSidecarContainer", Message: "failed to prepare sidecar container"}
	ErrPreparingSidecarVolumes         = &Error{Code: "ErrorPreparingSidecarVolumes", Message: "failed to prepare sidecar volumes"}
	ErrCreatingPodSpec                 = &Error{Code: "ErrorCreatingPodSpec", Message: "failed to create pod spec"}
	ErrGettingClusterConfig            = &Error{Code: "ErrorGettingClusterConfig", Message: "failed to get cluster config"}
	ErrCreatingRoundTripper            = &Error{Code: "ErrorCreatingRoundTripper", Message: "failed to create round tripper"}
	ErrCreatingPortForwarder           = &Error{Code: "ErrorCreatingPortForwarder", Message: "failed to create port forwarder"}
	ErrPortForwarding                  = &Error{Code: "ErrorPortForwarding", Message: "failed to port forward: %v"}
	ErrForwardingPorts                 = &Error{Code: "ErrorForwardingPorts", Message: "error forwarding ports"}
	ErrPortForwardingTimeout           = &Error{Code: "ErrorPortForwardingTimeout", Message: "timed out waiting for port forwarding to be ready"}
	ErrDeletingPersistentVolumeClaim   = &Error{Code: "ErrorDeletingPersistentVolumeClaim", Message: "error deleting PersistentVolumeClaim %s"}
	ErrCreatingPersistentVolumeClaim   = &Error{Code: "ErrorCreatingPersistentVolumeClaim", Message: "error creating PersistentVolumeClaim"}
	ErrGettingReplicaSet               = &Error{Code: "ErrorGettingReplicaSet", Message: "failed to get ReplicaSet %s"}
	ErrCreatingReplicaSet              = &Error{Code: "ErrorCreatingReplicaSet", Message: "failed to create ReplicaSet"}
	ErrDeletingReplicaSet              = &Error{Code: "ErrorDeletingReplicaSet", Message: "failed to delete ReplicaSet %s"}
	ErrWaitingForReplicaSet            = &Error{Code: "ErrorWaitingForReplicaSet", Message: "error waiting for ReplicaSet to delete"}
	ErrDeployingReplicaSet             = &Error{Code: "ErrorDeployingReplicaSet", Message: "failed to deploy ReplicaSet"}
	ErrPreparingPodSpec                = &Error{Code: "ErrorPreparingPodSpec", Message: "failed to prepare pod spec"}
	ErrListingPodsForReplicaSet        = &Error{Code: "ErrorListingPodsForReplicaSet", Message: "failed to list pods for ReplicaSet %s"}
	ErrNoPodsForReplicaSet             = &Error{Code: "NoPodsForReplicaSet", Message: "no pods found for ReplicaSet %s"}
	ErrGettingService                  = &Error{Code: "ErrorGettingService", Message: "error getting service %s"}
	ErrPreparingService                = &Error{Code: "ErrorPreparingService", Message: "error preparing service %s"}
	ErrCreatingService                 = &Error{Code: "ErrorCreatingService", Message: "error creating service %s"}
	ErrPatchingService                 = &Error{Code: "ErrorPatchingService", Message: "error patching service %s"}
	ErrDeletingService                 = &Error{Code: "ErrorDeletingService", Message: "error deleting service %s"}
	ErrNamespaceRequired               = &Error{Code: "NamespaceRequired", Message: "namespace is required"}
	ErrServiceNameRequired             = &Error{Code: "ServiceNameRequired", Message: "service name is required"}
	ErrNoPortsSpecified                = &Error{Code: "NoPortsSpecified", Message: "no ports specified for service %s"}
	ErrRetrievingKubernetesConfig      = &Error{Code: "RetrievingKubernetesConfig", Message: "retrieving the Kubernetes config"}
	ErrCreatingClientset               = &Error{Code: "CreatingClientset", Message: "creating clientset for Kubernetes"}
	ErrCreatingDiscoveryClient         = &Error{Code: "CreatingDiscoveryClient", Message: "creating discovery client for Kubernetes"}
	ErrCreatingDynamicClient           = &Error{Code: "CreatingDynamicClient", Message: "creating dynamic client for Kubernetes"}
	ErrGettingResourceList             = &Error{Code: "GettingResourceList", Message: "getting resource list for group version %s"}
	ErrResourceDoesNotExist            = &Error{Code: "ResourceDoesNotExist", Message: "resource %s does not exist in group version %s"}
	ErrCreatingCustomResource          = &Error{Code: "CreatingCustomResource", Message: "creating custom resource %s"}
	ErrCreatingRole                    = &Error{Code: "CreatingRole", Message: "creating role %s"}
	ErrCreatingRoleBinding             = &Error{Code: "CreatingRoleBinding", Message: "creating role binding %s"}
	ErrCreatingRoleBindingFailed       = &Error{Code: "CreatingRoleBindingFailed", Message: "creating role binding %s failed"}
	ErrNodePortNotSet                  = &Error{Code: "NodePortNotSet", Message: "node port not set"}
	ErrExternalIPsNotSet               = &Error{Code: "ExternalIPsNotSet", Message: "external IPs not set"}
	ErrGettingServiceEndpoint          = &Error{Code: "GettingServiceEndpoint", Message: "getting service endpoint %s"}
	ErrTimeoutWaitingForServiceReady   = &Error{Code: "TimeoutWaitingForServiceReady", Message: "timed out waiting for service %s to be ready"}
	ErrLoadBalancerIPNotAvailable      = &Error{Code: "LoadBalancerIPNotAvailable", Message: "load balancer IP not available"}
	ErrGettingNodes                    = &Error{Code: "GettingNodes", Message: "getting nodes"}
	ErrNoNodesFound                    = &Error{Code: "NoNodesFound", Message: "no nodes found"}
	ErrFailedToConnect                 = &Error{Code: "FailedToConnect", Message: "failed to connect to %s"}
	ErrWaitingForDeployment            = &Error{Code: "WaitingForDeployment", Message: "waiting for deployment %s to be ready"}
	ErrClusterRoleAlreadyExists        = &Error{Code: "ClusterRoleAlreadyExists", Message: "cluster role %s already exists"}
	ErrClusterRoleBindingAlreadyExists = &Error{Code: "ClusterRoleBindingAlreadyExists", Message: "cluster role binding %s already exists"}
	ErrCreateEndpoint                  = &Error{Code: "CreateEndpoint", Message: "failed to create endpoint for service %s"}
	ErrGetEndpoint                     = &Error{Code: "GetEndpoint", Message: "failed to get endpoint for service %s"}
	ErrUpdateEndpoint                  = &Error{Code: "UpdateEndpoint", Message: "failed to update endpoint for service %s"}
)
