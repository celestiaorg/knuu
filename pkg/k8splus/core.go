package k8splus

import (
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type PodsGetter interface {
	Pods(namespace string) ExtendedPodInterface
}

type ExtendedCoreV1Interface interface {
	PodsGetter
	RESTClient() rest.Interface
	kv1.ComponentStatusesGetter
	kv1.ConfigMapsGetter
	kv1.EndpointsGetter
	kv1.EventsGetter
	kv1.LimitRangesGetter
	kv1.NamespacesGetter
	kv1.NodesGetter
	kv1.PersistentVolumesGetter
	kv1.PersistentVolumeClaimsGetter
	kv1.PodTemplatesGetter
	kv1.ReplicationControllersGetter
	kv1.ResourceQuotasGetter
	kv1.SecretsGetter
	kv1.ServicesGetter
	kv1.ServiceAccountsGetter
}

type extendedCoreV1Client struct {
	corev1.CoreV1Interface
	client rest.Interface
}

func (c *extendedCoreV1Client) Pods(namespace string) ExtendedPodInterface {
	return &pods{
		PodInterface: c.CoreV1Interface.Pods(namespace),
	}
}

func (c *Clientset) CoreV1() corev1.CoreV1Interface {
	return c.Clientset.CoreV1()
}

func (c *Clientset) ExtendedCoreV1() ExtendedCoreV1Interface {
	return &extendedCoreV1Client{
		CoreV1Interface: c.Clientset.CoreV1(),
		client:          c.Clientset.RESTClient(),
	}
}
