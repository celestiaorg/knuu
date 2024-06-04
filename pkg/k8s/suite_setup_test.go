package k8s_test

import (
	"testing"

	"github.com/stretchr/testify/suite"
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

func (suite *TestSuite) SetupTest() {
	clientset := fake.NewSimpleClientset()
	discoveryClient := &discfake.FakeDiscovery{Fake: &k8stesting.Fake{}}
	dynamicClient := dynfake.NewSimpleDynamicClient(runtime.NewScheme())
	suite.namespace = "test"

	suite.client = k8s.NewCustom(clientset, discoveryClient, dynamicClient, suite.namespace)
}

func (suite *TestSuite) TearDownTest() {
}

func TestKubeManagerTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
