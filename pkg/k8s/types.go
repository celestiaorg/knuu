package k8s

import (
	"context"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type KubeManager interface {
	Clientset() *kubernetes.Clientset
	CreateClusterRole(ctx context.Context, name string, labels map[string]string, policyRules []rbacv1.PolicyRule) error
	CreateClusterRoleBinding(ctx context.Context, name string, labels map[string]string, clusterRole, serviceAccount string) error
	CreateConfigMap(ctx context.Context, name string, labels, data map[string]string) (*corev1.ConfigMap, error)
	CreateCustomResource(ctx context.Context, name string, gvr *schema.GroupVersionResource, obj *map[string]interface{}) error
	CreateDaemonSet(ctx context.Context, name string, labels map[string]string, initContainers []corev1.Container, containers []corev1.Container) (*appv1.DaemonSet, error)
	CreateNamespace(ctx context.Context, name string) error
	CreateNetworkPolicy(ctx context.Context, name string, selectorMap, ingressSelectorMap, egressSelectorMap map[string]string) error
	CreatePersistentVolumeClaim(ctx context.Context, name string, labels map[string]string, size resource.Quantity) error
	CreateReplicaSet(ctx context.Context, rsConfig ReplicaSetConfig, init bool) (*appv1.ReplicaSet, error)
	CreateRole(ctx context.Context, name string, labels map[string]string, policyRules []rbacv1.PolicyRule) error
	CreateRoleBinding(ctx context.Context, name string, labels map[string]string, role, serviceAccount string) error
	CreateService(ctx context.Context, name string, labels, selectorMap map[string]string, portsTCP, portsUDP []int) (*corev1.Service, error)
	CreateServiceAccount(ctx context.Context, name string, labels map[string]string) error
	CustomResourceDefinitionExists(ctx context.Context, gvr *schema.GroupVersionResource) bool
	DaemonSetExists(ctx context.Context, name string) (bool, error)
	DeleteConfigMap(ctx context.Context, name string) error
	DeleteDaemonSet(ctx context.Context, name string) error
	DeleteNamespace(ctx context.Context, name string) error
	DeleteNetworkPolicy(ctx context.Context, name string) error
	DeletePersistentVolumeClaim(ctx context.Context, name string) error
	DeletePod(ctx context.Context, name string) error
	DeletePodWithGracePeriod(ctx context.Context, name string, gracePeriodSeconds *int64) error
	DeleteReplicaSet(ctx context.Context, name string) error
	DeleteReplicaSetWithGracePeriod(ctx context.Context, name string, gracePeriodSeconds *int64) error
	DeleteRole(ctx context.Context, name string) error
	DeleteRoleBinding(ctx context.Context, name string) error
	DeleteService(ctx context.Context, name string) error
	DeleteServiceAccount(ctx context.Context, name string) error
	DeployPod(ctx context.Context, podConfig PodConfig, init bool) (*corev1.Pod, error)
	DynamicClient() dynamic.Interface
	GetConfigMap(ctx context.Context, name string) (*corev1.ConfigMap, error)
	GetDaemonSet(ctx context.Context, name string) (*appv1.DaemonSet, error)
	GetFirstPodFromReplicaSet(ctx context.Context, name string) (*corev1.Pod, error)
	GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error)
	GetNetworkPolicy(ctx context.Context, name string) (*netv1.NetworkPolicy, error)
	GetService(ctx context.Context, name string) (*corev1.Service, error)
	GetServiceEndpoint(ctx context.Context, name string) (string, error)
	GetServiceIP(ctx context.Context, name string) (string, error)
	IsPodRunning(ctx context.Context, name string) (bool, error)
	IsReplicaSetRunning(ctx context.Context, name string) (bool, error)
	Namespace() string
	NamespaceExists(ctx context.Context, name string) bool
	NetworkPolicyExists(ctx context.Context, name string) bool
	NewFile(source, dest string) *File
	NewVolume(path, size string, owner int64) *Volume
	PatchService(ctx context.Context, name string, labels, selectorMap map[string]string, portsTCP, portsUDP []int) (*corev1.Service, error)
	PortForwardPod(ctx context.Context, podName string, localPort, remotePort int) error
	ReplicaSetExists(ctx context.Context, name string) (bool, error)
	ReplacePod(ctx context.Context, podConfig PodConfig) (*corev1.Pod, error)
	ReplacePodWithGracePeriod(ctx context.Context, podConfig PodConfig, gracePeriod *int64) (*corev1.Pod, error)
	ReplaceReplicaSet(ctx context.Context, ReplicaSetConfig ReplicaSetConfig) (*appv1.ReplicaSet, error)
	ReplaceReplicaSetWithGracePeriod(ctx context.Context, ReplicaSetConfig ReplicaSetConfig, gracePeriod *int64) (*appv1.ReplicaSet, error)
	RunCommandInPod(ctx context.Context, podName, containerName string, cmd []string) (string, error)
	getPersistentVolumeClaim(ctx context.Context, name string) (*corev1.PersistentVolumeClaim, error)
	getPod(ctx context.Context, name string) (*corev1.Pod, error)
	getReplicaSet(ctx context.Context, name string) (*appv1.ReplicaSet, error)
	ConfigMapExists(ctx context.Context, name string) (bool, error)
	UpdateDaemonSet(ctx context.Context, name string, labels map[string]string, initContainers []corev1.Container, containers []corev1.Container) (*appv1.DaemonSet, error)
	WaitForDeployment(ctx context.Context, name string) error
	WaitForService(ctx context.Context, name string) error
}
