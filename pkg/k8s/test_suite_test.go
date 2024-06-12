package k8s_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	discfake "k8s.io/client-go/discovery/fake"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

type TestSuite struct {
	suite.Suite
	client    *k8s.Client
	namespace string
}

func TestKubeManagerTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) SetupTest() {
	clientset := fake.NewSimpleClientset()
	discoveryClient := &discfake.FakeDiscovery{Fake: &k8stesting.Fake{}}
	dynamicClient := dynfake.NewSimpleDynamicClient(runtime.NewScheme())
	s.namespace = "test"

	s.client = k8s.NewCustom(clientset, discoveryClient, dynamicClient, s.namespace)
}

func (s *TestSuite) createConfigMap(name string) error {
	_, err := s.client.Clientset().CoreV1().ConfigMaps(s.namespace).
		Create(context.Background(), &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: s.namespace,
			},
		}, metav1.CreateOptions{})
	return err
}

func (s *TestSuite) createDaemonSet(name string) error {
	_, err := s.client.Clientset().AppsV1().DaemonSets(s.namespace).
		Create(context.Background(), &appv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: s.namespace,
			},
		}, metav1.CreateOptions{})
	return err
}

func (s *TestSuite) createNamespace(name string) error {
	_, err := s.client.Clientset().CoreV1().Namespaces().
		Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		}, metav1.CreateOptions{})
	return err
}

func (s *TestSuite) createNetworkPolicy(name string) error {
	_, err := s.client.Clientset().NetworkingV1().
		NetworkPolicies(s.namespace).
		Create(context.Background(), &netv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: s.namespace,
			},
		}, metav1.CreateOptions{})
	return err
}
