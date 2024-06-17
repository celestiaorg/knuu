package obsy

import (
	"context"
	"testing"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/system"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
	obsy    *Obsy
	ctx     context.Context
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
	s.obsy = New()
	s.ctx = context.Background()

	s.sysDeps = system.SystemDependencies{
		K8sCli: &mockK8sCli{
			namespace: "test",
		},
	}
}

func (s *TestSuite) TestNew() {
	o := New()
	s.Assert().NotNil(o)
	s.Assert().Equal(DefaultOtelCollectorVersion, o.obsyConfig.otelCollectorVersion)
}

func (s *TestSuite) TestInitialize() {
	err := s.obsy.Initialize(s.ctx, s.sysDeps)
	s.Require().NoError(err)
	s.Assert().NotNil(s.obsy.Instance())
	s.Assert().True(s.obsy.Instance().IsSidecar())
}

func (s *TestSuite) TestPreStart() {
	s.T().Skip("skipping as it is tested in e2e tests")
}

func (s *TestSuite) TestCloneWithSuffix() {
	err := s.obsy.Initialize(s.ctx, s.sysDeps)
	s.Require().NoError(err)

	clone := s.obsy.CloneWithSuffix("test")
	s.Assert().NotNil(clone)

	clonedObsy, ok := clone.(*Obsy)
	s.Assert().True(ok)
	s.Assert().Equal(s.obsy.obsyConfig, clonedObsy.obsyConfig)
}

func (s *TestSuite) TestSetters() {
	s.obsy.SetOtelCollectorVersion("test-version")
	s.Assert().Equal("test-version", s.obsy.obsyConfig.otelCollectorVersion)

	s.obsy.SetOtelEndpoint(8080)
	s.Assert().Equal(8080, s.obsy.obsyConfig.otlpPort)

	s.obsy.SetPrometheusEndpoint(9090, "test-job", "5s")
	s.Assert().Equal(9090, s.obsy.obsyConfig.prometheusEndpointPort)
	s.Assert().Equal("test-job", s.obsy.obsyConfig.prometheusEndpointJobName)
	s.Assert().Equal("5s", s.obsy.obsyConfig.prometheusEndpointScrapeInterval)

	s.obsy.SetJaegerEndpoint(14250, 6831, 14268)
	s.Assert().Equal(14250, s.obsy.obsyConfig.jaegerGrpcPort)
	s.Assert().Equal(6831, s.obsy.obsyConfig.jaegerThriftCompactPort)
	s.Assert().Equal(14268, s.obsy.obsyConfig.jaegerThriftHttpPort)

	s.obsy.SetOtlpExporter("http://test-endpoint", "user", "pass")
	s.Assert().Equal("http://test-endpoint", s.obsy.obsyConfig.otlpEndpoint)
	s.Assert().Equal("user", s.obsy.obsyConfig.otlpUsername)
	s.Assert().Equal("pass", s.obsy.obsyConfig.otlpPassword)

	s.obsy.SetJaegerExporter("http://jaeger-endpoint")
	s.Assert().Equal("http://jaeger-endpoint", s.obsy.obsyConfig.jaegerEndpoint)

	s.obsy.SetPrometheusExporter("http://prometheus-endpoint")
	s.Assert().Equal("http://prometheus-endpoint", s.obsy.obsyConfig.prometheusExporterEndpoint)

	s.obsy.SetPrometheusRemoteWriteExporter("http://prometheus-remote-write-endpoint")
	s.Assert().Equal("http://prometheus-remote-write-endpoint", s.obsy.obsyConfig.prometheusRemoteWriteExporterEndpoint)
}
