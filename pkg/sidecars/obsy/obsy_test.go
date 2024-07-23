package obsy

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/system"
)

type TestSuite struct {
	suite.Suite
	sysDeps system.SystemDependencies
}

func TestObsyTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

type mockK8sCli struct {
	k8s.KubeManager
	namespace string
}

func (m *mockK8sCli) Namespace() string {
	return m.namespace
}

func (s *TestSuite) SetupTest() {
	s.sysDeps = system.SystemDependencies{
		K8sClient: &mockK8sCli{
			namespace:   "test",
			KubeManager: &k8s.Client{},
		},
		Logger: logrus.New(),
	}
}

func (s *TestSuite) TestNew() {
	o := New()
	s.Assert().NotNil(o)
	s.Assert().Equal(DefaultOtelCollectorVersion, o.obsyConfig.otelCollectorVersion)
}

func (s *TestSuite) TestInitialize() {
	o := New()
	err := o.Initialize(context.Background(), s.sysDeps)
	s.Require().NoError(err)
	s.Assert().NotNil(o.Instance())
	s.Assert().True(o.Instance().IsSidecar())
}

func (s *TestSuite) TestPreStart() {
	s.T().Skip("skipping as it is tested in e2e tests")
}

func (s *TestSuite) TestCloneWithSuffix() {
	o := New()
	err := o.Initialize(context.Background(), s.sysDeps)
	s.Require().NoError(err)

	clone := o.CloneWithSuffix("test")
	s.Assert().NotNil(clone)

	clonedObsy, ok := clone.(*Obsy)
	s.Assert().True(ok)
	s.Assert().Equal(o.obsyConfig, clonedObsy.obsyConfig)
}

func (s *TestSuite) TestSetters() {
	o := New()

	s.Require().NoError(o.SetOtelCollectorVersion("test-version"))
	s.Assert().Equal("test-version", o.obsyConfig.otelCollectorVersion)

	s.Require().NoError(o.SetOtelEndpoint(8080))
	s.Assert().Equal(8080, o.obsyConfig.otlpPort)

	s.Require().NoError(o.SetPrometheusEndpoint(9090, "test-job", "5s"))
	s.Assert().Equal(9090, o.obsyConfig.prometheusEndpointPort)
	s.Assert().Equal("test-job", o.obsyConfig.prometheusEndpointJobName)
	s.Assert().Equal("5s", o.obsyConfig.prometheusEndpointScrapeInterval)

	s.Require().NoError(o.SetJaegerEndpoint(14250, 6831, 14268))
	s.Assert().Equal(14250, o.obsyConfig.jaegerGrpcPort)
	s.Assert().Equal(6831, o.obsyConfig.jaegerThriftCompactPort)
	s.Assert().Equal(14268, o.obsyConfig.jaegerThriftHttpPort)

	s.Require().NoError(o.SetOtlpExporter("http://test-endpoint", "user", "pass"))
	s.Assert().Equal("http://test-endpoint", o.obsyConfig.otlpEndpoint)
	s.Assert().Equal("user", o.obsyConfig.otlpUsername)
	s.Assert().Equal("pass", o.obsyConfig.otlpPassword)

	s.Require().NoError(o.SetJaegerExporter("http://jaeger-endpoint"))
	s.Assert().Equal("http://jaeger-endpoint", o.obsyConfig.jaegerEndpoint)

	s.Require().NoError(o.SetPrometheusExporter("http://prometheus-endpoint"))
	s.Assert().Equal("http://prometheus-endpoint", o.obsyConfig.prometheusExporterEndpoint)

	s.Require().NoError(o.SetPrometheusRemoteWriteExporter("http://prometheus-remote-write-endpoint"))
	s.Assert().Equal("http://prometheus-remote-write-endpoint", o.obsyConfig.prometheusRemoteWriteExporterEndpoint)
}
