package basic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/celestiaorg/knuu/pkg/sidecars/observability"
)

const (
	prometheusPort   = observability.DefaultOtelMetricsPort
	prometheusImage  = "prom/prometheus:latest"
	prometheusConfig = "/etc/prometheus/prometheus.yml"
	prometheusArgs   = "--config.file=/etc/prometheus/prometheus.yml"

	curlImage = "curlimages/curl:latest"
	otlpPort  = observability.DefaultOtelOtlpPort
)

// TestObservabilityCollector is a test function that verifies the functionality of the otel collector setup
func (s *Suite) TestObservabilityCollector() {
	const (
		namePrefix         = "observability"
		targetStartCommand = "while true; do curl -X POST http://localhost:8888/v1/traces; sleep 5; done"
	)
	ctx := context.Background()

	// Setup Prometheus
	prometheus, err := s.Knuu.NewInstance(namePrefix + "-prometheus")
	s.Require().NoError(err)

	s.Require().NoError(prometheus.Build().SetImage(ctx, prometheusImage))
	s.Require().NoError(prometheus.Network().AddPortTCP(prometheusPort))

	// enable proxy for this port
	prometheusEndpoint, err := prometheus.Network().AddHost(ctx, prometheusPort)
	s.Require().NoError(err)

	s.Require().NoError(prometheus.Build().Commit(ctx))

	// Add Prometheus config file
	prometheusConfigContent := fmt.Sprintf(`
global:
  scrape_interval: '10s'
scrape_configs:
  - job_name: 'otel-collector'
    static_configs:
      - targets: ['otel-collector:%d']
`, otlpPort)
	s.Require().NoError(prometheus.Storage().AddFileBytes([]byte(prometheusConfigContent), prometheusConfig, "0:0"))

	s.Require().NoError(prometheus.Build().SetArgs(prometheusArgs))
	s.Require().NoError(prometheus.Execution().Start(ctx))

	// Setup observabilitySidecar collector
	observabilitySidecar := observability.New()

	s.Require().NoError(observabilitySidecar.SetOtelEndpoint(4318))
	s.Require().NoError(observabilitySidecar.SetPrometheusEndpoint(otlpPort, fmt.Sprintf("knuu-%s", s.Knuu.Scope), "10s"))
	s.Require().NoError(observabilitySidecar.SetJaegerEndpoint(14250, 6831, 14268))
	s.Require().NoError(observabilitySidecar.SetOtlpExporter("prometheus:9090", "", ""))

	// Create and start a target pod and configure it to use the obsySidecar to push metrics
	target, err := s.Knuu.NewInstance(namePrefix + "target")
	s.Require().NoError(err)

	s.Require().NoError(target.Build().SetImage(ctx, curlImage))

	err = target.Build().SetStartCommand(targetStartCommand)
	s.Require().NoError(err)

	s.Require().NoError(target.Sidecars().Add(ctx, observabilitySidecar))
	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Execution().Start(ctx))

	// Wait for the target pod to push data to the otel collector
	s.T().Log("Waiting one minute for the target pod to push data to the otel collector")
	time.Sleep(1 * time.Minute)

	// Verify that data has been pushed to Prometheus

	prometheusURL := fmt.Sprintf("%s/api/v1/query?query=up", prometheusEndpoint)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", prometheusURL, nil)
	s.Require().NoError(err)

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	s.Require().Equal(http.StatusOK, resp.StatusCode, "Prometheus API is not accessible")

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Require().Contains(string(body), "otel-collector", "otel-collector data source not found in Prometheus")

	s.T().Log("otel-collector data source is available in Prometheus")
}
